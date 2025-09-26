package models

// Poll represents a voting poll
type Poll struct {
	ID       string            `json:"id"`
	Question string            `json:"question"`
	Options  []string          `json:"options"`
	Votes    map[string]int    `json:"votes"` // option -> count
}
