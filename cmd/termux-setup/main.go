// Comando termux-setup: configura o Termux do zero.
package main

import (
	"os"

	"github.com/Rafael-devR/termux-setup.sh/internal/cli"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

func main() {
	sys.FixTimezone()
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
