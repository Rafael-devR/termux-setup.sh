package catalog_test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
	"github.com/Rafael-devR/termux-setup.sh/modules"
)

func load(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load(modules.FS)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestEmbeddedCatalogIsValid(t *testing.T) {
	c := load(t)
	if err := c.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if len(c.All()) < 20 {
		t.Fatalf("esperava os ~26 módulos, veio %d", len(c.All()))
	}
}

func TestResolveAddsDependenciesInOrder(t *testing.T) {
	c := load(t)
	r, err := c.Resolve([]string{"codex"}, "aarch64")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range r.Modules {
		ids = append(ids, m.ID)
	}
	if !slices.Equal(ids, []string{"ubuntu", "ubnode", "codex"}) {
		t.Fatalf("ordem: %v", ids)
	}
	if !slices.Contains(r.Added, "ubuntu") || !slices.Contains(r.Added, "ubnode") {
		t.Fatalf("added: %v", r.Added)
	}
}

func TestResolveDrops64BitOnlyOn32Bit(t *testing.T) {
	c := load(t)
	r, err := c.Resolve([]string{"claude", "codex", "gemini"}, "armv7l")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range r.Modules {
		if m.ID == "claude" || m.ID == "codex" {
			t.Fatalf("%s não roda em 32 bits", m.ID)
		}
	}
	if !slices.Contains(r.Dropped, "claude") || !slices.Contains(r.Dropped, "codex") {
		t.Fatalf("dropped: %v", r.Dropped)
	}
}

func TestPresets(t *testing.T) {
	c := load(t)
	if slices.Contains(c.Preset("tudo"), "root") {
		t.Fatal("root não deve entrar em 'tudo'")
	}
	if !slices.Contains(c.Preset("dev"), "python") {
		t.Fatal("dev sem python")
	}
}

func TestValidateCatchesErrors(t *testing.T) {
	fs := fstest.MapFS{
		"groups.toml": {Data: []byte("[[group]]\nid = \"g\"\ntitle = \"G\"\n")},
		"a.toml":      {Data: []byte("name = \"A\"\ngroup = \"g\"\nrequires = [\"b\"]\n")},
		"b.toml":      {Data: []byte("name = \"B\"\ngroup = \"x\"\nrequires = [\"a\"]\n[[steps]]\nlabel = \"s\"\naction = \"nada\"\n")},
	}
	c, err := catalog.Load(fs)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Validate(func(a string) bool { return a != "nada" })
	for _, want := range []string{"grupo \"x\"", "ação \"nada\"", "circular"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("faltou erro %q em: %v", want, err)
		}
	}
}

func TestUnknownFieldIsError(t *testing.T) {
	fs := fstest.MapFS{"a.toml": {Data: []byte("name = \"A\"\npacotes = [\"x\"]\n")}}
	if _, err := catalog.Load(fs); err == nil {
		t.Fatal("campo desconhecido deveria dar erro")
	}
}
