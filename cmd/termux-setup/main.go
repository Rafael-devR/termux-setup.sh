// Comando termux-setup: configura o Termux do zero.
package main

import (
	"os"

	"github.com/Rafael-devR/termux-setup.sh/internal/cli"
)

func main() { os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr)) }
