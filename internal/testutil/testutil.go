package testutil

import (
	"os"
	"testing"
)

func ReadFile(t *testing.T, file string) []byte {
	data, err := os.ReadFile(file)
	IsNil(t, err, "File was found")

	return data
}

func Assert[T comparable](t *testing.T, expected T, value T, message string) {
	if expected != value {
		t.Fatalf("%s: expected %v got %v", message, expected, value)
	}
}

func AssertErr(t *testing.T, expected error, value error, message string) {
	if expected == nil && value == nil {
		return
	}

	if expected == nil || value == nil || expected.Error() != value.Error() {
		t.Fatalf("%s: expected %v got %v", message, expected, value)
	}
}

func IsNil(t *testing.T, value interface{}, message string) {
	if value != nil {
		t.Fatalf("%s: expected nil got %v", message, value)
	}
}

func IsNotNil(t *testing.T, value interface{}, message string) {
	if value == nil {
		t.Fatalf("%s: expected not nil got nil", message)
	}
}
