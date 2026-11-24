// Package update procura versões novas do termux-setup no GitHub e instala
// a nova por cima do próprio programa (como os CLIs oficiais fazem).
//
// A rede passa pelo curl do Termux (sys.Runner/Fetcher) pelo mesmo motivo
// do resto do app: binário Go estático no Android não acha DNS/certificados.
package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// Repo é onde ficam as releases.
const Repo = "Rafael-devR/termux-setup.sh"

// CacheTTL é o intervalo mínimo entre duas consultas ao GitHub.
const CacheTTL = 12 * time.Hour

// Updater junta o que a checagem e a instalação precisam.
type Updater struct {
	Run     sys.Runner
	Fetch   sys.Fetcher
	Current string // versão em uso ("dev" = compilado na mão, nunca atualiza)
	Cache   string // arquivo do cache (JSON)
	Work    string // pasta temporária para os downloads
	Exe     string // caminho do programa a trocar
	Asset   string // ex.: "android_arm64"
	Repo    string // padrão: Repo
	Fake    string // modo --teste: finge que esta é a versão mais nova
	DryRun  bool   // modo --teste: Apply só simula
	Now     func() time.Time
}

type cache struct {
	Checked time.Time `json:"checked"`
	Latest  string    `json:"latest"`
}

func (u *Updater) repo() string {
	if u.Repo != "" {
		return u.Repo
	}
	return Repo
}

func (u *Updater) now() time.Time {
	if u.Now != nil {
		return u.Now()
	}
	return time.Now()
}

// Latest descobre a versão mais nova seguindo o redirecionamento de
// /releases/latest (não conta no limite da API do GitHub).
func (u *Updater) Latest(ctx context.Context) (string, error) {
	url := "https://github.com/" + u.repo() + "/releases/latest"
	out, err := sys.Output(ctx, u.Run, sys.C("curl", "-fsSL", "--max-time", "10", "-o", "/dev/null", "-w", "%{url_effective}", url))
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	i := strings.LastIndex(out, "/releases/tag/")
	if i < 0 {
		return "", fmt.Errorf("resposta inesperada do GitHub: %q", out)
	}
	v := out[i+len("/releases/tag/"):]
	if _, ok := parse(v); !ok {
		return "", fmt.Errorf("versão inválida: %q", v)
	}
	return v, nil
}

// Check devolve a versão nova, se houver ("" = está em dia). Consulta o
// GitHub no máximo uma vez a cada CacheTTL (force ignora o cache).
func (u *Updater) Check(ctx context.Context, force bool) (string, error) {
	if u.Fake != "" {
		if Newer(u.Current, u.Fake) {
			return u.Fake, nil
		}
		return "", nil
	}
	if _, ok := parse(u.Current); !ok {
		return "", nil // "dev": nada a atualizar
	}
	latest := ""
	if c, ok := u.readCache(); ok && !force && u.now().Sub(c.Checked) < CacheTTL {
		latest = c.Latest
	} else {
		v, err := u.Latest(ctx)
		if err != nil {
			return "", err
		}
		latest = v
		u.writeCache(cache{Checked: u.now(), Latest: v})
	}
	if Newer(u.Current, latest) {
		return latest, nil
	}
	return "", nil
}

func (u *Updater) readCache() (cache, bool) {
	var c cache
	b, err := os.ReadFile(u.Cache)
	if err != nil || json.Unmarshal(b, &c) != nil || c.Latest == "" {
		return c, false
	}
	return c, true
}

func (u *Updater) writeCache(c cache) {
	if u.Cache == "" {
		return
	}
	b, _ := json.Marshal(c)
	_ = os.MkdirAll(filepath.Dir(u.Cache), 0o755)
	_ = os.WriteFile(u.Cache, b, 0o644)
}

// Apply baixa a versão v, confere o SHA256 e troca o programa em Exe.
// A troca é atômica (rename): se algo falhar antes, o programa antigo
// continua intacto.
func (u *Updater) Apply(ctx context.Context, v string, say func(string)) error {
	if say == nil {
		say = func(string) {}
	}
	if u.DryRun {
		say("baixando " + v + "… (teste)")
		time.Sleep(600 * time.Millisecond)
		say("conferindo SHA256… (teste)")
		time.Sleep(400 * time.Millisecond)
		return nil
	}
	if u.Exe == "" || u.Asset == "" {
		return errors.New("não sei qual programa atualizar")
	}
	base := "https://github.com/" + u.repo() + "/releases/download/" + v
	file := "termux-setup_" + u.Asset + ".tar.gz"
	tgz, sums := filepath.Join(u.Work, file), filepath.Join(u.Work, "checksums.txt")

	say("baixando " + v + "…")
	if err := u.Fetch.Fetch(ctx, base+"/checksums.txt", sums); err != nil {
		return fmt.Errorf("baixar checksums: %w", err)
	}
	if err := u.Fetch.Fetch(ctx, base+"/"+file, tgz); err != nil {
		return fmt.Errorf("baixar %s: %w", file, err)
	}
	say("conferindo SHA256…")
	if err := verifySum(sums, tgz, file); err != nil {
		return err
	}
	say("instalando…")
	tmp := u.Exe + ".novo"
	if err := extract(tgz, "termux-setup", tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, u.Exe); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("trocar o programa: %w", err)
	}
	_ = os.Remove(u.Cache) // a próxima checagem enxerga a versão nova
	return nil
}

func verifySum(sums, file, name string) error {
	b, err := os.ReadFile(sums)
	if err != nil {
		return err
	}
	want := ""
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			want = f[0]
		}
	}
	if want == "" {
		return fmt.Errorf("%s não está no checksums.txt", name)
	}
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return errors.New("SHA256 não confere: download corrompido")
	}
	return nil
}

// extract tira o arquivo name de um .tar.gz e grava em dest.
func extract(tgz, name, dest string) error {
	f, err := os.Open(tgz)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s não está no pacote", name)
		}
		if err != nil {
			return err
		}
		if filepath.Base(h.Name) != name || h.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil { //nolint:gosec // pacote próprio, já conferido por SHA256
			out.Close()
			return err
		}
		return out.Close()
	}
}

// Newer diz se latest é mais novo que cur (vX.Y.Z; sufixos como -rc1 contam
// como mais antigos que a versão final).
func Newer(cur, latest string) bool {
	a, ok1 := parse(cur)
	b, ok2 := parse(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return b[i] > a[i]
		}
	}
	return a[3] == 1 && b[3] == 0 // cur é pré-release e latest é final
}

// parse devolve [maior, menor, correção, pre] (pre=1 se tiver sufixo).
func parse(v string) ([4]int, bool) {
	var r [4]int
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v, r[3] = v[:i], 1
	}
	p := strings.Split(v, ".")
	if len(p) != 3 {
		return r, false
	}
	for i, s := range p {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return r, false
		}
		r[i] = n
	}
	return r, true
}
