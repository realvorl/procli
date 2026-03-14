package net

type Message struct {
	Type string `json:"type"`
}

type Join struct {
	Type    string `json:"type"`
	Session string `json:"session"`
	Name    string `json:"name"`
}

type Welcome struct {
	Type     string `json:"type"`
	Session  string `json:"session"`
	ClientID string `json:"client_id"`
	Role     string `json:"role"`
}

type ClientInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Voted bool   `json:"voted"`
	Vote  string `json:"vote,omitempty"`
}

type State struct {
	Type         string         `json:"type"`
	Session      string         `json:"session"`
	Phase        string         `json:"phase"`
	StoryTitle   string         `json:"story_title"`
	StoryURL     string         `json:"story_url,omitempty"`
	Clients      []ClientInfo   `json:"clients"`
	CurrentRound CurrentRound   `json:"current_round"`
	History      []HistoryEntry `json:"history"`
}

type CurrentRound struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	URL           string `json:"url,omitempty"`
	Reveal        bool   `json:"reveal"`
	FinalEstimate string `json:"final_estimate,omitempty"`
}

type HistoryEntry struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	URL           string `json:"url,omitempty"`
	Phase         string `json:"phase"`
	FinalEstimate string `json:"final_estimate,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}

type VoteMessage struct {
	Type  string `json:"type"`
	Vote  string `json:"vote"`
	Round int    `json:"round,omitempty"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Session string `json:"session"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
