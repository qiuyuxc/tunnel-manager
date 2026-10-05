package store

import (
	"encoding/json"
	"errors"

	"tunnel-manager/models"
)

func cloneAIState(state models.AIState) models.AIState {
	data, _ := json.Marshal(state)
	var copy models.AIState
	_ = json.Unmarshal(data, &copy)
	if copy.Users == nil {
		copy.Users = map[string]models.AIUserState{}
	}
	return copy
}

func (s *Store) AIStateFor(userID string) (bool, models.AIConnection, models.AIUserState) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := cloneAIState(models.AIState{Users: map[string]models.AIUserState{userID: s.assistant.Users[userID]}})
	return s.assistant.SharedEnabled, s.assistant.Shared, state.Users[userID]
}

func (s *Store) UpdateAI(userID string, update func(*models.AIState, *models.AIUserState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for _, user := range s.users {
		if user.ID == userID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("用户不存在")
	}
	previous := s.assistant
	next := cloneAIState(previous)
	userState := next.Users[userID]
	if err := update(&next, &userState); err != nil {
		return err
	}
	next.Users[userID] = userState
	s.assistant = next
	if err := s.saveLocked(); err != nil {
		s.assistant = previous
		return errors.New("保存 AI 数据失败")
	}
	return nil
}
