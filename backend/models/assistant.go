package models

import "encoding/json"

type AIConnection struct {
	Endpoint     string `json:"endpoint"`
	Model        string `json:"model"`
	KeyEncrypted string `json:"key_encrypted,omitempty"`
}

type AIBindingRequest struct {
	TunnelID   string `json:"tunnel_id"`
	ZoneID     string `json:"zone_id"`
	Hostname   string `json:"hostname"`
	ServiceURL string `json:"service_url"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AITask struct {
	ID           string          `json:"id"`
	Tool         string          `json:"tool"`
	Title        string          `json:"title"`
	Arguments    json.RawMessage `json:"arguments"`
	Status       string          `json:"status"`
	Result       string          `json:"result,omitempty"`
	ConnectionID string          `json:"connection_id,omitempty"`
}

type AIConversation struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	UpdatedAt int64       `json:"updated_at"`
	Messages  []AIMessage `json:"messages"`
	Tasks     []AITask    `json:"tasks"`
}

type AIUserState struct {
	Connection    AIConnection     `json:"connection"`
	Conversations []AIConversation `json:"conversations"`
}

type AIState struct {
	SharedEnabled bool                   `json:"shared_enabled"`
	Shared        AIConnection           `json:"shared"`
	Users         map[string]AIUserState `json:"users"`
}
