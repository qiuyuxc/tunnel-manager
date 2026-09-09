package store

import "tunnel-manager/models"

const maxLabRuns = 20

// GetLabSettings returns the experimental IP selector configuration.
func (s *Store) GetLabSettings() models.LabIPSelectorSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.labSettings
}

// SetLabSettings persists the experimental IP selector configuration.
func (s *Store) SetLabSettings(settings models.LabIPSelectorSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.labSettings
	s.labSettings = settings
	if err := s.saveLocked(); err != nil {
		s.labSettings = previous
		return err
	}
	return nil
}

// GetLabRuns returns recent experimental IP selector runs.
func (s *Store) GetLabRuns(limit int) []models.LabIPSelectorRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.labRuns) {
		limit = len(s.labRuns)
	}
	runs := make([]models.LabIPSelectorRun, limit)
	copy(runs, s.labRuns[len(s.labRuns)-limit:])
	for i, j := 0, len(runs)-1; i < j; i, j = i+1, j-1 {
		runs[i], runs[j] = runs[j], runs[i]
	}
	return runs
}

// AppendLabRun persists a completed run and keeps a bounded history.
func (s *Store) AppendLabRun(run models.LabIPSelectorRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]models.LabIPSelectorRun(nil), s.labRuns...)
	s.labRuns = append(s.labRuns, run)
	if len(s.labRuns) > maxLabRuns {
		s.labRuns = append([]models.LabIPSelectorRun(nil), s.labRuns[len(s.labRuns)-maxLabRuns:]...)
	}
	if err := s.saveLocked(); err != nil {
		s.labRuns = previous
		return err
	}
	return nil
}
