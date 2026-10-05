package models

// SaveLabIPSelectorRequest is the body for the experimental lab settings.
type SaveLabIPSelectorRequest struct {
	Host         string `json:"host"`
	SNI          string `json:"sni"`
	Path         string `json:"path"`
	Statuses     string `json:"statuses"`
	Timeout      int    `json:"timeout"`
	Workers      int    `json:"workers"`
	Top          int    `json:"top"`
	IPTargets    string `json:"ip_targets"`
	Schedule     bool   `json:"schedule"`
	IntervalMins int    `json:"interval_minutes"`
	UpdateDNS    bool   `json:"update_dns"`
	Zone         string `json:"zone"`
	ZoneID       string `json:"zone_id"`
	Record       string `json:"record"`
	TTL          int    `json:"ttl"`
	Endpoint     string `json:"endpoint"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key,omitempty"`
}
