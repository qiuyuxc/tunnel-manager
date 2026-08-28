package store

import (
	"errors"
	"time"

	"tunnel-manager/models"
)

var (
	ErrConnectionNotFound = errors.New("cloudflare connection not found")
)

// ListCFConnections returns every connection owned by the user.
func (s *Store) ListCFConnections(userID string) []models.CFConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.CFConnection, 0)
	for i := range s.cfConns {
		if s.cfConns[i].UserID == userID {
			out = append(out, copyCFConnection(s.cfConns[i]))
		}
	}
	return out
}

// ListCFConnectionViews returns the API-safe projection with the active flag.
func (s *Store) ListCFConnectionViews(userID string) []models.CFConnectionView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	owner := s.findUserLocked(userID)
	activeID := ""
	if owner != nil {
		activeID = owner.ActiveCFConnectionID
	}
	out := make([]models.CFConnectionView, 0)
	for i := range s.cfConns {
		conn := &s.cfConns[i]
		if conn.UserID != userID {
			continue
		}
		out = append(out, models.CFConnectionView{
			ID:          conn.ID,
			Label:       conn.Label,
			AccountID:   conn.AccountID,
			AccountName: conn.AccountName,
			Active:      conn.ID == activeID,
			ExpiresAt:   conn.ExpiresAt,
			CreatedAt:   conn.CreatedAt,
		})
	}
	return out
}

// ActiveCFConnection returns the connection the user currently operates.
func (s *Store) ActiveCFConnection(userID string) (models.CFConnection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	owner := s.findUserLocked(userID)
	if owner == nil || owner.ActiveCFConnectionID == "" {
		return models.CFConnection{}, false
	}
	conn := s.findCFConnectionLocked(owner.ActiveCFConnectionID)
	if conn == nil || conn.UserID != userID {
		return models.CFConnection{}, false
	}
	return copyCFConnection(*conn), true
}

func (s *Store) findCFConnectionLocked(id string) *models.CFConnection {
	for i := range s.cfConns {
		if s.cfConns[i].ID == id {
			return &s.cfConns[i]
		}
	}
	return nil
}

func copyCFConnection(conn models.CFConnection) models.CFConnection {
	return conn
}

// CreateCFConnection stores a new connection and returns its id.
func (s *Store) CreateCFConnection(conn models.CFConnection) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findUserLocked(conn.UserID) == nil {
		return "", ErrUserNotFound
	}
	conn.ID = newID()
	conn.CreatedAt = time.Now().Unix()
	s.cfConns = append(s.cfConns, conn)
	if err := s.saveLocked(); err != nil {
		s.cfConns = s.cfConns[:len(s.cfConns)-1]
		return "", err
	}
	return conn.ID, nil
}

// UpdateCFConnectionTokens persists rotated OAuth tokens for one connection.
func (s *Store) UpdateCFConnectionTokens(id, accessToken, refreshToken string, expiresAt int64, scope string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.findCFConnectionLocked(id)
	if conn == nil {
		return ErrConnectionNotFound
	}
	prevAccess, prevRefresh, prevExpires, prevScope := conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.Scope
	conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.Scope = accessToken, refreshToken, expiresAt, scope
	if err := s.saveLocked(); err != nil {
		conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.Scope = prevAccess, prevRefresh, prevExpires, prevScope
		return err
	}
	return nil
}

// UpdateCFConnectionAccount records the account selected for a connection.
func (s *Store) UpdateCFConnectionAccount(id, accountID, accountName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.findCFConnectionLocked(id)
	if conn == nil {
		return ErrConnectionNotFound
	}
	prevID, prevName := conn.AccountID, conn.AccountName
	conn.AccountID, conn.AccountName = accountID, accountName
	if err := s.saveLocked(); err != nil {
		conn.AccountID, conn.AccountName = prevID, prevName
		return err
	}
	return nil
}

// SetActiveCFConnection switches the connection a user operates on.
func (s *Store) SetActiveCFConnection(userID, connID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.findUserLocked(userID)
	if owner == nil {
		return ErrUserNotFound
	}
	if connID != "" {
		conn := s.findCFConnectionLocked(connID)
		if conn == nil || conn.UserID != userID {
			return ErrConnectionNotFound
		}
	}
	previous := owner.ActiveCFConnectionID
	owner.ActiveCFConnectionID = connID
	if err := s.saveLocked(); err != nil {
		owner.ActiveCFConnectionID = previous
		return err
	}
	return nil
}

// DeleteCFConnection removes one owned connection, revoking its active
// selection and clearing the tunnel selection tied to it.
func (s *Store) DeleteCFConnection(userID, connID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.findCFConnectionLocked(connID)
	if conn == nil || conn.UserID != userID {
		return ErrConnectionNotFound
	}
	idx := -1
	for i := range s.cfConns {
		if s.cfConns[i].ID == connID {
			idx = i
			break
		}
	}
	previousConns := append([]models.CFConnection(nil), s.cfConns...)
	s.cfConns = append(s.cfConns[:idx], s.cfConns[idx+1:]...)
	owner := s.findUserLocked(userID)
	clearedActive, clearedTunnel := false, false
	if owner != nil && owner.ActiveCFConnectionID == connID {
		owner.ActiveCFConnectionID = ""
		clearedActive = true
		s.clearAllTunnelSelectionsLocked()
		clearedTunnel = true
	}
	if err := s.saveLocked(); err != nil {
		s.cfConns = previousConns
		if owner != nil && clearedActive {
			owner.ActiveCFConnectionID = connID
		}
		_ = clearedTunnel
		return err
	}
	return nil
}
