package auth

import (
	"testing"
	"time"
)

func TestSigner_IssueAndParse_RoundTrip(t *testing.T) {
	s := NewSigner("test-secret", time.Minute)
	tok, exp, err := s.Issue("user-123")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
	if time.Until(exp) <= 0 {
		t.Fatalf("exp must be in future, got %v", exp)
	}

	claims, err := s.Parse(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("user_id: got %q want %q", claims.UserID, "user-123")
	}
}

func TestSigner_Parse_InvalidToken(t *testing.T) {
	s := NewSigner("test-secret", time.Minute)
	if _, err := s.Parse("not-a-jwt"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestSigner_Parse_WrongSecret(t *testing.T) {
	s1 := NewSigner("secret-one", time.Minute)
	s2 := NewSigner("secret-two", time.Minute)

	tok, _, err := s1.Issue("user-123")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := s2.Parse(tok); err == nil {
		t.Fatal("expected error parsing with different secret")
	}
}

func TestSigner_Parse_Expired(t *testing.T) {
	s := NewSigner("test-secret", -time.Second) // already expired
	tok, _, err := s.Issue("user-123")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := s.Parse(tok); err == nil {
		t.Fatal("expected error for expired token")
	}
}
