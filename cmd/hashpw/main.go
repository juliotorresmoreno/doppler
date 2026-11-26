// hashpw prints a bcrypt hash for a proxy user's password.
//
//	read -rs P && printf '%s\n' "$P" | go run ./cmd/hashpw; unset P
//
// The password is read from stdin, so the shell's `read -s` keeps it off the
// screen. Put the output in auth.users in config.yml, e.g. "alice: $2a$10$...".
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/juliotorresmoreno/doppler/helper"
)

func main() {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stderr, "no password received on stdin")
		os.Exit(1)
	}
	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		fmt.Fprintln(os.Stderr, "empty password")
		os.Exit(1)
	}
	hash, err := helper.GeneratePassword(password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not hash password:", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
