package models

import "fmt"

const (
	LabMaxWorkers          = 32
	LabRequestsPerSecond   = 10
	LabDailyRequestLimit   = 100000
	LabWorkerCeiling       = 256
	LabRequestRateCeiling  = 1000
	LabDailyRequestCeiling = 10000000
)

type LabLimits struct {
	DailyRequests     int
	RequestsPerSecond int
	MaxWorkers        int
}

func (limits LabLimits) Normalized() LabLimits {
	if limits.DailyRequests < 1 || limits.DailyRequests > LabDailyRequestCeiling {
		limits.DailyRequests = LabDailyRequestLimit
	}
	if limits.RequestsPerSecond < 1 || limits.RequestsPerSecond > LabRequestRateCeiling {
		limits.RequestsPerSecond = LabRequestsPerSecond
	}
	if limits.MaxWorkers < 1 || limits.MaxWorkers > LabWorkerCeiling {
		limits.MaxWorkers = LabMaxWorkers
	}
	return limits
}

func (settings AppSettings) LabLimits() LabLimits {
	return (LabLimits{DailyRequests: settings.LabDailyRequestLimit, RequestsPerSecond: settings.LabRequestsPerSecond, MaxWorkers: settings.LabMaxWorkers}).Normalized()
}

func (limits LabLimits) Validate() error {
	if limits.DailyRequests < 1 || limits.DailyRequests > LabDailyRequestCeiling {
		return fmt.Errorf("每日请求预算必须在 1–%d 之间", LabDailyRequestCeiling)
	}
	if limits.RequestsPerSecond < 1 || limits.RequestsPerSecond > LabRequestRateCeiling {
		return fmt.Errorf("每秒请求数必须在 1–%d 之间", LabRequestRateCeiling)
	}
	if limits.MaxWorkers < 1 || limits.MaxWorkers > LabWorkerCeiling {
		return fmt.Errorf("并发上限必须在 1–%d 之间", LabWorkerCeiling)
	}
	return nil
}

type LabDomainVerification struct {
	Domain     string `json:"domain"`
	Host       string `json:"host,omitempty"`
	SNI        string `json:"sni,omitempty"`
	RecordName string `json:"record_name"`
	Token      string `json:"token"`
	OwnerID    string `json:"owner_id"`
	IssuedAt   int64  `json:"issued_at"`
	VerifiedAt int64  `json:"verified_at"`
	CheckedAt  int64  `json:"checked_at"`
	LastError  string `json:"last_error"`
}

type LabSecurityState struct {
	InstanceID    string                `json:"instance_id"`
	Verification  LabDomainVerification `json:"verification"`
	BudgetDay     string                `json:"budget_day"`
	BudgetUsed    int                   `json:"budget_used"`
	LastStartedAt int64                 `json:"last_started_at"`
}

type LabDNSAvailability struct {
	Available   bool   `json:"available"`
	AccountName string `json:"account_name,omitempty"`
	Message     string `json:"message"`
}

type LabDNSProvisionResult struct {
	Verification  LabDomainVerification `json:"verification"`
	RecordCreated bool                  `json:"record_created"`
	Verified      bool                  `json:"verified"`
	Message       string                `json:"message"`
}

// LabIPSelectorSettings is the global configuration for the experimental
// direct-IP scanner and DNS updater. The secret key is stored encrypted.
type LabIPSelectorSettings struct {
	Limits       LabLimits `json:"-"`
	Host         string    `json:"host"`
	SNI          string    `json:"sni"`
	Path         string    `json:"path"`
	Statuses     string    `json:"statuses"`
	Timeout      int       `json:"timeout"`
	Workers      int       `json:"workers"`
	Top          int       `json:"top"`
	IPTargets    string    `json:"ip_targets"`
	Schedule     bool      `json:"schedule"`
	IntervalMins int       `json:"interval_minutes"`
	UpdateDNS    bool      `json:"update_dns"`
	Zone         string    `json:"zone"`
	ZoneID       string    `json:"zone_id"`
	Record       string    `json:"record"`
	TTL          int       `json:"ttl"`
	Endpoint     string    `json:"endpoint"`
	AccessKey    string    `json:"access_key"`
	SecretKey    string    `json:"secret_key,omitempty"`
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

type LabProbeSnapshot struct {
	Host     string `json:"host"`
	SNI      string `json:"sni"`
	Path     string `json:"path"`
	Statuses string `json:"statuses"`
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
	Diagnostics []LabIPScanResult    `json:"diagnostics,omitempty"`
	Rejected    int                  `json:"rejected,omitempty"`
	Probe       *LabProbeSnapshot    `json:"probe,omitempty"`
	Segments    []LabIPSegmentResult `json:"segments,omitempty"`
	DNSUpdated  bool                 `json:"dns_updated"`
}
