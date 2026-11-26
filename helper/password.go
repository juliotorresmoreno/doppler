package helper

import (
	"fmt"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// Policy is the set of requirements a new password must meet.
type Policy struct {
	MinLength     int
	RequireUpper  bool
	RequireDigit  bool
	RequireSymbol bool
}

// CheckPolicy returns one message per requirement the password does not meet.
// An empty result means the password is acceptable.
func CheckPolicy(s string, p Policy) []string {
	var problems []string
	if len([]rune(s)) < p.MinLength {
		problems = append(problems, fmt.Sprintf("must be at least %d characters", p.MinLength))
	}
	var upper, digit, symbol bool
	for _, c := range s {
		switch {
		case unicode.IsUpper(c):
			upper = true
		case unicode.IsDigit(c):
			digit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			symbol = true
		}
	}
	if p.RequireUpper && !upper {
		problems = append(problems, "must contain an uppercase letter")
	}
	if p.RequireDigit && !digit {
		problems = append(problems, "must contain a digit")
	}
	if p.RequireSymbol && !symbol {
		problems = append(problems, "must contain a symbol")
	}
	return problems
}

func GeneratePassword(password string) (string, error) {
	s, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func ValidatePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(password),
	)
	return err
}
