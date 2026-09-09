package models

// LabIPSelectorSettings is the global configuration for the experimental
// direct-IP scanner and DNS updater. The secret key is stored encrypted.
type LabIPSelectorSettings struct {
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

// LabIPSelectorView omits the encrypted Huawei Cloud secret.
type LabIPSelectorView struct {
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
	HasSecretKey bool   `json:"has_secret_key"`
}

// LabIPScanResult is one scanner outcome.
type LabIPScanResult struct {
	IP        string `json:"ip"`
	Status    int    `json:"status,omitempty"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// LabIPSegmentResult summarizes one user-entered IP target segment.
type LabIPSegmentResult struct {
	Target  string `json:"target"`
	Scanned int    `json:"scanned"`
	Matched int    `json:"matched"`
}

// LabIPSelectorRun is a persisted execution record.
type LabIPSelectorRun struct {
	ID          string               `json:"id"`
	StartedAt   int64                `json:"started_at"`
	FinishedAt  int64                `json:"finished_at"`
	Success     bool                 `json:"success"`
	Error       string               `json:"error,omitempty"`
	Scanned     int                  `json:"scanned"`
	Matched     int                  `json:"matched"`
	SelectedIPs []string             `json:"selected_ips,omitempty"`
	Results     []LabIPScanResult    `json:"results,omitempty"`
	Segments    []LabIPSegmentResult `json:"segments,omitempty"`
	DNSUpdated  bool                 `json:"dns_updated"`
}
