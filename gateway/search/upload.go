package search

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Helper to get UserID (in a real app, this might be cached or in JWT)
func GetUserID(client *mongo.Client, username string) (string, error) {
	collection := client.Database("nexus_search").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		ID interface{} `bson:"_id"`
	}
	// Depending on how _id is stored (ObjectId or string). Mgo driver handles it.
	err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&result)
	if err != nil {
		return "", err
	}

	// Convert ObjectID to hex string if needed, or just string
	if oid, ok := result.ID.(interface{ Hex() string }); ok {
		return oid.Hex(), nil
	}
	return result.ID.(string), nil
}

func UploadProxyHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get User ID
		username := r.Context().Value("user").(string)
		userID, err := GetUserID(client, username)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// 2. Read File from Request
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 3. Prepare Request to Python Worker
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add File
		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			http.Error(w, "Processing error", http.StatusInternalServerError)
			return
		}
		io.Copy(part, file)

		// Add User ID field
		writer.WriteField("user_id", userID)

		writer.Close()

		// 4. Send to Worker
		workerURL := "http://127.0.0.1:5000/process" // Use IPv4 loopback to avoid resolution issues
		req, err := http.NewRequest("POST", workerURL, body)
		if err != nil {
			http.Error(w, "Worker unreachable: "+err.Error(), http.StatusBadGateway)
			return
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		clientHTTP := &http.Client{Timeout: 120 * time.Second} // Increased to 120s for local embedding latency
		resp, err := clientHTTP.Do(req)
		if err != nil {
			// This is the error the user saw. Now we include the actual error message.
			http.Error(w, "Worker failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// 5. Return Worker Response
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
