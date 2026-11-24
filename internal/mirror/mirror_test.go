package mirror_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/mirror"
	"github.com/Rafael-devR/termux-setup.sh/internal/testutil"
)

func TestReleaseAge(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	age, err := mirror.ReleaseAge([]byte("Origin: x\nDate: Mon, 23 Nov 2026 12:00:00 UTC\n"), now)
	if err != nil || age != 24 {
		t.Fatal(age, err)
	}
	if _, err := mirror.ReleaseAge([]byte("nada"), now); err == nil {
		t.Fatal("esperava erro")
	}
}

func TestRank(t *testing.T) {
	ps := mirror.Rank([]mirror.Probe{
		{URL: "velho-perto", AgeHours: 100, Latency: time.Millisecond},
		{URL: "fresco-lento", AgeHours: 2, Latency: 900 * time.Millisecond},
		{URL: "fresco-rapido", AgeHours: 10, Latency: 100 * time.Millisecond},
		{URL: "muito-velho", AgeHours: 500},
	})
	got := strings.Join(mirror.URLs(ps), ",")
	if got != "fresco-rapido,fresco-lento,velho-perto,muito-velho" {
		t.Fatal(got)
	}
}

func TestApplyLegacyAndBackupOnce(t *testing.T) {
	e := testutil.Env(t)
	bk, err := mirror.Apply(e, "https://a.exemplo/", "t1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mirror.Apply(e, "https://b.exemplo", "t1"); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, e.AptSources()); got != "deb https://b.exemplo stable main\n" {
		t.Fatal(got)
	}
	if !strings.Contains(testutil.Read(t, filepath.Join(bk, "sources.list")), "velho.exemplo") {
		t.Fatal("o backup tem que guardar o ORIGINAL, não o da 1ª troca")
	}
	if mirror.Current(e) != "b.exemplo" {
		t.Fatal(mirror.Current(e))
	}
	if err := mirror.Restore(e, bk); err != nil || !strings.Contains(testutil.Read(t, e.AptSources()), "velho.exemplo") {
		t.Fatal("restore", err)
	}
}

func TestApplyDeb822(t *testing.T) {
	e := testutil.Env(t)
	src := filepath.Join(e.AptSourcesD(), "main.sources")
	testutil.Write(t, src, "Types: deb\nURIs: https://velho.exemplo\nSuites: stable\nComponents: main\n")
	if _, err := mirror.Apply(e, "https://novo.exemplo", "t1"); err != nil {
		t.Fatal(err)
	}
	if got := testutil.Read(t, src); !strings.Contains(got, "URIs: https://novo.exemplo\n") || !strings.Contains(got, "Suites: stable") {
		t.Fatal(got)
	}
	if got := testutil.Read(t, e.AptSources()); !strings.HasPrefix(got, "#deb ") {
		t.Fatalf("o sources.list antigo deve ser comentado: %q", got)
	}
	_ = os.Remove(src)
}
