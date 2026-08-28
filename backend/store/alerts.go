package store

import (
	"log"
	"time"

	"tunnel-manager/models"
)

// UpdateTargetLastState persists the most recent probe state of a target so
// alert edge detection survives restarts.
func (s *Store) UpdateTargetLastState(monitorID, targetID, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.config.Monitors {
		if s.config.Monitors[i].ID != monitorID {
			continue
		}
		for j := range s.config.Monitors[i].Targets {
			if s.config.Monitors[i].Targets[j].ID == targetID {
				previous := s.config.Monitors[i].Targets[j].LastState
				s.config.Monitors[i].Targets[j].LastState = state
				if err := s.saveLocked(); err != nil {
					s.config.Monitors[i].Targets[j].LastState = previous
					return err
				}
				return nil
			}
		}
	}
	return ErrMonitorNotFound
}

// AddAlertLog records one alert delivery attempt.
func (s *Store) AddAlertLog(entry models.AlertLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.CreatedAt = time.Now().Unix()
	s.alertLogs = append(s.alertLogs, entry)
	// Cap the log to the most recent 1000 entries across all monitors.
	if len(s.alertLogs) > 1000 {
		s.alertLogs = s.alertLogs[len(s.alertLogs)-1000:]
	}
	if err := s.saveLocked(); err != nil {
		log.Printf("save alert log: %v", err)
	}
}

// ListAlertLogs returns the newest alerts of one monitor (max 100).
func (s *Store) ListAlertLogs(monitorID string, limit int) []models.AlertLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	out := make([]models.AlertLog, 0, limit)
	for i := len(s.alertLogs) - 1; i >= 0 && len(out) < limit; i-- {
		if s.alertLogs[i].MonitorID == monitorID {
			out = append(out, s.alertLogs[i])
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Application encryption key (admin panel; environment variable wins)

// GetEncryptionKeyRaw returns the encryption key stored in the database, if any.
func (s *Store) GetEncryptionKeyRaw() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.encryptionKeyRaw
}

// SetEncryptionKeyRaw stores the encryption key (hex encoded) in the database.
func (s *Store) SetEncryptionKeyRaw(encoded string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.encryptionKeyRaw
	s.encryptionKeyRaw = encoded
	if err := s.saveLocked(); err != nil {
		s.encryptionKeyRaw = previous
		return err
	}
	return nil
}
