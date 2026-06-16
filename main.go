package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// TransactionRequest represents incoming data from your iPhone
type TransactionRequest struct {
	Amount     float64 `json:"amount"`
	Merchant   string  `json:"merchant"`
	Location   string  `json:"location"`
	SplitInput string  `json:"split_input"`
}

// TransactionRecord represents the final enriched JSON saved to the cloud
type TransactionRecord struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	Amount          float64   `json:"amount"`
	Merchant        string    `json:"merchant"`
	Location        string    `json:"location"`
	SplitInput      string    `json:"split_input"`
	SplitPeople     []string  `json:"split_people"`
	AmountPerPerson float64   `json:"amount_per_person"`
}

var bucketName string
var apiKey string
var storageClient *storage.Client

func main() {
	// 1. Load configuration from environment variables
	bucketName = os.Getenv("BUCKET_NAME")
	apiKey = os.Getenv("API_KEY")
	port := os.Getenv("PORT")

	if bucketName == "" || apiKey == "" {
		log.Fatal("Missing required environment variables: BUCKET_NAME or API_KEY")
	}
	if port == "" {
		port = "8080"
	}

	// 2. Initialize Google Cloud Storage Client
	ctx := context.Background()
	var err error
	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Google Cloud Storage client: %v", err)
	}
	defer storageClient.Close()

	// 3. Define HTTP routes
	http.HandleFunc("/transaction", handleTransaction)

	log.Printf("Server listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple Security Check
	authHeader := r.Header.Get("X-API-Key")
	if authHeader != apiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req TransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Process the splitting logic
	people, amountPerPerson := parseSplit(req.SplitInput, req.Amount)

	now := time.Now()
	record := TransactionRecord{
		ID:              fmt.Sprintf("tx-%d", now.UnixNano()),
		Timestamp:       now,
		Amount:          req.Amount,
		Merchant:        req.Merchant,
		Location:        req.Location,
		SplitInput:      req.SplitInput,
		SplitPeople:     people,
		AmountPerPerson: amountPerPerson,
	}

	// Save to Google Cloud Storage
	err = saveToGCS(r.Context(), record)
	if err != nil {
		log.Printf("Error saving to GCS: %v", err)
		http.Error(w, "Failed to save transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"success","message":"Transaction logged successfully"}`))
}

// parseSplit handles the text logic from your iPhone input
func parseSplit(input string, total float64) ([]string, float64) {
	input = strings.ToLower(strings.TrimSpace(input))

	// Default case: if empty, just "b", or "ben", it's 100% you
	if input == "" || input == "b" || input == "ben" {
		return []string{"Ben"}, total
	}

	// Split by commas for multiple people
	parts := strings.Split(input, ",")
	var people []string

	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "b", "ben":
			people = append(people, "Ben")
		case "dc":
			people = append(people, "Dan Chan")
		case "dg":
			people = append(people, "Dan Green")
		case "l", "liam":
			people = append(people, "Liam")
		default:
			// If you type a new name/shorthand on the fly, it defaults to capitalizing it
			people = append(people, strings.Title(p))
		}
	}

	// Fallback check
	if len(people) == 0 {
		return []string{"Ben"}, total
	}

	// Calculate even split
	amountPerPerson := total / float64(len(people))
	return people, amountPerPerson
}

// saveToGCS uploads the JSON file into your free bucket
func saveToGCS(ctx context.Context, record TransactionRecord) error {
	fileName := fmt.Sprintf("%s.json", record.ID)
	bucket := storageClient.Bucket(bucketName)
	obj := bucket.Object(fileName)

	wc := obj.NewWriter(ctx)
	wc.ContentType = "application/json"

	err := json.NewEncoder(wc).Encode(record)
	if err != nil {
		wc.Close()
		return err
	}

	return wc.Close()
}