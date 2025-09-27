package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// Global Firestore client
var client *firestore.Client

// Poll represents a single poll
type Poll struct {
	ID       string         `json:"id,omitempty"`
	Question string         `json:"question"`
	Options  []string       `json:"options"`
	Votes    map[string]int `json:"votes"`
}

func main() {
	// Read service account JSON from environment variable
	credsJSON := os.Getenv("GOOGLE_CREDENTIALS")
	if credsJSON == "" {
		log.Fatal("🔥 GOOGLE_CREDENTIALS environment variable not set")
	}

	// Initialize Firebase app
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON([]byte(credsJSON)))
	if err != nil {
		log.Fatalf("🔥 Failed to initialize Firebase: %v", err)
	}

	// Initialize Firestore client
	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("🔥 Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Setup routes using ServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/createPoll", createPollHandler)
	mux.HandleFunc("/getPolls", getPollsHandler)
	mux.HandleFunc("/vote", voteHandler)

	// Wrap mux with global CORS
	handler := cors(mux)

	// Port for Render or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// CORS middleware for http.Handler
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// createPollHandler creates a new poll
func createPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"only POST method allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var poll Poll
	if err := json.NewDecoder(r.Body).Decode(&poll); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if poll.Question == "" || len(poll.Options) < 2 {
		http.Error(w, `{"error":"question and at least 2 options required"}`, http.StatusBadRequest)
		return
	}

	// Initialize votes map
	poll.Votes = make(map[string]int)
	for _, opt := range poll.Options {
		poll.Votes[opt] = 0
	}

	ctx := context.Background()
	docRef, _, err := client.Collection("polls").Add(ctx, map[string]interface{}{
		"question": poll.Question,
		"options":  poll.Options,
		"votes":    poll.Votes,
	})
	if err != nil {
		log.Printf("🔥 Firestore add error: %v", err)
		http.Error(w, `{"error":"failed to create poll"}`, http.StatusInternalServerError)
		return
	}

	// Fetch the full poll back from Firestore
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		log.Printf("🔥 Firestore fetch error: %v", err)
		http.Error(w, `{"error":"failed to fetch poll"}`, http.StatusInternalServerError)
		return
	}

	var fullPoll Poll
	if err := docSnap.DataTo(&fullPoll); err != nil {
		log.Printf("🔥 Firestore DataTo error: %v", err)
		http.Error(w, `{"error":"failed to decode poll data"}`, http.StatusInternalServerError)
		return
	}

	// Make sure ID is set
	fullPoll.ID = docRef.ID

	// Respond with the full poll object
	if err := json.NewEncoder(w).Encode(fullPoll); err != nil {
		log.Printf("🔥 JSON encode error: %v", err)
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}


// getPollsHandler returns all polls
func getPollsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	iter := client.Collection("polls").Documents(ctx)

	var polls []Poll
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var p Poll
		doc.DataTo(&p)
		p.ID = doc.Ref.ID
		polls = append(polls, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(polls)
}

// voteHandler allows voting for a poll option
func voteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PollID string `json:"poll_id"`
		Option string `json:"option"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	docRef := client.Collection("polls").Doc(req.PollID)

	// Transaction to safely increment vote
	err := client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		votesInterface := doc.Data()["votes"].(map[string]interface{})
		votes := make(map[string]int)
		for k, v := range votesInterface {
			switch val := v.(type) {
			case int64:
				votes[k] = int(val)
			case float64:
				votes[k] = int(val)
			default:
				votes[k] = 0
			}
		}

		if _, ok := votes[req.Option]; !ok {
			return fmt.Errorf("option does not exist")
		}

		votes[req.Option]++
		return tx.Set(docRef, map[string]interface{}{"votes": votes}, firestore.MergeAll)
	})

	if err != nil {
		http.Error(w, "Failed to vote: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "vote recorded successfully"})
}
