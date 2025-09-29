package models

// Poll represents a voting poll
type Poll struct {
	ID        string         `json:"id"`        // Unique poll ID
	Question  string         `json:"question"`  // The poll question
	Options   []string       `json:"options"`   // Available options
	Votes     map[string]int `json:"votes"`     // option -> count
	CreatedAt int64          `json:"createdAt"` // Unix timestamp for when poll was created
}
