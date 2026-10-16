package ollama

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bg/internal/llm"
)

func TestComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"play_index\":0,\"reason\":\"ok\"}"}}]}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Model: "llama3.2", HTTP: srv.Client()}
	text, err := c.Complete(t.Context(), []llm.Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	d, err := llm.ParseDecision(text)
	if err != nil || d.PlayIndex != 0 {
		t.Fatalf("%s %+v", text, d)
	}
}

func TestCheckMissingModelAndDownServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"}]}`))
	}))
	defer srv.Close()
	okClient := &Client{BaseURL: srv.URL, Model: "llama3.2", HTTP: srv.Client()}
	if err := okClient.Check(t.Context()); err != nil {
		t.Fatal(err)
	}
	missing := &Client{BaseURL: srv.URL, Model: "missing", HTTP: srv.Client()}
	if err := missing.Check(t.Context()); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("err %v", err)
	}
	down := &Client{BaseURL: "http://127.0.0.1:1", Model: "llama3.2", HTTP: &http.Client{Timeout: time.Second}}
	if err := down.Check(t.Context()); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("err %v", err)
	}
}

func TestBaseURLWithoutScheme(t *testing.T) {
	got := NormalizeBaseURL("127.0.0.1:11434")
	if got != "http://127.0.0.1:11434" {
		t.Fatal(got)
	}
	if NormalizeBaseURL("http://127.0.0.1:11434/") != "http://127.0.0.1:11434" {
		t.Fatal(NormalizeBaseURL("http://127.0.0.1:11434/"))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:8b"}]}`))
	}))
	defer srv.Close()
	// srv.URL already has a scheme; the host:port form is what the settings field stored.
	c := &Client{BaseURL: strings.TrimPrefix(srv.URL, "http://"), Model: "qwen3:8b", HTTP: srv.Client()}
	names, err := c.Models(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "qwen3:8b" {
		t.Fatal(names)
	}
}

func TestMalformedChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}
	_, err := c.Complete(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatal(err)
	}
}
