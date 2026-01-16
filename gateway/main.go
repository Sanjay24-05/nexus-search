package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"nexus-gateway/auth"
	"nexus-gateway/middleware"
	"nexus-gateway/search"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable is required")
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(context.Background()); err != nil {
			log.Printf("Failed to disconnect MongoDB: %v", err)
		}
	}()

	log.Println("Connected to MongoDB")

	// Setup Router
	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("/api/login", auth.LoginHandler(client))
	mux.HandleFunc("/api/register", auth.RegisterHandler(client))

	// Search Handler
	searchHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Query required", http.StatusBadRequest)
			return
		}

		// Toggles
		web := r.URL.Query().Get("web") == "true"
		wiki := r.URL.Query().Get("wiki") == "true"
		ddg := r.URL.Query().Get("ddg") == "true"
		pkb := r.URL.Query().Get("pkb") == "true"

		// Get User ID from Context (set by Auth middleware)
		userID, ok := r.Context().Value("user").(string)
		if !ok {
			// Should not happen if Auth middleware is there, but handle safety
			userID = ""
		}

		// We need the ObjectId for the PKB search, but 'user' in context is username!
		// Wait, Auth middleware sets "user" to username string.
		// But pkb.go expects userID hex string to convert to ObjectId.
		// We need to resolve username -> ObjectId here or inside Orchestrator.
		// Orchestrator takes "userID string".
		// pkb.go converts it to ObjectID.
		// So we need to look up the ID from the username.
		// We have search.getUserID (helper in upload.go) but it's private.
		// We should probably export it or duplicate logic?
		// Let's export 'GetUserID' in upload.go if possible or just make it public.
		// Or better yet, we can do the lookup in Orchestrator? No, Orchestrator should correspond to IDs.

		// Actually, let's fix Auth middleware to put the ID in context?
		// But Auth middleware only has username from JWT claims.
		// JWT claims *should* have ID.
		// If JWT only has username, we MUST look up ID.
		// Let's assume for now we look it up.
		// Since I can't easily export the function from `upload.go` without editing it again (I can),
		// I'll edit `upload.go` to export `GetUserID`.

		start := time.Now()

		// Resolve ID
		realID := ""
		if userID != "" {
			id, err := search.GetUserID(client, userID) // access exported function
			if err == nil {
				realID = id
			} else {
				fmt.Printf("Error resolving UserID for username %s: %v\n", userID, err)
			}
		} else {
			fmt.Println("UserID in context is empty")
		}

		fmt.Printf("Search Params - Query: %s, Web: %v, Wiki: %v, DDG: %v, PKB: %v, UserID: %s\n", query, web, wiki, ddg, pkb, realID)

		results, err := search.Orchestrator(r.Context(), client, realID, query, web, wiki, ddg, pkb)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := search.SearchResponse{
			Results:     results,
			TimeTakenMs: time.Since(start).Milliseconds(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	finalMux := http.NewServeMux()
	finalMux.HandleFunc("/api/login", auth.LoginHandler(client))
	finalMux.HandleFunc("/api/register", auth.RegisterHandler(client))

	// User Profile
	finalMux.Handle("/api/user", middleware.Auth(auth.GetProfileHandler(client), os.Getenv("JWT_SECRET")))

	// Search and Upload are protected
	finalMux.Handle("/api/search", middleware.Auth(searchHandler, os.Getenv("JWT_SECRET")))

	// Upload with content-length check
	finalMux.Handle("/api/upload", middleware.Auth(
		middleware.StorageCheck(search.UploadProxyHandler(client)), os.Getenv("JWT_SECRET")))

	// Global Middleware (CORS, RateLimit, Logging)
	globalHandler := middleware.Logging(middleware.CORS(middleware.RateLimit(finalMux)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Gateway running on port %s", port)
	if err := http.ListenAndServe(":"+port, globalHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
