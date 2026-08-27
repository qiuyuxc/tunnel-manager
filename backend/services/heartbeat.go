package services

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxHeartbeatsPerTarget bounds retention in memory and on disk.
const maxHeartbeatsPerTarget = 1200

// Heartbeat is one recorded probe outcome.
type Heartbeat struct {
	T int64  `json:"t"` // unix millis
	S string `json:"s"` // ok | warn | down
	M int64  `json:"ms,omitempty"`
	C int    `json:"c,omitempty"` // http status code when any
	E string `json:"e,omitempty"`
}

func heartbeatKey(monitorID, targetID string) string { return monitorID + "|" + targetID }

// HeartbeatLog stores rolling probe history in a single JSON file,
// flushed on an interval so frequent checks stay cheap.
type HeartbeatLog struct {
	mu    sync.Mutex
	path  string
	data  map[string][]Heartbeat
	dirty bool
	stop  chan struct{}
}

// NewHeartbeatLog loads prior history from path (empty log when missing).
func NewHeartbeatLog(path string) *HeartbeatLog {
	l := &HeartbeatLog{path: path, data: map[string][]Heartbeat{}, stop: make(chan struct{})}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &l.data); err != nil {
			log.Printf("load heartbeat history: %v", err)
			l.data = map[string][]Heartbeat{}
		}
	}
	return l
}

// Append records one probe result, trimming retention.
func (l *HeartbeatLog) Append(monitorID, targetID string, hb Heartbeat) {
	key := heartbeatKey(monitorID, targetID)
	l.mu.Lock()
	list := append(l.data[key], hb)
	if len(list) > maxHeartbeatsPerTarget {
		list = list[len(list)-maxHeartbeatsPerTarget:]
	}
	l.data[key] = list
	l.dirty = true
	l.mu.Unlock()
}

// Recent returns a copy of entries for one target newer than sinceMS.
func (l *HeartbeatLog) Recent(monitorID, targetID string, sinceMS int64) []Heartbeat {
	return l.filter(heartbeatKey(monitorID, targetID), sinceMS)
}

// All returns copies of every entry newer than sinceMS keyed by monitor|target.
func (l *HeartbeatLog) All(sinceMS int64) map[string][]Heartbeat {
	out := make(map[string][]Heartbeat, len(l.data))
	l.mu.Lock()
	for k, list := range l.data {
		if v := filterLocked(list, sinceMS); len(v) > 0 {
			out[k] = v
		}
	}
	l.mu.Unlock()
	return out
}

func (l *HeartbeatLog) filter(key string, sinceMS int64) []Heartbeat {
	l.mu.Lock()
	defer l.mu.Unlock()
	return filterLocked(l.data[key], sinceMS)
}

func filterLocked(list []Heartbeat, sinceMS int64) []Heartbeat {
	out := make([]Heartbeat, 0, len(list))
	for _, hb := range list {
		if sinceMS <= 0 || hb.T >= sinceMS {
			out = append(out, hb)
		}
	}
	return out
}

// StartFlusher persists dirty state every interval until Stop.
func (l *HeartbeatLog) StartFlusher(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-l.stop:
				return
			case <-t.C:
				if err := l.Flush(); err != nil {
					log.Printf("flush heartbeat history: %v", err)
				}
			}
		}
	}()
}

// Stop halts the flusher after one final flush.
func (l *HeartbeatLog) Stop() {
	close(l.stop)
	if err := l.Flush(); err != nil {
		log.Printf("flush heartbeat history: %v", err)
	}
}

// Flush writes the log atomically when dirty.
func (l *HeartbeatLog) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.dirty {
		return nil
	}
	raw, err := json.Marshal(l.data)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(l.path); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, l.path); err != nil {
		return err
	}
	l.dirty = false
	return nil
}

// Forget drops every stored entry belonging to one monitor.
// ForgetTarget drops stored entries for one specific target.
func (l *HeartbeatLog) ForgetTarget(monitorID, targetID string) {
	key := heartbeatKey(monitorID, targetID)
	l.mu.Lock()
	delete(l.data, key)
	l.dirty = true
	l.mu.Unlock()
}
func (l *HeartbeatLog) Forget(monitorID string) {
	prefix := monitorID + "|"
	l.mu.Lock()
	for k := range l.data {
		if len(k) > len(prefix)-1 && k[:len(prefix)] == prefix {
			delete(l.data, k)
			l.dirty = true
		}
	}
	l.mu.Unlock()
}
