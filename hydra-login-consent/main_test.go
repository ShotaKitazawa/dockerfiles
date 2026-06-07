package main

import (
	"testing"
)

func TestNewClaimChecker_InvalidExpression(t *testing.T) {
	_, err := newClaimChecker("this is not valid CEL !!!")
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}

func TestNewClaimChecker_NonBoolExpression(t *testing.T) {
	checker, err := newClaimChecker("claims.email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = checker.allow(map[string]any{"email": "user@example.com"})
	if err == nil {
		t.Fatal("expected error when expression returns non-bool, got nil")
	}
}

func TestClaimChecker_Allow(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		claims     map[string]any
		want       bool
	}{
		{
			name:       "email exact match allowed",
			expression: "claims.email == 'user@example.com'",
			claims:     map[string]any{"email": "user@example.com"},
			want:       true,
		},
		{
			name:       "email exact match denied",
			expression: "claims.email == 'user@example.com'",
			claims:     map[string]any{"email": "other@example.com"},
			want:       false,
		},
		{
			name:       "OR condition: first matches",
			expression: "claims.email == 'a@example.com' || claims.email == 'b@example.com'",
			claims:     map[string]any{"email": "a@example.com"},
			want:       true,
		},
		{
			name:       "OR condition: second matches",
			expression: "claims.email == 'a@example.com' || claims.email == 'b@example.com'",
			claims:     map[string]any{"email": "b@example.com"},
			want:       true,
		},
		{
			name:       "OR condition: none matches",
			expression: "claims.email == 'a@example.com' || claims.email == 'b@example.com'",
			claims:     map[string]any{"email": "c@example.com"},
			want:       false,
		},
		{
			name:       "AND condition: both match",
			expression: "claims.email_verified == true && claims.email == 'user@example.com'",
			claims:     map[string]any{"email": "user@example.com", "email_verified": true},
			want:       true,
		},
		{
			name:       "AND condition: one fails",
			expression: "claims.email_verified == true && claims.email == 'user@example.com'",
			claims:     map[string]any{"email": "user@example.com", "email_verified": false},
			want:       false,
		},
		{
			name:       "missing claim field",
			expression: "claims.email == 'user@example.com'",
			claims:     map[string]any{},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker, err := newClaimChecker(tt.expression)
			if err != nil {
				t.Fatalf("newClaimChecker: %v", err)
			}
			got, err := checker.allow(tt.claims)
			if err != nil {
				t.Fatalf("allow: %v", err)
			}
			if got != tt.want {
				t.Errorf("allow() = %v, want %v", got, tt.want)
			}
		})
	}
}
