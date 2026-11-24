// Package modules embute os manifestos *.toml no binário.
package modules

import "embed"

// FS contém groups.toml e um arquivo por ferramenta.
//
//go:embed *.toml
var FS embed.FS
