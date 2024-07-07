package main

import (
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Errorf("Error: Failed to generate secret: %v", err)
	}
	token, err := GenerateTotp(secret, time.Now())
	if err != nil {
		t.Errorf("Error: Failed to generate TOTP: %v", err)
	}
	if !ValidateTotp(secret, token) {
		t.Errorf("Error: Failed to validate TOTP")
	}
}
