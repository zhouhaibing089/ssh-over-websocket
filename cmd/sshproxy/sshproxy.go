package main

import (
	"os"

	"github.com/zhouhaibing089/sshproxy/cmd/sshproxy/app"
)

func main() {
	cmd := app.NewProxyCommand()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
