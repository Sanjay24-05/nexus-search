package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type SearchResult struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type SearchResponse struct {
	Results     []SearchResult `json:"results"`
	TimeTakenMs int64          `json:"time_taken_ms"`
}

// Orchestrator handles parallel search requests
func Orchestrator(ctx context.Context, client *mongo.Client, userID string, query string, enableWeb, enableWiki, enableDDG, enablePKB bool) ([]SearchResult, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan []SearchResult, 3)
	errChan := make(chan error, 3)

	// ctx is passed in now
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel() => caller handles timeout or we add timeout here

	// We should enforce a timeout for search
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if enableWeb {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := searchSerpApi(ctx, query)
			if err != nil {
				errChan <- fmt.Errorf("SerpApi: %v", err)
				return
			}
			resultsChan <- res
		}()
	}

	if enableDDG {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := searchDuckDuckGo(ctx, query)
			if err != nil {
				errChan <- fmt.Errorf("DDG: %v", err)
				return
			}
			resultsChan <- res
		}()
	}

	if enableWiki {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := searchWikipedia(ctx, query)
			if err != nil {
				errChan <- fmt.Errorf("Wiki: %v", err)
				return
			}
			resultsChan <- res
		}()
	}

	if enablePKB && userID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := searchPKB(ctx, client, userID, query)
			if err != nil {
				errChan <- fmt.Errorf("PKB: %v", err)
				return
			}
			resultsChan <- res
		}()
	}

	// Wait in a separate goroutine to close channel
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errChan)
	}()

	var allResults []SearchResult
	for res := range resultsChan {
		allResults = append(allResults, res...)
	}

	// Identify if any errors occurred (logging sake)
	go func() {
		for err := range errChan {
			fmt.Printf("Search Error: %v\n", err)
		}
	}()

	return allResults, nil
}

// searchSerpApi uses the real SerpApi
func searchSerpApi(ctx context.Context, query string) ([]SearchResult, error) {
	apiKey := os.Getenv("SERPAPI_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("SERPAPI_KEY not set")
	}

	urlStr := fmt.Sprintf("https://serpapi.com/search.json?q=%s&api_key=%s", url.QueryEscape(query), apiKey)

	req, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[SerpApi] Error Status: %s\n", resp.Status)
		return nil, fmt.Errorf("SerpApi failed with status: %s", resp.Status)
	}

	// Minimal struct for parsing
	var result struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("[SerpApi] Decode Error: %v\n", err)
		return nil, err
	}

	if result.Error != "" {
		fmt.Printf("[SerpApi] API returned error: %s\n", result.Error)
		return nil, fmt.Errorf("SerpApi Error: %s", result.Error)
	}

	fmt.Printf("[SerpApi] Found %d results\n", len(result.OrganicResults))

	var results []SearchResult
	// Limit to top 3
	for i, r := range result.OrganicResults {
		if i >= 3 {
			break
		}
		results = append(results, SearchResult{
			Source:  "Google (SerpApi)",
			Title:   r.Title,
			Snippet: r.Snippet,
			URL:     r.Link,
		})
	}
	return results, nil
}

func searchDuckDuckGo(ctx context.Context, query string) ([]SearchResult, error) {
	fmt.Printf("[DDG] Starting search for: %s\n", query)
	// DDG Instant Answer API (Free)
	urlStr := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json", url.QueryEscape(query))

	req, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		return nil, fmt.Errorf("DDG failed with status: %s", resp.Status)
	}

	var result struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		Heading       string `json:"Heading"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var results []SearchResult

	// Add Abstract if exists
	if result.AbstractText != "" {
		results = append(results, SearchResult{
			Source:  "DuckDuckGo (Instant)",
			Title:   result.Heading,
			Snippet: result.AbstractText,
			URL:     result.AbstractURL,
		})
	}

	// Add Related Topics (Top 2)
	for i, r := range result.RelatedTopics {
		if i >= 2 {
			break
		}
		if r.Text == "" {
			continue
		}

		// Extract a better title
		// If FirstURL contains text after last slash, use it?
		// Or split text by " - " or first sentence?
		title := "DuckDuckGo Result"
		if r.FirstURL != "" {
			// Try to extract slug
			u, err := url.Parse(r.FirstURL)
			if err == nil {
				// /Computer -> Computer
				slug := u.Path
				if len(slug) > 1 {
					title = slug[1:] // remove slash
					// Replace underscores with spaces
					// (Doing simple replace here)
				}
			}
		}

		results = append(results, SearchResult{
			Source:  "DuckDuckGo",
			Title:   title,
			Snippet: r.Text,
			URL:     r.FirstURL,
		})
	}

	// Fallback if empty (DDG API is strict)
	if len(results) == 0 {
		// Return a helpful link if API return nothing (common for general queries vs facts)
		results = append(results, SearchResult{
			Source:  "DuckDuckGo",
			Title:   "Search on DuckDuckGo",
			Snippet: "Instant Answer API returned no direct summary. Click to view full results.",
			URL:     "https://duckduckgo.com/?q=" + url.QueryEscape(query),
		})
	}

	return results, nil
}

func searchWikipedia(ctx context.Context, query string) ([]SearchResult, error) {
	// Use action=query&list=search for meaningful snippets
	urlStr := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json&srlimit=3", url.QueryEscape(query))

	req, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	req.Header.Set("User-Agent", "NexusSearch/1.0 (sanjay@example.com)")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
				PageID  int    `json:"pageid"`
			} `json:"search"`
		} `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, item := range result.Query.Search {
		// Snippet comes with HTML spans (searchmatch), we can strip them or keep them for frontend.
		// For now, let's keep them or simple string strip if needed.
		// Frontend displays snippets as text, so html tags might show up.
		// Let's strip simple tags.
		cleanSnippet := stripHTML(item.Snippet)

		results = append(results, SearchResult{
			Source:  "Wikipedia",
			Title:   item.Title,
			Snippet: cleanSnippet,
			URL:     fmt.Sprintf("https://en.wikipedia.org/?curid=%d", item.PageID),
		})
	}

	return results, nil
}

// Helper to strip HTML tags from snippets
func stripHTML(s string) string {
	output := ""
	inTag := false
	for _, char := range s {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			output += string(char)
		}
	}
	return output
}
