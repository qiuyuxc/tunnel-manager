package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestLabHuaweiCancellationDuringLookupPreventsWrite(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Error("unexpected DNS write")
		}
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := newHuaweiDNSClient(server.URL, "access", "secret")
	client.httpClient.Transport = server.Client().Transport
	client.context = ctx
	done := make(chan error, 1)
	go func() { done <- client.UpdateARecord("zone-id", "", "edge.example.com", 300, []string{"1.1.1.1"}) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("DNS lookup did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled DNS update succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("DNS cancellation did not propagate")
	}
}

func TestLabHuaweiDNSUpdatesAndPreservesRecords(t *testing.T) {
	for _, mode := range []string{"create", "update", "unchanged", "empty", "error", "redirect", "cancel", "ipv6"} {
		t.Run(mode, func(t *testing.T) {
			var writes, requests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				if request.Header.Get("Authorization") == "" {
					t.Error("unsigned DNS API request")
				}
				if mode == "error" {
					http.Error(response, "unavailable", 503)
					return
				}
				if mode == "redirect" {
					response.Header().Set("Location", "/redirected")
					response.WriteHeader(302)
					return
				}
				if request.Method == http.MethodGet {
					if mode == "create" {
						fmt.Fprint(response, `{"recordsets":[]}`)
						return
					}
					ttl := 60
					if mode == "unchanged" {
						ttl = 300
					}
					fmt.Fprintf(response, `{"recordsets":[{"id":"record-id","name":"edge.example.com.","type":"A","ttl":%d,"records":["1.1.1.1"]}]}`, ttl)
					return
				}
				writes.Add(1)
				var payload huaweiRecordSet
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil || payload.TTL != 300 || len(payload.Records) != 1 {
					t.Error("invalid DNS payload")
				}
				fmt.Fprint(response, `{}`)
			}))
			defer server.Close()
			client := newHuaweiDNSClient(server.URL, "access", "secret")
			client.httpClient.Transport = server.Client().Transport
			ips := []string{"1.1.1.1"}
			if mode == "ipv6" {
				ips = []string{"2606:4700::1"}
			}
			if mode == "empty" {
				ips = nil
			}
			if mode == "cancel" {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				client.context = ctx
			}
			err := client.UpdateARecord("zone-id", "", "edge.example.com", 300, ips)
			wantSuccess := mode == "create" || mode == "update" || mode == "unchanged"
			if (err == nil) != wantSuccess {
				t.Fatalf("mode %s: %v", mode, err)
			}
			wantWrites := int32(0)
			if mode == "create" || mode == "update" {
				wantWrites = 1
			}
			if writes.Load() != wantWrites {
				t.Fatalf("writes: %d", writes.Load())
			}
			if mode == "redirect" && requests.Load() != 1 {
				t.Fatal("followed redirect")
			}
		})
	}
}
