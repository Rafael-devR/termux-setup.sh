package termux_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/termux"
	"github.com/Rafael-devR/termux-setup.sh/internal/testutil"
)

func TestBashrcIdempotentAndKeepsUserLines(t *testing.T) {
	e := testutil.Env(t)
	rc := filepath.Join(e.Home, ".bashrc")
	testutil.Write(t, rc, "alias meu='echo oi'\n")
	for range 3 {
		if err := termux.EnsureBashrc(e.Home); err != nil {
			t.Fatal(err)
		}
	}
	got := testutil.Read(t, rc)
	if strings.Count(got, ">>> termux-setup >>>") != 1 || !strings.Contains(got, "alias meu") || strings.Contains(got, "starship") {
		t.Fatal(got)
	}
	if err := termux.UndoBashrc(e.Home); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, rc); strings.Contains(got, "termux-setup") || !strings.Contains(got, "alias meu") {
		t.Fatal(got)
	}
}

func TestKeysReplaceOnlyExtraKeys(t *testing.T) {
	e := testutil.Env(t)
	p := filepath.Join(e.TermuxDir(), "termux.properties")
	testutil.Write(t, p, "bell-character=ignore\nextra-keys = [['ESC']]\n")
	if err := termux.SetKeys(e, "t1", "1"); err != nil {
		t.Fatal(err)
	}
	got := testutil.Read(t, p)
	if strings.Count(got, "extra-keys") != 1 || !strings.Contains(got, "claude ENTER") || !strings.Contains(got, "bell-character") {
		t.Fatal(got)
	}
	_ = termux.SetKeys(e, "t1", "3")
	if got := testutil.Read(t, p); strings.Contains(got, "extra-keys") {
		t.Fatal(got)
	}
}

func TestBackupOnceAndUndoLook(t *testing.T) {
	e := testutil.Env(t)
	colors := filepath.Join(e.TermuxDir(), "colors.properties")
	testutil.Write(t, colors, "ORIGINAL")
	if err := termux.WriteTheme(e, "t1"); err != nil {
		t.Fatal(err)
	}
	_ = termux.SetKeys(e, "t1", "1") // 2ª mudança: não pode sobrescrever o backup
	if err := termux.Hush(e); err != nil {
		t.Fatal(err)
	}
	if err := termux.UndoLook(e, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, colors); got != "ORIGINAL" {
		t.Fatalf("restaurado: %q", got)
	}
	if _, err := os.Stat(filepath.Join(e.Home, ".hushlogin")); err == nil {
		t.Fatal(".hushlogin deveria sumir")
	}
}

func TestUndoLookWhenTermuxDirDidNotExist(t *testing.T) {
	e := testutil.Env(t)
	_ = termux.SetKeys(e, "t1", "1")
	if err := termux.UndoLook(e, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, filepath.Join(e.TermuxDir(), "termux.properties")); strings.Contains(got, "extra-keys") {
		t.Fatal(got)
	}
}

func TestFontChecksSHA(t *testing.T) {
	e := testutil.Env(t)
	e.DryRun = false // força a conferência do SHA: o zip falso não bate
	err := termux.InstallFont(context.Background(), e, sys.DryFetcher{}, "t1", io.Discard)
	if err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Fatal(err)
	}
	e.DryRun = true
	if err := termux.InstallFont(context.Background(), e, sys.DryFetcher{}, "t1", io.Discard); err != nil {
		t.Fatal(err)
	}
	if testutil.Read(t, filepath.Join(e.TermuxDir(), "font.ttf")) == "" {
		t.Fatal("fonte não gravada")
	}
}

func TestUbuntuCommands(t *testing.T) {
	e := testutil.Env(t)
	if err := termux.WriteUbuntuCommands(e, termux.UbuntuNames{Distro: "ubuntu", User: "dev"}); err != nil {
		t.Fatal(err)
	}
	got := testutil.Read(t, filepath.Join(e.BinDir(), "ia-run"))
	if !strings.Contains(got, "login ubuntu --user dev") || strings.Contains(got, "{{") {
		t.Fatal(got)
	}
	if err := termux.WriteCommands(e); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(filepath.Join(e.BinDir(), "atualizar"))
	if st == nil || st.Mode()&0o100 == 0 {
		t.Fatal("atualizar deve ser executável")
	}
}

func TestOpenCodeCommand(t *testing.T) {
	e := testutil.Env(t)
	if err := termux.WriteUbuntuCommands(e, termux.UbuntuNames{Distro: "ubuntu", User: "dev"}); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, filepath.Join(e.BinDir(), "opencode")); !strings.Contains(got, `ia-run" opencode "$@"`) {
		t.Fatal(got)
	}
}
