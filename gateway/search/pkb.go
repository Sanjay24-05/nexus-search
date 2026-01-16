package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Helper to get embedding from Python Worker
func getEmbedding(query string) ([]float32, error) {
	workerURL := "http://127.0.0.1:5000/embed"
	payload := map[string]string{"text": query}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", workerURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("worker embedding failed: %s", resp.Status)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

func searchPKB(ctx context.Context, client *mongo.Client, userID string, query string) ([]SearchResult, error) {
	// 1. Get Query Vector
	vector, err := getEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("embedding gen failed: %v", err)
	}

	// 2. Vector Search in MongoDB
	collection := client.Database("nexus_search").Collection("docs")

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		fmt.Printf("[PKB] Invalid UserID: %v\n", err)
		return nil, fmt.Errorf("invalid user id: %v", err)
	}

	fmt.Printf("[PKB] Searching for UserOID: %s, Query: %s\n", userOID.Hex(), query)
	fmt.Printf("[PKB] Query Vector Length: %d\n", len(vector))

	// Vector Search Pipeline
	pipeline := []bson.M{
		{
			"$vectorSearch": bson.M{
				"index":         "vector_index",
				"path":          "embedding",
				"queryVector":   vector,
				"numCandidates": 100,
				"limit":         5,
				"filter":        bson.M{"user_id": userOID},
			},
		},
		{
			"$project": bson.M{
				"_id":      0,
				"filename": 1,
				"content":  1,
				"score":    bson.M{"$meta": "vectorSearchScore"},
			},
		},
	}

	// BUT wait, MongoDB Go driver might treat $vectorSearch differently or use aggregate
	// Yes, aggregate is correct.

	// Does 'user_id' in filter need ObjectId?
	// In storage.py we saved it as ObjectId.
	// We expect userID string here, so we might need to convert?
	// No, if we passed string in $match or $vectorSearch filter, Atlas Search usually handles types well
	// BUT standard Mongo strictness applies.
	// However, for simplicity let's assume the string match works or we need to pass ObjectId.
	// The previous code in `upload.go` returned `userID` as hex string.
	// We should probably convert to ObjectID for the filter if we saved as ObjectID.
	// Let's rely on string for now, but mark TODO if no results.
	// Actually, `storage.py` does `ObjectId(user_id)`.
	// So we MUST query with ObjectId if we want to match.
	// But we don't have the `primitive` package imported easily in this snippet?
	// Let's try to query without filter first? No, security risk.
	// We need to import "go.mongodb.org/mongo-driver/bson/primitive"

	// We'll fix imports later. This is logic.

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mongoResults []struct {
		Filename string  `bson:"filename"`
		Content  string  `bson:"content"`
		Score    float64 `bson:"score"`
	}
	if err := cursor.All(ctx, &mongoResults); err != nil {
		return nil, err
	}

	// 3. Convert to SearchResult
	var results []SearchResult
	for _, res := range mongoResults {
		results = append(results, SearchResult{
			Source:  "PKB (" + res.Filename + ")",
			Title:   res.Filename,
			Snippet: res.Content, // Maybe truncate?
			URL:     "#",         // No URL for local files
		})
	}

	return results, nil
}
