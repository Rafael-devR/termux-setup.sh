// Package apt instala pacotes do Termux com tolerância a falhas:
// alternativas ("a|b"), pacotes que não existem no espelho e queda para
// instalação um a um quando o bloco falha.
package apt

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// Opts são as opções usadas em todo apt-get (sem perguntas, com retry).
var Opts = []string{"-y",
	"-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold",
	"-o", "Acquire::Retries=3", "-o", "Acquire::http::Timeout=25", "-o", "Acquire::https::Timeout=25"}

// Get roda apt-get <sub> com as opções padrão.
func Get(ctx context.Context, s *engine.StepCtx, sub string, args ...string) error {
	all := append([]string{sub}, Opts...)
	return s.Exec(ctx, "apt-get", append(all, args...)...)
}

// Pick devolve a primeira alternativa de "a|b" que existe no espelho.
func Pick(ctx context.Context, r sys.Runner, token string) (string, bool) {
	for _, alt := range strings.Split(token, "|") {
		if alt = strings.TrimSpace(alt); alt != "" && sys.OK(ctx, r, sys.C("apt-cache", "show", alt)) {
			return alt, true
		}
	}
	return "", false
}

// Install instala os pacotes. Os indisponíveis são pulados (e anotados);
// se o bloco falhar, tenta um a um e anota os que falharem.
func Install(ctx context.Context, s *engine.StepCtx, tokens []string) error {
	var list []string
	for _, t := range tokens {
		if p, ok := Pick(ctx, s.Run, t); ok {
			list = append(list, p)
		} else {
			s.Unavailable(t)
			s.Logf("[aviso] indisponível: %s", t)
		}
	}
	if len(list) == 0 {
		return nil
	}
	s.Logf("Instalando: %s", strings.Join(list, " "))
	if Get(ctx, s, "install", list...) == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.Logf("[aviso] instalação em bloco falhou; tentando um a um")
	var failed []string
	for _, p := range list {
		if err := Get(ctx, s, "install", p); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.FailedPkg(p)
			failed = append(failed, p)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("falharam: %s", strings.Join(failed, " "))
	}
	return nil
}

// Repair conserta o dpkg/apt e atualiza tudo.
func Repair(ctx context.Context, s *engine.StepCtx) error {
	_ = s.Exec(ctx, "dpkg", "--configure", "-a")
	_ = Get(ctx, s, "--fix-broken", "install")
	if err := Get(ctx, s, "update"); err != nil {
		return err
	}
	return Get(ctx, s, "full-upgrade")
}
