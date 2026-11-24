package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		cur, latest string
		want        bool
	}{
		{"v6.0.3", "v6.1.0", true}, {"v6.1.0", "v6.1.0", false}, {"v6.1.0", "v6.0.9", false},
		{"v6.9.0", "v6.10.0", true}, {"dev", "v6.1.0", false}, {"v6.1.0", "lixo", false},
		{"v6.1.0-rc1", "v6.1.0", true}, {"v6.1.0", "v6.1.0-rc1", false},
	} {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q,%q)=%v", c.cur, c.latest, got)
		}
	}
}

func TestLatestAndCache(t *testing.T) {
	dir := t.TempDir()
	run := &sys.FakeRunner{Rules: []sys.Rule{{Match: "releases/latest", Out: "https://github.com/x/y/releases/tag/v6.2.0"}}}
	now := time.Now()
	u := &Updater{Run: run, Current: "v6.0.3", Cache: filepath.Join(dir, "c.json"), Now: func() time.Time { return now }}
	v, err := u.Check(context.Background(), false)
	if err != nil || v != "v6.2.0" {
		t.Fatal(v, err)
	}
	// dentro do TTL não consulta de novo
	now = now.Add(time.Hour)
	if v, _ := u.Check(context.Background(), false); v != "v6.2.0" || run.Ran("releases/latest") != 1 {
		t.Fatal(v, run.Ran("releases/latest"))
	}
	// passou o TTL: consulta
	now = now.Add(CacheTTL)
	u.Check(context.Background(), false)
	if run.Ran("releases/latest") != 2 {
		t.Fatal(run.Ran("releases/latest"))
	}
	// versão de desenvolvimento nunca consulta
	d := &Updater{Run: run, Current: "dev"}
	if v, _ := d.Check(context.Background(), true); v != "" || run.Ran("releases/latest") != 2 {
		t.Fatal("dev consultou")
	}
}

type fileFetcher struct{ files map[string][]byte }

func (f fileFetcher) Fetch(_ context.Context, url, dest string) error {
	return os.WriteFile(dest, f.files[url[strings.LastIndex(url, "/")+1:]], 0o644)
}

func pack(t *testing.T, body string) []byte {
	t.Helper()
	p := filepath.Join(t.TempDir(), "p.tgz")
	f, _ := os.Create(p)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "termux-setup", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
	tw.Write([]byte(body))
	tw.Close()
	gz.Close()
	f.Close()
	b, _ := os.ReadFile(p)
	return b
}

func TestApplyReplacesBinaryAndChecksSHA(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "termux-setup")
	os.WriteFile(exe, []byte("antigo"), 0o755)
	tgz := pack(t, "novo")
	sum := sha256.Sum256(tgz)
	sums := hex.EncodeToString(sum[:]) + "  termux-setup_linux_amd64.tar.gz\n"
	u := &Updater{Fetch: fileFetcher{map[string][]byte{"termux-setup_linux_amd64.tar.gz": tgz, "checksums.txt": []byte(sums)}},
		Work: t.TempDir(), Exe: exe, Asset: "linux_amd64", Cache: filepath.Join(dir, "c.json")}
	if err := u.Apply(context.Background(), "v6.2.0", nil); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "novo" {
		t.Fatalf("binário: %q", b)
	}
	if _, err := os.Stat(exe + ".novo"); err == nil {
		t.Fatal("sobrou .novo")
	}

	// SHA errado: o programa antigo continua intacto
	os.WriteFile(exe, []byte("antigo"), 0o755)
	u.Fetch = fileFetcher{map[string][]byte{"termux-setup_linux_amd64.tar.gz": tgz, "checksums.txt": []byte(strings.Repeat("0", 64) + "  termux-setup_linux_amd64.tar.gz\n")}}
	if err := u.Apply(context.Background(), "v6.2.0", nil); err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "antigo" {
		t.Fatalf("binário mudou apesar do erro: %q", b)
	}
}
