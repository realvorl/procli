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
}

type State struct {
	Type    string       `json:"type"`
	Session string       `json:"session"`
	Phase   string       `json:"phase"`
	Clients []ClientInfo `json:"clients"`
}
