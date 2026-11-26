// hashpw prints a bcrypt hash for a proxy user's password, after checking it
// against the auth.password policy in the config.
//
//	read -rs P && printf '%s\n' "$P" | go run ./cmd/hashpw -c /etc/doppler/config.yml; unset P
//
// The password is read from stdin, so the shell's `read -s` keeps it off the
// screen. Without -c (or if the config cannot be read) the default policy is used.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/helper"
)

func main() {
	policy := helper.Policy{MinLength: 10, RequireUpper: true, RequireDigit: true, RequireSymbol: true}
	if conf, err := config.GetConfig(); err == nil {
		p := conf.Auth.Password
		policy = helper.Policy{MinLength: p.MinLength, RequireUpper: p.RequireUpper, RequireDigit: p.RequireDigit, RequireSymbol: p.RequireSymbol}
	} else {
		fmt.Fprintln(os.Stderr, "warning: config not read, using default password policy:", err)
	}

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
	if problems := helper.CheckPolicy(password, policy); len(problems) > 0 {
		fmt.Fprintln(os.Stderr, "password rejected:")
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, " -", p)
		}
		os.Exit(1)
	}
	hash, err := helper.GeneratePassword(password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not hash password:", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
