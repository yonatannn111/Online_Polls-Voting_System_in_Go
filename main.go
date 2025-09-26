package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
//  eyJ0eXBlIjoic2VydmljZV9hY2NvdW50IiwicHJvamVjdF9pZCI6Im9ubGluZS1wb2xsLXZvdGluZy1pbi1nbyIsInByaXZhdGVfa2V5X2lkIjoiY2YwNTRkNTU5MmJlNjQ3NWIxZjEwOWZkMTBjZTlhZGVhODRlN2JjNCIsInByaXZhdGVfa2V5IjoiLS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tXG5NSUlFdndJQkFEQU5CZ2txaGtpRzl3MEJBUUVGQUFTQ0JLa3dnZ1NsQWdFQUFvSUJBUURKRVQ3V1doRnE1V00xXG5yK1d3UEk3aXo2NmlXWUVOSnNjRzBvVUF0Rk02TThmOVRoNVpONjE5L0J1dWRuRlJ6K0lONjFtWXd5NUpwMzY3XG5KRk41cDhFV3lRenF6eDIyYUZpVzVrd05XMVQ1dVI1c2JhVUo4dEJ5UzBld2xtS0pSUW1lWHJBWWNqOGRmcTRaXG5Ta0pEL3liZUhkNTB5Uit3c3ZWWFVOZlpGb0JONThOWEd3RjFmUllLRlBXZ01tOU1sa3N2cHVsbU5tLzdCa3JCXG4rTmgvTTdPVFo5Wm1BNE0wcENTZXlhenRsVDdFUDhyYnlIMmVrSElXWExwbXVQVmlQdXhCWEp6OXgxYkhZZWg3XG5LWTlsayt2N2RPSHJ0WVBkL3VjRWFKU2FVcHZld0xRT09BSUs2aXB3czhuWFZBUldTVHRoUDExM1lVQmdyU3luXG51MGRPSGxYSkFnTUJBQUVDZ2dFQUNMRVBKMUx6Wjh5TXNsaHROR1lad3lJNUtWbDNJNnRtZFJ3S2xkSTF1dEZjXG5OczMycitLaFJFM3VvR2NNVlA0SWhwT0M1d0NnOHB3ajlWRUhycjJhc2pKMHRYM0lpZ2NPdFU4MVFHcVBUTkE4XG55WEVmN2tNcDBaY0JmVmV5NEw0MFFUMVVuV25xNG9xRVdpR0VwYzNVejBzNVA1RW1Kb3hGNE1WazMwU0N2VnRRXG55UkgzSVpJSGxIOXVDK0dNOWp3Wnlsc2JjdVBrV3JYZVJXT3VQSzE5c0dEc1RQNVJSMHJjcFJ3cC9ldTF4RG1qXG5BaEg1ZjQ2bVA0QU5neHpyd3JrclRjY2RKYVAydHZyQnJZS3pqbVp6MVF2SmRlT3IvbFJtN0FWQ0gvUVBTUHBnXG5pVzJLMTEzN1lHc0lrYWRtc29YZEpqbGQyYW5Fb0ozZGxJNlYyamdLVlFLQmdRRHZGZ1RTT29UdC9MQVV1bW10XG5lOGxyMElXNTdLb2p0anZ4a3J6alczVHF4NXZYWC9XUlc0bWZSeWNHZE5uTHdURGcybXBrbnJEMUJlWXM5bFhWXG5UVStDaDZKeGIvK1ZEd3VScnJQTmlVcWFseG9qU0JwS1RhNExWWW11VXVkUTJBemZybWVjYVJxVkVKaXgrdFluXG5SRU16a0lIbHFOaXdGZndFNnlZSFRyanJKd0tCZ1FEWFNyQU9Nd0M2Tm9JeWp1UVBvMWQ4N0tYeWVzaEcrU0xRXG41SmtuKzJDWHBtdmcyWW9yaVFkZWxESTZrZUZYQVU0T29RcWtCVjh3bW12TVVTd3lHS1YrUGltWVg2OVRrbTZ0XG5oZ2hTS1hUREJqcmtGQy81VUtvR05NWHlNQmF3OCsvM0YyejVCZk1lbWYzb0xYWEtHZFdnRWUrYzhZbmtrUE5JXG5sejkxUFU0Tmp3S0JnUUR1MGRmUEI1VnhCRS8rNUpaYkxLTnVoc1NOaTlJSUNpaW1qaVVRRm5NYmNuaEJFeUdCXG5LU2EzYTZPWDEzRVhEc3Q4VDdDbkFiMVJnNnNBanEvK2VWTksxNkYwSHFQMmlTak5STzFtQ2hYemhhd2VRZy9BXG4yUWRaV3dCRW1adG1MZW51SlpCcHRMTlE0MXNqcmFQdFpVcWJYMlhodWw5NHhQMFJETExYNmRMZFVRS0JnUUNpXG5WV2E2emlwV1BwT2RtN0RMT2RiV0UzcHRnN2RRRExyNzErTEVDditpV1pJdVVObW1TZ1NNaENIN2w5UFp6dG9VXG5uY2x3TTd5NjRUVTNNbDJveUh6QTNBNXhIblVOQnZUOVVuc2p1SzZaL3pDWW1jQXl0V2YrbGZ4THlZYlNscHp5XG5LMiszdFl6RUhra2RzR21Jb2tJNkdFd2NndVArdkcwMDV4YXFTRGQ2Y1FLQmdRQ21QUXVkSkhQaTgwVXZ1NGdsXG55L1JhN1JmNzV1YjdkR3RmUUZlM204MzZXbnpnUHRNUjR2MmQ2cTRIQjlkS2JPbDFJYnBQbk1PUU1xU2hlS2RiXG5Oa0RybGpyejNoNThOWCtqQ0JmRWI0eFFRbHRMVnlKdGp2U3pzQ0poV1hDbC9xZjAvS05PMFRUTnJNYzNyZjlhXG53aVpZNGhPSE5ldnlJVjZ5Qlg2b1NGWGxxUT09XG4tLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tXG4iLCJjbGllbnRfZW1haWwiOiJmaXJlYmFzZS1hZG1pbnNkay1mYnN2Y0BvbmxpbmUtcG9sbC12b3RpbmctaW4tZ28uaWFtLmdzZXJ2aWNlYWNjb3VudC5jb20iLCJjbGllbnRfaWQiOiIxMDY0NjkxOTUwODI2Njc1Nzk0NTQiLCJhdXRoX3VyaSI6Imh0dHBzOi8vYWNjb3VudHMuZ29vZ2xlLmNvbS9vL29hdXRoMi9hdXRoIiwidG9rZW5fdXJpIjoiaHR0cHM6Ly9vYXV0aDIuZ29vZ2xlYXBpcy5jb20vdG9rZW4iLCJhdXRoX3Byb3ZpZGVyX3g1MDlfY2VydF91cmwiOiJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9vYXV0aDIvdjEvY2VydHMiLCJjbGllbnRfeDUwOV9jZXJ0X3VybCI6Imh0dHBzOi8vd3d3Lmdvb2dsZWFwaXMuY29tL3JvYm90L3YxL21ldGFkYXRhL3g1MDkvZmlyZWJhc2UtYWRtaW5zZGstZmJzdmMlNDBvbmxpbmUtcG9sbC12b3RpbmctaW4tZ28uaWFtLmdzZXJ2aWNlYWNjb3VudC5jb20iLCJ1bml2ZXJzZV9kb21haW4iOiJnb29nbGVhcGlzLmNvbSJ9DQogIA==
func main() {
	// 1️⃣ Decode Firebase credentials from environment variable
	credsBase64 := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_BASE64")
	if credsBase64 == "" {
		log.Fatal("⚠️ GOOGLE_APPLICATION_CREDENTIALS_BASE64 not set")
	}
	creds, err := base64.StdEncoding.DecodeString(credsBase64)
	if err != nil {
		log.Fatalf("Failed to decode Firebase credentials: %v", err)
	}

	// 2️⃣ Write the decoded JSON to a local file
	err = ioutil.WriteFile("serviceAccountKey.json", creds, 0644)
	if err != nil {
		log.Fatalf("Failed to write serviceAccountKey.json: %v", err)
	}

	// 3️⃣ Initialize Firebase app
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile("serviceAccountKey.json"))
	if err != nil {
		log.Fatalf("🔥 Failed to initialize Firebase: %v", err)
	}

	// 4️⃣ Initialize Firestore client
	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("🔥 Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// 5️⃣ Routes with CORS middleware
	http.HandleFunc("/createPoll", cors(createPollHandler))
	http.HandleFunc("/getPolls", cors(getPollsHandler))
	http.HandleFunc("/vote", cors(voteHandler))

	// 6️⃣ Port for Railway or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Simple CORS middleware
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	}
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

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "✅ Vote recorded successfully")
}
