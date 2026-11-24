package sys

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Cmd descreve um comando externo.
type Cmd struct {
	Name  string
	Args  []string
	Stdin string   // texto enviado na entrada (ex.: token do GitHub)
	Env   []string // variáveis extras ("CHAVE=valor")
	Dir   string
}

// C cria um Cmd.
func C(name string, args ...string) Cmd { return Cmd{Name: name, Args: args} }

// String mostra o comando como seria digitado.
func (c Cmd) String() string { return strings.TrimSpace(c.Name + " " + strings.Join(c.Args, " ")) }

// Runner executa comandos. Existem três: o real, o do modo --teste
// (simula tudo) e o falso dos testes automáticos (respostas programadas).
type Runner interface {
	// Run executa e manda stdout+stderr para out (pode ser nil).
	Run(ctx context.Context, c Cmd, out io.Writer) error
	// Has diz se o programa existe no PATH.
	Has(name string) bool
}

// ExitError é um comando que terminou com código diferente de zero.
type ExitError struct {
	Cmd  string
	Code int
}

func (e *ExitError) Error() string { return fmt.Sprintf("%s: saiu com código %d", e.Cmd, e.Code) }

// Output roda o comando e devolve a saída como texto.
func Output(ctx context.Context, r Runner, c Cmd) (string, error) {
	var b bytes.Buffer
	err := r.Run(ctx, c, &b)
	return b.String(), err
}

// OK roda o comando descartando a saída e diz se deu certo.
func OK(ctx context.Context, r Runner, c Cmd) bool { return r.Run(ctx, c, nil) == nil }

// ---------------------------------------------------------------- real

// ExecRunner roda comandos de verdade. Cada comando vira líder do próprio
// grupo de processos: ao cancelar (Ctrl+C), o grupo inteiro é encerrado
// (apt, dpkg e os filhos deles), não só o processo principal.
type ExecRunner struct{}

func (ExecRunner) Has(name string) bool { _, err := exec.LookPath(name); return err == nil }

func (ExecRunner) Run(ctx context.Context, c Cmd, out io.Writer) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Env = append(cmd.Env, c.Env...)
	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	}
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	cmd.WaitDelay = 3 * time.Second
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) && ctx.Err() == nil {
		return &ExitError{Cmd: c.Name, Code: ee.ExitCode()}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// ---------------------------------------------------------------- --teste

// DryRunner é o modo --teste: escreve o que faria, espera um pouco (para a
// interface ter o que animar) e finge sucesso. TSETUP_FALHAR=texto faz
// falhar qualquer comando que contenha esse texto.
type DryRunner struct {
	Fail        string        // simula falha nos comandos que contêm isto
	Unavailable []string      // pacotes que "não existem" no espelho
	Delay       time.Duration // espera média por comando (0 = 300ms)
}

func (DryRunner) Has(string) bool { return true }

func (d DryRunner) Run(ctx context.Context, c Cmd, out io.Writer) error {
	line := c.String()
	if c.Name == "apt-cache" && len(c.Args) > 1 {
		for _, p := range d.Unavailable {
			if c.Args[1] == p {
				return &ExitError{Cmd: c.Name, Code: 100}
			}
		}
		return nil
	}
	if out != nil {
		short := line
		if len(short) > 90 {
			short = short[:90] + "…"
		}
		fmt.Fprintf(out, "[teste] %s\n", strings.ReplaceAll(short, "\n", " "))
	}
	delay := d.Delay
	if delay == 0 {
		delay = 300 * time.Millisecond
	}
	wait := delay/2 + time.Duration(rand.Int64N(int64(delay)))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
	}
	if d.Fail != "" && strings.Contains(line, d.Fail) {
		if out != nil {
			fmt.Fprintln(out, "E: falha simulada")
		}
		return &ExitError{Cmd: c.Name, Code: 100}
	}
	return nil
}

// ---------------------------------------------------------------- testes

// Rule é uma resposta programada do FakeRunner.
type Rule struct {
	Match string // trecho do comando ("apt-get install")
	Out   string
	Code  int
}

// FakeRunner é para os testes automáticos: guarda todos os comandos e
// responde conforme as regras (a primeira que casar). Sem regra = sucesso.
type FakeRunner struct {
	mu      sync.Mutex
	Rules   []Rule
	Missing []string // programas que Has() diz não existir
	Calls   []Cmd
}

func (f *FakeRunner) Has(name string) bool {
	for _, m := range f.Missing {
		if m == name {
			return false
		}
	}
	return true
}

func (f *FakeRunner) Run(_ context.Context, c Cmd, out io.Writer) error {
	f.mu.Lock()
	f.Calls = append(f.Calls, c)
	f.mu.Unlock()
	line := c.String()
	for _, r := range f.Rules {
		if strings.Contains(line, r.Match) {
			if out != nil && r.Out != "" {
				io.WriteString(out, r.Out)
			}
			if r.Code != 0 {
				return &ExitError{Cmd: c.Name, Code: r.Code}
			}
			return nil
		}
	}
	return nil
}

// Ran diz quantas chamadas contêm o trecho.
func (f *FakeRunner) Ran(sub string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.Calls {
		if strings.Contains(c.String(), sub) {
			n++
		}
	}
	return n
}
