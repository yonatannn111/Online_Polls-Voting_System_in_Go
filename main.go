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
	// Read service account JSON from env var
	credsStr := os.Getenv("GOOGLE_CREDENTIALS")
	if credsStr == "" {
		log.Fatal("🔥 GOOGLE_CREDENTIALS environment variable not set")
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON([]byte(credsStr)))
	if err != nil {
		log.Fatalf("🔥 Failed to initialize Firebase: %v", err)
	}

	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("🔥 Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/polls", createPollHandler)        // POST
	mux.HandleFunc("/getPolls", getPollsHandler)      // GET
	mux.HandleFunc("/vote", voteHandler)              // POST
	mux.HandleFunc("/deletePoll", deletePollHandler)  // DELETE

	// Wrap with global CORS
	handler := cors(mux)

	// Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// CORS middleware
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
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
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"only POST method allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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
	if err := json.NewEncoder(w).Encode(poll); err != nil {
		log.Printf("🔥 JSON encode error: %v", err)
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// getPollsHandler returns all polls
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

// voteHandler allows voting for a poll option
func voteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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

	json.NewEncoder(w).Encode(map[string]string{"message": "vote recorded successfully"})
}

// deletePollHandler deletes a poll
func deletePollHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE method allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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
