package handlers

import (
	"net/http"
	"strconv"
	"time"
)

// alertFeedItem is one alert as the mobile shell consumes it: the AlertLog row
// plus the owning monitor's display name, which the log itself does not carry.
type alertFeedItem struct {
	MonitorID   string `json:"monitor_id"`
	MonitorName string `json:"monitor_name"`
	TargetID    string `json:"target_id"`
	TargetName  string `json:"target_name"`
	State       string `json:"state"`
	HTTPCode    int    `json:"http_code,omitempty"`
	Error       string `json:"error,omitempty"`
	Notified    bool   `json:"notified"`
	Detail      string `json:"detail,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

// AlertsFeed handles GET /api/alerts?since=<unix seconds>.
//
// Alerts are already produced by the monitor runner for the email channel; this
// exposes the same rows as a cursor feed so a client can poll once for every
// monitor it may see and catch up after being offline. A missing or zero since
// seeds the cursor without replaying history, which is what a client wants on
// its first run.
func (h *MonitorsHandler) AlertsFeed(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	if since <= 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"cursor": now,
			"now":    now,
			"alerts": []alertFeedItem{},
		})
		return
	}

	user := SessionUser(r)
	names := map[string]string{}
	for _, m := range h.st.GetConfig().Monitors {
		if user != nil && (user.IsAdmin() || m.UserID == user.ID) {
			names[m.ID] = m.Name
		}
	}

	items := make([]alertFeedItem, 0, 16)
	for _, e := range h.st.AlertsSince(since, 100) {
		name, ok := names[e.MonitorID]
		if !ok {
			continue
		}
		items = append(items, alertFeedItem{
			MonitorID:   e.MonitorID,
			MonitorName: name,
			TargetID:    e.TargetID,
			TargetName:  e.TargetName,
			State:       e.State,
			HTTPCode:    e.HTTPCode,
			Error:       e.Error,
			Notified:    e.Notified,
			Detail:      e.Detail,
			CreatedAt:   e.CreatedAt,
		})
	}

	cursor := since
	for _, it := range items {
		if it.CreatedAt > cursor {
			cursor = it.CreatedAt
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"cursor": cursor,
		"now":    now,
		"alerts": items,
	})
}
