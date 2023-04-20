package common

// For messages sent from server
type MessageEvent struct {
	Text    string `json:"text"`
	Channel string `json:"channel"`
	Author  string `json:"author"`
}
