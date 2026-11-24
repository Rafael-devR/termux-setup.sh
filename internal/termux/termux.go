// Package termux mexe nos arquivos do Termux: ~/.bashrc, ~/bin,
// ~/.termux (tema, fonte, barra de teclas) e os backups para desfazer.
package termux

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

//go:embed assets
var assets embed.FS

func asset(name string) []byte {
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		panic("asset ausente: " + name) // erro de build, coberto pelos testes
	}
	return b
}

// Fonte JetBrains Mono (versão e SHA256 conferido do zip oficial).
const (
	FontVersion = "2.304"
	FontURL     = "https://github.com/JetBrains/JetBrainsMono/releases/download/v" + FontVersion + "/JetBrainsMono-" + FontVersion + ".zip"
	FontSHA256  = "6f6376c6ed2960ea8a963cd7387ec9d76e3f629125bc33d1fdcd7eb7012f7bbf"
)

const (
	blockStart = "# >>> termux-setup >>>"
	blockEnd   = "# <<< termux-setup <<<"
)

// ---------------------------------------------------------------- .bashrc

// RemoveBlock tira o bloco do termux-setup de um texto.
func RemoveBlock(text string) string {
	var out []string
	in := false
	for _, l := range strings.Split(text, "\n") {
		switch {
		case strings.TrimSpace(l) == blockStart:
			in = true
		case strings.TrimSpace(l) == blockEnd:
			in = false
		case !in:
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// EnsureBashrc escreve (ou reescreve) o bloco do termux-setup no ~/.bashrc
// sem tocar no resto do arquivo. Rodar várias vezes dá o mesmo resultado.
func EnsureBashrc(home string) error {
	p := filepath.Join(home, ".bashrc")
	old, _ := os.ReadFile(p)
	text := strings.TrimRight(RemoveBlock(string(old)), "\n")
	if text != "" {
		text += "\n"
	}
	return os.WriteFile(p, []byte(text+string(asset("bashrc-block.sh"))), 0o644)
}

// UndoBashrc remove o bloco do ~/.bashrc.
func UndoBashrc(home string) error {
	p := filepath.Join(home, ".bashrc")
	old, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(RemoveBlock(string(old))), 0o644)
}

// ---------------------------------------------------------------- ~/bin

func writeExec(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o755)
}

// WriteCommands cria atualizar, limpar, info e novo em ~/bin.
func WriteCommands(env *sys.Env) error {
	if err := os.MkdirAll(filepath.Join(env.Home, "projetos"), 0o755); err != nil {
		return err
	}
	for _, n := range []string{"atualizar", "limpar", "info", "novo"} {
		if err := writeExec(env.BinDir(), n, asset("bin/"+n)); err != nil {
			return err
		}
	}
	return nil
}

// UbuntuNames são o nome da distro e do usuário comum do Ubuntu.
type UbuntuNames struct{ Distro, User string }

// WriteUbuntuCommands cria ia-run, ia-update, claude, codex, gemini,
// ubuntu e ubuntu-root em ~/bin.
func WriteUbuntuCommands(env *sys.Env, n UbuntuNames) error {
	var b bytes.Buffer
	if err := template.Must(template.New("ia-run").Parse(string(asset("bin/ia-run.tmpl")))).Execute(&b, n); err != nil {
		return err
	}
	files := map[string]string{
		"ia-run":      b.String(),
		"ia-update":   string(asset("bin/ia-update")),
		"ubuntu":      shebang + "exec \"$HOME/bin/ia-run\" bash -l\n",
		"ubuntu-root": shebang + "exec proot-distro login " + n.Distro + "\n",
	}
	for _, c := range []string{"claude", "codex", "gemini"} {
		files[c] = shebang + "exec \"$HOME/bin/ia-run\" " + c + " \"$@\"\n"
	}
	for name, body := range files {
		if err := writeExec(env.BinDir(), name, []byte(body)); err != nil {
			return err
		}
	}
	return nil
}

const shebang = "#!/data/data/com.termux/files/usr/bin/bash\n"

// ---------------------------------------------------------------- aparência

// BackupTermuxOnce copia ~/.termux para os logs na 1ª mudança da execução
// (as seguintes não sobrescrevem o original). Se ~/.termux não existia,
// grava a marca .vazio para o desfazer saber.
func BackupTermuxOnce(env *sys.Env, stamp string) error {
	bk := filepath.Join(env.LogDir, "backup-termux-"+stamp)
	if _, err := os.Stat(bk); err == nil {
		return nil
	}
	if err := os.MkdirAll(bk, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(env.TermuxDir()); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(filepath.Join(bk, ".vazio"), nil, 0o644)
	}
	return os.CopyFS(bk, os.DirFS(env.TermuxDir()))
}

// Hush cria ~/.hushlogin (some a mensagem de boas-vindas).
func Hush(env *sys.Env) error { return os.WriteFile(filepath.Join(env.Home, ".hushlogin"), nil, 0o644) }

// WriteTheme aplica as cores do VS Code.
func WriteTheme(env *sys.Env, stamp string) error {
	if err := BackupTermuxOnce(env, stamp); err != nil {
		return err
	}
	if err := os.MkdirAll(env.TermuxDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(env.TermuxDir(), "colors.properties"), asset("colors.properties"), 0o644)
}

var extraKeysRe = regexp.MustCompile(`^\s*extra-keys\s*=`)

// SetKeys troca só a linha extra-keys do termux.properties. Preset "3"
// remove a linha (volta ao padrão do Termux).
func SetKeys(env *sys.Env, stamp, preset string) error {
	if err := BackupTermuxOnce(env, stamp); err != nil {
		return err
	}
	if err := os.MkdirAll(env.TermuxDir(), 0o755); err != nil {
		return err
	}
	return setKeysFile(filepath.Join(env.TermuxDir(), "termux.properties"), preset)
}

func setKeysFile(p, preset string) error {
	old, _ := os.ReadFile(p)
	var keep []string
	for _, l := range strings.Split(strings.TrimRight(string(old), "\n"), "\n") {
		if l != "" && !extraKeysRe.MatchString(l) {
			keep = append(keep, l)
		}
	}
	if preset == "1" || preset == "2" {
		keep = append(keep, strings.TrimRight(string(asset("keys-"+preset+".txt")), "\n"))
	}
	text := strings.Join(keep, "\n")
	if text != "" {
		text += "\n"
	}
	return os.WriteFile(p, []byte(text), 0o644)
}

// InstallFont baixa a JetBrains Mono, confere o SHA256 (exceto no modo
// teste) e grava a Regular como ~/.termux/font.ttf.
func InstallFont(ctx context.Context, env *sys.Env, f sys.Fetcher, stamp string, log io.Writer) error {
	zipPath := filepath.Join(env.Work, "jb.zip")
	if err := f.Fetch(ctx, FontURL, zipPath); err != nil {
		return fmt.Errorf("download da fonte falhou: %w", err)
	}
	if !env.DryRun {
		sum, err := fileSHA256(zipPath)
		if err != nil {
			return err
		}
		if sum != FontSHA256 {
			return fmt.Errorf("SHA256 da fonte não confere (esperado %s, veio %s)", FontSHA256, sum)
		}
		fmt.Fprintln(log, "SHA256 ok")
	}
	data, err := extractFromZip(zipPath, "JetBrainsMono-Regular.ttf")
	if err != nil {
		return err
	}
	if err := BackupTermuxOnce(env, stamp); err != nil {
		return err
	}
	if err := os.MkdirAll(env.TermuxDir(), 0o755); err != nil {
		return err
	}
	fmt.Fprintf(log, "Fonte JetBrains Mono %s instalada\n", FontVersion)
	return os.WriteFile(filepath.Join(env.TermuxDir(), "font.ttf"), data, 0o644)
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractFromZip(zipPath, base string) ([]byte, error) {
	z, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("zip inválido: %w", err)
	}
	defer z.Close()
	for _, f := range z.File {
		if path.Base(f.Name) == base {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, 20<<20))
		}
	}
	return nil, fmt.Errorf("%s não encontrado no zip", base)
}

// ---------------------------------------------------------------- desfazer

// Oldest devolve o backup mais antigo com o prefixo (= estado original).
func Oldest(env *sys.Env, prefix string) string {
	m, _ := filepath.Glob(filepath.Join(env.LogDir, prefix+"-*"))
	sort.Strings(m) // o carimbo AAAAMMDD-HHMMSS ordena por data
	if len(m) == 0 {
		return ""
	}
	return m[0]
}

// UndoLook restaura ~/.termux do backup mais antigo e apaga ~/.hushlogin.
func UndoLook(env *sys.Env, log io.Writer) error {
	bk := Oldest(env, "backup-termux")
	if bk == "" {
		fmt.Fprintln(log, "Nenhum backup da aparência (nunca foi mudada).")
		return nil
	}
	fmt.Fprintf(log, "Restaurando aparência de: %s\n", bk)
	_ = os.Remove(filepath.Join(env.TermuxDir(), "colors.properties"))
	_ = os.Remove(filepath.Join(env.TermuxDir(), "font.ttf"))
	if _, err := os.Stat(filepath.Join(bk, ".vazio")); err == nil {
		// ~/.termux não existia: só tira a barra de teclas
		p := filepath.Join(env.TermuxDir(), "termux.properties")
		if _, err := os.Stat(p); err == nil {
			if err := setKeysFile(p, "3"); err != nil {
				return err
			}
		}
	} else {
		if err := os.MkdirAll(env.TermuxDir(), 0o755); err != nil {
			return err
		}
		if err := copyOver(bk, env.TermuxDir()); err != nil {
			return err
		}
	}
	_ = os.Remove(filepath.Join(env.Home, ".hushlogin"))
	return nil
}

// copyOver copia os arquivos de src para dst, sobrescrevendo.
func copyOver(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
