package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"tunnel-manager/models"
)

func TestDNSDetailByFullHostname(t *testing.T) {
	var lastText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(req.URL.Path, "/zones"):
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"zone-a","name":"kukie.cn"}]}`))
		case strings.Contains(req.URL.Path, "/dns_records"):
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"r1","type":"CNAME","name":"bbs.kukie.cn","content":"t.cfargotunnel.com","ttl":1,"proxied":true}]}`))
		case strings.Contains(req.URL.Path, "/sendMessage"):
			var body struct {
				Text string `json:"text"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			lastText = body.Text
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":false,"description":"unhandled"}`))
		}
	}))
	defer server.Close()
	cf := &CloudflareClient{apiToken: "test", accountID: "acct", baseURL: server.URL, httpClient: server.Client()}
	bot := &TelegramBot{cf: cf, httpClient: server.Client(), confirmations: map[string]dnsDeleteConfirmation{}}
	cfg := models.Config{TGBotToken: "test", TGApiEndpoint: server.URL}
	bot.handleDNSDetail(cfg, 123, "bbs.kukie.cn", "bbs.kukie.cn")
	if !strings.Contains(lastText, "bbs.kukie.cn") || !strings.Contains(lastText, "t.cfargotunnel.com") {
		t.Fatalf("unexpected output: %q", lastText)
	}
}

func TestLooksLikeRecordID(t *testing.T) {
	if !looksLikeRecordID("0123456789abcdef0123456789abcdef") {
		t.Fatal("valid 32-hex rejected")
	}
	if looksLikeRecordID("0123456789abcdef0123456789abcde") {
		t.Fatal("short ID accepted")
	}
	if looksLikeRecordID("bbs.kukie.cn") {
		t.Fatal("hostname accepted as ID")
	}
	if looksLikeRecordID("0123456789abcdef0123456789abcdez") {
		t.Fatal("non-hex accepted")
	}
}

func TestResolveRecordArg(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"rec-1","type":"CNAME","name":"bbs.kukie.cn","content":"a.cfargotunnel.com"},{"id":"11111111111111111111111111111111","type":"A","name":"www.kukie.cn","content":"1.2.3.4"},{"id":"rec-3","type":"AAAA","name":"www.kukie.cn","content":"::1"}]}`))
	}))
	defer server.Close()
	cf := &CloudflareClient{apiToken: "test", accountID: "acct", baseURL: server.URL, httpClient: server.Client()}
	bot := &TelegramBot{cf: cf}
	if r, err := bot.resolveRecordArg("z", "bbs"); err != nil || r.ID != "rec-1" {
		t.Fatalf("by name: %+v %v", r, err)
	}
	if r, err := bot.resolveRecordArg("z", "11111111111111111111111111111111"); err != nil || r.ID != "11111111111111111111111111111111" {
		t.Fatalf("by id: %+v %v", r, err)
	}
	if _, err := bot.resolveRecordArg("z", "missing"); err == nil {
		t.Fatal("expected not found")
	}
	if _, err := bot.resolveRecordArg("z", "www"); err == nil || !strings.Contains(err.Error(), "多条") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
}

func TestDNSUpdateByRecordName(t *testing.T) {
	var updatedID, lastText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(req.URL.Path, "/zones"):
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"zone-a","name":"kukie.cn"}]}`))
		case strings.HasSuffix(req.URL.Path, "/dns_records"):
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"rec-1","type":"CNAME","name":"bbs.kukie.cn","content":"old.cfargotunnel.com","ttl":1,"proxied":true}]}`))
		case strings.Contains(req.URL.Path, "/dns_records/"):
			updatedID = req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
			_, _ = w.Write([]byte(`{"success":true,"result":{}}`))
		case strings.Contains(req.URL.Path, "/sendMessage"):
			var body struct {
				Text string `json:"text"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			lastText = body.Text
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":false,"description":"unhandled"}`))
		}
	}))
	defer server.Close()
	cf := &CloudflareClient{apiToken: "test", accountID: "acct", baseURL: server.URL, httpClient: server.Client()}
	bot := &TelegramBot{cf: cf, httpClient: server.Client(), confirmations: map[string]dnsDeleteConfirmation{}}
	cfg := models.Config{TGBotToken: "test", TGApiEndpoint: server.URL}
	// 模拟新格式: /DNS修改 kukie.cn bbs.kukie.cn CNAME saas.com auto
	bot.handleDNSWriteCommand(cfg, 123, "bbs.kukie.cn", []string{"kukie.cn", "CNAME", "bbs.kukie.cn", "saas.com", "auto"})
	if updatedID != "rec-1" {
		t.Fatalf("updated wrong record: %q", updatedID)
	}
	if !strings.Contains(lastText, "saas.com") || !strings.Contains(lastText, "已修改") {
		t.Fatalf("unexpected output: %q", lastText)
	}
}

func TestResolveTunnelArg(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"t-1","name":"haxvps-2","status":"healthy"},{"id":"t-2","name":"haxvps-3","status":"healthy"}]}`))
	}))
	defer server.Close()
	cf := &CloudflareClient{apiToken: "test", accountID: "acct", baseURL: server.URL, httpClient: server.Client()}
	bot := &TelegramBot{cf: cf}
	if tun, err := bot.resolveTunnelArg("t-1"); err != nil || tun.Name != "haxvps-2" {
		t.Fatalf("by id: %+v %v", tun, err)
	}
	if tun, err := bot.resolveTunnelArg("HAXVPS-2"); err != nil || tun.ID != "t-1" {
		t.Fatalf("by name: %+v %v", tun, err)
	}
	if _, err := bot.resolveTunnelArg("missing"); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestParseDNSWrite(t *testing.T) {
	zone, p, err := parseDNSWrite([]string{"zone", "CNAME", "app.example.com", "target.example.com", "auto", "on"})
	if err != nil {
		t.Fatal(err)
	}
	if zone != "zone" || p.Type != "CNAME" || p.TTL != 1 || !p.Proxied {
		t.Fatalf("unexpected: %s %+v", zone, p)
	}
	_, mx, err := parseDNSWrite([]string{"zone", "MX", "example.com", "mail.example.com", "300", "off", "10"})
	if err != nil {
		t.Fatal(err)
	}
	if mx.Priority == nil || *mx.Priority != 10 || mx.Proxied {
		t.Fatalf("unexpected MX: %+v", mx)
	}
	for _, bad := range [][]string{{"zone", "SRV", "x", "y"}, {"zone", "A", "x", "1.2.3.4", "auto", "onn"}, {"zone", "TXT", "x", "value", "auto", "on"}, {"zone", "A", "x", "1.2.3.4", "auto", "off", "extra"}} {
		if _, _, err = parseDNSWrite(bad); err == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
}

func TestParseDNSWriteDefaults(t *testing.T) {
	// 无区域：返回空 zone，TTL 缺省 auto，代理缺省 on
	zone, p, err := parseDNSWrite([]string{"CNAME", "app", "target"})
	if err != nil {
		t.Fatal(err)
	}
	if zone != "" || p.TTL != 1 || !p.Proxied {
		t.Fatalf("defaults: %q %+v", zone, p)
	}
	// 只带 TTL，不带代理
	_, p, err = parseDNSWrite([]string{"A", "www", "1.2.3.4", "300"})
	if err != nil || p.TTL != 300 || !p.Proxied {
		t.Fatalf("ttl only: %+v %v", p, err)
	}
	// 只带代理，不带 TTL
	_, p, err = parseDNSWrite([]string{"A", "www", "1.2.3.4", "off"})
	if err != nil || p.TTL != 1 || p.Proxied {
		t.Fatalf("proxy only: %+v %v", p, err)
	}
	// MX 末尾数字为优先级，TTL 缺省 auto，代理强制关
	_, p, err = parseDNSWrite([]string{"MX", "mail", "mail.example.com", "10"})
	if err != nil || p.Priority == nil || *p.Priority != 10 || p.TTL != 1 || p.Proxied {
		t.Fatalf("mx default: %+v %v", p, err)
	}
	// TXT 默认关代理（不显式填 on 时不报错）
	_, p, err = parseDNSWrite([]string{"TXT", "x", "v"})
	if err != nil || p.Proxied {
		t.Fatalf("txt default: %+v %v", p, err)
	}
	// TTL 越界
	if _, _, err = parseDNSWrite([]string{"A", "www", "1.2.3.4", "50"}); err == nil {
		t.Fatal("expected TTL range error")
	}
	// 参数不足
	if _, _, err = parseDNSWrite([]string{"A", "www"}); err == nil {
		t.Fatal("expected arity error")
	}
}

func TestIsRecordType(t *testing.T) {
	for _, ok := range []string{"A", "a", "AAAA", "CNAME", "cname", "TXT", "MX"} {
		if !isRecordType(ok) {
			t.Fatalf("%s should be a record type", ok)
		}
	}
	for _, bad := range []string{"SRV", "NS", "SOA", ""} {
		if isRecordType(bad) {
			t.Fatalf("%s should not be a record type", bad)
		}
	}
}

func TestDNSDeleteConfirmationIsScopedAndSingleUse(t *testing.T) {
	b := &TelegramBot{confirmations: map[string]dnsDeleteConfirmation{"abcd": {ChatID: 10, UserID: 20, ZoneID: "z", Record: models.DNSRecord{ID: "r"}, Expires: time.Now().Add(time.Minute)}}}
	b.confirmMu.Lock()
	c, ok := b.confirmations["abcd"]
	if ok {
		delete(b.confirmations, "abcd")
	}
	b.confirmMu.Unlock()
	if !ok || c.ChatID != 10 || c.UserID != 20 {
		t.Fatalf("bad confirmation: %+v", c)
	}
	if _, ok := b.confirmations["abcd"]; ok {
		t.Fatal("confirmation was reusable")
	}
}

func TestSplitCommandArgs(t *testing.T) {
	got, err := splitCommandArgs("/DNS添加 zone TXT example.com \"hello world\" auto off")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/DNS添加", "zone", "TXT", "example.com", "hello world", "auto", "off"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	got, err = splitCommandArgs("/DNS添加 z TXT n \"a\\b\"")
	if err != nil || got[4] != "a\\b" {
		t.Fatalf("backslash lost: %v %v", got, err)
	}
	if _, err = splitCommandArgs("/DNS添加 z TXT n \"open"); err == nil {
		t.Fatal("expected unclosed quote error")
	}
	empty, err := splitCommandArgs("cmd \"\"")
	if err != nil || len(empty) != 2 || empty[1] != "" {
		t.Fatalf("empty quote failed: %v %v", empty, err)
	}
}

func TestTelegramCommandRecipient(t *testing.T) {
	if cmd, ok := telegramCommandForBot("/dns_list@MyBot", "mybot"); !ok || cmd != "/dns_list" {
		t.Fatal(cmd, ok)
	}
	if _, ok := telegramCommandForBot("/dns_delete@OtherBot", "mybot"); ok {
		t.Fatal("accepted another bot recipient")
	}
	if cmd, ok := telegramCommandForBot("/dns_list", "mybot"); !ok || cmd != "/dns_list" {
		t.Fatal(cmd, ok)
	}
}
func TestSplitTelegramText(t *testing.T) {
	text := strings.Repeat("字", 9000)
	parts := splitTelegramText(text, 3900)
	if len(parts) != 3 {
		t.Fatalf("parts=%d", len(parts))
	}
	for _, p := range parts {
		if len([]rune(p)) > 3900 {
			t.Fatal("oversized chunk")
		}
	}
	if strings.Join(parts, "") != text {
		t.Fatal("content changed")
	}
}

func TestSendMessageReturnsTelegramFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"ok\":false,\"description\":\"chat not found\"}"))
	}))
	defer server.Close()
	bot := &TelegramBot{httpClient: server.Client()}
	err := bot.sendMessage(models.Config{TGBotToken: "test", TGApiEndpoint: server.URL}, 123, "hello")
	if err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveZoneArg(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"success\":true,\"result\":[{\"id\":\"zone-a\",\"name\":\"example.com\"},{\"id\":\"zone-b\",\"name\":\"example.org\"}]}"))
	}))
	defer server.Close()
	cf := &CloudflareClient{apiToken: "test", baseURL: server.URL, httpClient: server.Client()}
	bot := &TelegramBot{cf: cf}
	if id, name, err := bot.resolveZoneArg("app.example.com"); err != nil || id != "zone-a" || name != "example.com" {
		t.Fatalf("hostname match: id=%s name=%s err=%v", id, name, err)
	}
	if id, name, err := bot.resolveZoneArg("example.org"); err != nil || id != "zone-b" || name != "example.org" {
		t.Fatalf("exact zone name: id=%s name=%s err=%v", id, name, err)
	}
	if id, _, err := bot.resolveZoneArg("zone-b"); err != nil || id != "zone-b" {
		t.Fatalf("zone id: id=%s err=%v", id, err)
	}
	if _, _, err := bot.resolveZoneArg("unknown.net"); err == nil {
		t.Fatal("expected error for unknown hostname")
	}
}

func TestRecordNameMatches(t *testing.T) {
	cases := []struct {
		rec, arg string
		want     bool
	}{
		{"www.169977.xyz", "www", true},
		{"www.169977.xyz", "www.169977.xyz", true},
		{"www.169977.xyz", "WWW", true},
		{"www.169977.xyz", "web", false},
		{"169977.xyz", "169977.xyz", true},
		{"169977.xyz", "ww", false},
	}
	for _, c := range cases {
		if got := recordNameMatches(c.rec, c.arg); got != c.want {
			t.Fatalf("recordNameMatches(%q, %q) = %v, want %v", c.rec, c.arg, got, c.want)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("1234567890", 4); got != "1234…" {
		t.Fatalf("got %q", got)
	}
	if got := truncateRunes("1234", 4); got != "1234" {
		t.Fatalf("should not truncate: %q", got)
	}
	if got := truncateRunes("", 4); got != "" {
		t.Fatalf("empty: %q", got)
	}
}

func TestSameDNSRecord(t *testing.T) {
	a := models.DNSRecord{ID: "r", Type: "A", Name: "x", Content: "1.2.3.4", TTL: 1, Proxied: true}
	if !sameDNSRecord(a, a) {
		t.Fatal("same record differs")
	}
	b := a
	b.Content = "5.6.7.8"
	if sameDNSRecord(a, b) {
		t.Fatal("changed record accepted")
	}
}
