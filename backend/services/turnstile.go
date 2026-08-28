package services

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TurnstileSiteVerifyURL is the canonical Cloudflare siteverify endpoint.
// It is a package variable so tests can point at a local server.
var TurnstileSiteVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// TurnstileVerifyResult is the subset of the siteverify response the
// application needs.
type TurnstileVerifyResult struct {
	Success bool   `json:"success"`
	Action  string `json:"action,omitempty"`
}

// VerifyTurnstile redeems a single-use cf-turnstile-response token against
// the canonical Cloudflare siteverify endpoint. When expectedAction is
// non-empty the returned widget action must match it. Any transport or
// decoding failure fails closed (ok=false) so an unavailable verifier never
// silently grants access.
func VerifyTurnstile(secret, token, remoteIP, expectedAction string) (bool, error) {
	if secret == "" || token == "" {
		return false, nil
	}
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequest(http.MethodPost, TurnstileSiteVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var result TurnstileVerifyResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if !result.Success {
		return false, nil
	}
	if expectedAction != "" && result.Action != "" && result.Action != expectedAction {
		return false, nil
	}
	return true, nil
}
