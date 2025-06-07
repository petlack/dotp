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

func TestGenerateTotpUri(t *testing.T) {
	uri := GenerateTotpUri("ABCDEF", "foo@bar", "myapp")
	expected := "otpauth://totp/myapp:foo@bar?issuer=myapp&secret=ABCDEF"
	if uri != expected {
		t.Errorf("Unexpected URI: %s", uri)
	}
}
