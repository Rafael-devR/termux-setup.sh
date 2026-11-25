package tools

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Itens (relativos à casa) que entram no backup de configuração.
var backupPaths = []string{".bashrc", ".termux", "bin", ".gitconfig"}

// Backup grava ~/termux-setup-logs/config-AAAAMMDD-HHMMSS.tar.gz.
func (t *Toolbox) Backup() (string, error) {
	_ = os.MkdirAll(t.Env.LogDir, 0o755)
	dest := filepath.Join(t.Env.LogDir, "config-"+time.Now().Format("20060102-150405")+".tar.gz")
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	n := 0
	for _, rel := range backupPaths {
		root := filepath.Join(t.Env.Home, rel)
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || !(info.Mode().IsRegular() || info.IsDir()) {
				return nil
			}
			name, _ := filepath.Rel(t.Env.Home, p)
			h, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return nil
			}
			h.Name = name
			if err := tw.WriteHeader(h); err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				src, err := os.Open(p)
				if err != nil {
					return nil
				}
				defer src.Close()
				if _, err := io.Copy(tw, src); err != nil {
					return err
				}
				n++
			}
			return nil
		})
		if err != nil {
			f.Close()
			os.Remove(dest)
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if n == 0 {
		os.Remove(dest)
		return "", errors.New("não há configuração para guardar ainda")
	}
	return dest, nil
}

// Backups lista os backups, do mais novo para o mais antigo.
func (t *Toolbox) Backups() []string {
	files, _ := filepath.Glob(filepath.Join(t.Env.LogDir, "config-*.tar.gz"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files
}

// Restore devolve os arquivos de um backup para a casa. Só aceita caminhos
// dentro da casa (nada de "../").
func (t *Toolbox) Restore(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	tr := tar.NewReader(gz)
	n := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		clean := filepath.Clean(h.Name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return n, fmt.Errorf("caminho suspeito no backup: %q", h.Name)
		}
		dest := filepath.Join(t.Env.Home, clean)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return n, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return n, err
			}
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o777)
			if err != nil {
				return n, err
			}
			if _, err := io.Copy(out, tr); err != nil { //nolint:gosec // backup nosso, caminhos validados acima
				out.Close()
				return n, err
			}
			out.Close()
			n++
		}
	}
}
