package models

// AlertLog records one monitor alert (state-change notification attempt).
type AlertLog struct {
	ID         int64  `json:"id"`
	MonitorID  string `json:"monitor_id"`
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name"`
	State      string `json:"state"`
	HTTPCode   int    `json:"http_code"`
	Error      string `json:"error"`
	Notified   bool   `json:"notified"`
	Detail     string `json:"detail"`
	CreatedAt  int64  `json:"created_at"`
}
