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
	ctx := context.Background()

	// ✅ Initialize Firebase app
	credsStr := os.Getenv("GOOGLE_CREDENTIALS")
	if credsStr == "" {
		log.Fatal("🔥 GOOGLE_CREDENTIALS environment variable not set")
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON([]byte(credsStr)))
	if err != nil {
		log.Fatalf("🔥 Failed to initialize Firebase app: %v", err)
	}

	// ✅ Assign global Firestore client (no shadowing)
	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("🔥 Failed to connect Firestore: %v", err)
	}
	defer client.Close()

	// ✅ Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("/polls", createPollHandler)
	mux.HandleFunc("/getPolls", getPollsHandler)
	mux.HandleFunc("/vote", voteHandler)
	mux.HandleFunc("/deletePoll", deletePollHandler)

	// ✅ Wrap routes with CORS middleware
	handler := cors(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// ✅ Universal CORS Middleware
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173") // or "*" for all
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// ✅ Create Poll Handler
func createPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

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

	// Initialize votes
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

	poll.ID = docRef.ID
	json.NewEncoder(w).Encode(poll)
}

// ✅ Get All Polls
func getPollsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	json.NewEncoder(w).Encode(polls)
}

// ✅ Vote Handler
func voteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
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

	json.NewEncoder(w).Encode(map[string]string{"message": "Vote recorded successfully"})
}

// ✅ Delete Poll
func deletePollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PollID string `json:"poll_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	docRef := client.Collection("polls").Doc(req.PollID)
	_, err := docRef.Get(ctx)
	if err != nil {
		http.Error(w, "Poll not found", http.StatusNotFound)
		return
	}

	_, err = docRef.Delete(ctx)
	if err != nil {
		http.Error(w, "Failed to delete poll", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Poll deleted successfully"})
}
