package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"tunnel-manager/models"
)

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
