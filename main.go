package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// Global Firestore client
var client *firestore.Client

func main() {
	// Initialize Firebase
	ctx := context.Background()
	sa := option.WithCredentialsFile("serviceAccountKey.json") // <- put your downloaded JSON here
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalf("🔥 Failed to initialize Firebase: %v", err)
	}

	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("🔥 Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Routes
	http.HandleFunc("/createPoll", createPollHandler)
	http.HandleFunc("/getPolls", getPollsHandler)
	http.HandleFunc("/vote", voteHandler)

	fmt.Println("🚀 Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Poll represents a single poll structure
type Poll struct {
	ID       string         `json:"id,omitempty"`
	Question string         `json:"question"`
	Options  []string       `json:"options"`
	Votes    map[string]int `json:"votes"`
}

// createPollHandler creates a new poll
func createPollHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	var poll Poll
	if err := json.NewDecoder(r.Body).Decode(&poll); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if poll.Question == "" || len(poll.Options) == 0 {
		http.Error(w, "Question and options are required", http.StatusBadRequest)
		return
	}

	// Initialize votes
	poll.Votes = make(map[string]int)
	for _, option := range poll.Options {
		poll.Votes[option] = 0
	}

	ctx := context.Background()
	docRef, _, err := client.Collection("polls").Add(ctx, poll)
	if err != nil {
		http.Error(w, "Failed to create poll", http.StatusInternalServerError)
		return
	}

	poll.ID = docRef.ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(poll)
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
		if err := doc.DataTo(&p); err != nil {
			continue
		}
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
	
		// Read votes map as map[string]int
		votesInterface := doc.Data()["votes"].(map[string]interface{})
		votes := make(map[string]int)
		for k, v := range votesInterface {
			// Firestore stores numbers as int64 or float64
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

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "✅ Vote recorded successfully")
}
