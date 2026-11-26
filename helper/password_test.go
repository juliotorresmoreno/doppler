package helper

import "testing"

func TestCheckPolicy(t *testing.T) {
	full := Policy{MinLength: 10, RequireUpper: true, RequireDigit: true, RequireSymbol: true}
	if got := CheckPolicy("Abcdefg1!x", full); len(got) != 0 {
		t.Errorf("valid password rejected: %v", got)
	}
	if got := CheckPolicy("Ab1!", full); len(got) != 1 {
		t.Errorf("short password should fail only length, got %v", got)
	}
	if got := CheckPolicy("abcdefgh1!", full); len(got) != 1 {
		t.Errorf("missing uppercase should fail once, got %v", got)
	}
	relaxed := Policy{MinLength: 4}
	if got := CheckPolicy("abcd", relaxed); len(got) != 0 {
		t.Errorf("relaxed policy rejected valid password: %v", got)
	}
}
