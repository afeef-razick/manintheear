package stt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/afeef-razick/manintheear/internal/stt"
)

func TestNew_RejectsEmptyKey(t *testing.T) {
	_, err := stt.New("")
	if err == nil {
		t.Error("New() with empty key should return error")
	}
}

func TestTranscribe_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/transcriptions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing bearer token")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world\n"))
	}))
	defer srv.Close()

	p, err := stt.New("test-key")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	p.SetBaseURL(srv.URL)

	text, err := p.Transcribe(context.Background(), []byte("fake-audio"))
	if err != nil {
		t.Fatalf("Transcribe() error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("Transcribe() = %q, want %q", text, "hello world")
	}
}

func TestTranscribe_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid key"))
	}))
	defer srv.Close()

	p, err := stt.New("bad-key")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	p.SetBaseURL(srv.URL)

	_, err = p.Transcribe(context.Background(), []byte("fake-audio"))
	if err == nil {
		t.Error("Transcribe() expected error on 401, got nil")
	}
}
