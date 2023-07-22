package helper

import (
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func VerifyPassword(s string) (valid, sevenOrMore, number, upper, special bool) {
	letters := 0
	for _, c := range s {
		switch {
		case unicode.IsNumber(c):
			number = true
		case unicode.IsUpper(c):
			upper = true
			letters++
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			special = true
		case unicode.IsLetter(c) || c == ' ':
			letters++
		default:
		}
		valid = sevenOrMore && number && upper && special
		if valid {
			return
		}
	}
	sevenOrMore = letters >= 7
	valid = sevenOrMore && number && upper && special
	return
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
