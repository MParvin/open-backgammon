package openrouter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bg/internal/llm"
)

func TestParseDecision(t *testing.T) {
	d, err := ParseDecision("Sure.\n```json\n{\"play_index\": 2, \"reason\": \"hits the blot\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if d.PlayIndex != 2 || d.Reason != "hits the blot" {
		t.Fatalf("%+v", d)
	}
}

func TestComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"play_index\":1,\"reason\":\"ok\"}"}}]}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "secret", Model: "openai/gpt-4o-mini", Endpoint: srv.URL, HTTP: srv.Client()}
	text, err := c.Complete(t.Context(), []llm.Message{{Role: "user", Content: "move"}})
	if err != nil {
		t.Fatal(err)
	}
	d, err := llm.ParseDecision(text)
	if err != nil || d.PlayIndex != 1 {
		t.Fatalf("text=%s err=%v d=%+v", text, err, d)
	}
}

func TestCheckRejectsBadKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatal("missing auth")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := &Client{APIKey: "secret-key-value", ModelsURL: srv.URL, HTTP: srv.Client()}
	err := c.Check(t.Context())
	if err == nil {
		t.Fatal("expected rejection")
	}
	if strings.Contains(err.Error(), "secret-key-value") {
		t.Fatal("error leaked the key")
	}
	empty := &Client{}
	if err := empty.Check(t.Context()); err == nil {
		t.Fatal("empty key")
	}
}
