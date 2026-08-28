package models

// CFConnection is one authorized Cloudflare OAuth grant owned by a user.
// Tokens are stored encrypted; API responses use CFConnectionView.
type CFConnection struct {
	ID           string
	UserID       string
	Label        string
	AccountID    string
	AccountName  string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	Scope        string
	CreatedAt    int64
}

// CFConnectionView is the API-safe projection of a connection.
type CFConnectionView struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	Active      bool   `json:"active"`
	ExpiresAt   int64  `json:"expires_at"`
	CreatedAt   int64  `json:"created_at"`
}

// HasToken reports whether the connection carries OAuth credentials.
func (c *CFConnection) HasToken() bool {
	return c.AccessToken != ""
}
