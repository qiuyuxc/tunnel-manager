package models

// OIDCProvider is an administrator-managed login provider. ClientSecret is
// encrypted at rest and is never included in a public or admin projection.
type OIDCProvider struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	Scopes       string `json:"scopes"`
	Enabled      bool   `json:"enabled"`
	Order        int    `json:"order"`
	Revision     uint64 `json:"revision"`
}

type OIDCIdentity struct {
	ProviderID string `json:"provider_id"`
	Issuer     string `json:"issuer"`
	Subject    string `json:"subject"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	CreatedAt  int64  `json:"created_at"`
}

// OIDCState participates in the same transaction as users, so provisioning
// an account and linking its external identity either both commit or neither.
type OIDCState struct {
	Providers  []OIDCProvider `json:"providers"`
	Identities []OIDCIdentity `json:"identities"`
}

type OIDCProviderView struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Issuer          string `json:"issuer"`
	ClientID        string `json:"client_id"`
	RedirectURI     string `json:"redirect_uri"`
	Scopes          string `json:"scopes"`
	Enabled         bool   `json:"enabled"`
	Order           int    `json:"order"`
	HasClientSecret bool   `json:"has_client_secret"`
}

type OIDCPublicProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
