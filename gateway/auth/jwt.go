package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username          string `bson:"username" json:"username"`
	Password          string `bson:"password,omitempty" json:"password"`
	TotalStorageBytes int64  `bson:"total_storage_bytes" json:"total_storage_bytes"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func RegisterHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		collection := client.Database("nexus_search").Collection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Check if user exists
		var existing User
		err = collection.FindOne(ctx, bson.M{"username": creds.Username}).Decode(&existing)
		if err == nil {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}

		newUser := User{
			Username:          creds.Username,
			Password:          string(hashedPassword),
			TotalStorageBytes: 0,
		}

		_, err = collection.InsertOne(ctx, newUser)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func LoginHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		collection := client.Database("nexus_search").Collection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user User
		err := collection.FindOne(ctx, bson.M{"username": creds.Username}).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Generate JWT
		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &Claims{
			Username: creds.Username,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		secret := []byte(os.Getenv("JWT_SECRET"))
		tokenString, err := token.SignedString(secret)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    tokenString,
			Expires:  expirationTime,
			HttpOnly: true, // Secure
		})

		// Also return it in JSON for convenience
		json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
	}
}

func GetProfileHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user from context (set by middleware)
		userVal := r.Context().Value("user")
		if userVal == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		username := userVal.(string)

		collection := client.Database("nexus_search").Collection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user User
		err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Hide password
		user.Password = ""

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}
