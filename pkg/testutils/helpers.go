package testutils

import "testing"

func ErrorsShouldMatch(t *testing.T, expectedError error, actualError error) {
	t.Helper()
	if expectedError != actualError {
		t.Errorf("expected error [%v], got error [%v]", expectedError, actualError)
	}
}
