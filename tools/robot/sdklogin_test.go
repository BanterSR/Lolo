package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestSdkPostRetriesTransportEOF(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var out struct {
		OK bool `json:"ok"`
	}
	if err := sdkPost(server.URL, map[string]string{"test": "value"}, false, &out); err != nil {
		t.Fatalf("sdkPost failed after retry: %v", err)
	}
	if !out.OK || calls.Load() != 2 {
		t.Fatalf("out=%+v calls=%d, want successful retry", out, calls.Load())
	}
}

func TestSdkPostRetriesServerError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var out struct {
		OK bool `json:"ok"`
	}
	if err := sdkPost(server.URL, map[string]string{"test": "value"}, false, &out); err != nil {
		t.Fatalf("sdkPost failed after retry: %v", err)
	}
	if !out.OK || calls.Load() != 2 {
		t.Fatalf("out=%+v calls=%d, want successful retry", out, calls.Load())
	}
}
