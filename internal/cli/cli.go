// Package cli são os comandos de linha de comando (Cobra). Sem comando,
// abre a interface; com comando, faz direto (com --sim, sem perguntas).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/tools"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui"
)

type flags struct {
	dry, auto, noAnim                                     bool
	legacyTudo, legacyDev, legacyIA, legacyUp, legacyUndo bool
}

// ExitError carrega o código de saída.
type ExitError struct{ Code int }

func (e ExitError) Error() string { return fmt.Sprintf("saída %d", e.Code) }

// Execute roda a CLI e devolve o código de saída.
func Execute(args []string, stdout, stderr io.Writer) int {
	cmd := New(stdout, stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		var ee ExitError
		if errors.As(err, &ee) {
			return ee.Code
		}
		fmt.Fprintln(stderr, "erro:", err)
		return 1
	}
	return 0
}

// New monta a árvore de comandos.
func New(stdout, stderr io.Writer) *cobra.Command {
	f := &flags{}
	root := &cobra.Command{
		Use:   "termux-setup",
		Short: "Configura o Termux do zero: linguagens, Ubuntu, IAs e aparência",
		Long: `termux-setup configura o Termux do zero.

Sem comando abre a interface. Os comandos abaixo fazem direto;
com --sim não fazem nenhuma pergunta.`,
		Version:               app.Version,
		DisableFlagsInUseLine: true,
		SilenceUsage:          true,
		SilenceErrors:         true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch { // compatibilidade com as opções do script antigo
			case f.legacyTudo:
				return installPreset(cmd, f, "tudo")
			case f.legacyDev:
				return installPreset(cmd, f, "dev")
			case f.legacyIA:
				return installPreset(cmd, f, "ia")
			case f.legacyUp:
				return runUpdate(cmd, f, false)
			case f.legacyUndo:
				return runFixed(cmd, f, plan.Undo())
			}
			s, err := services(f)
			if err != nil {
				return err
			}
			defer s.Close()
			if !isTTY() {
				return cmd.Help()
			}
			return runTUI(s, ui.NewIntro(ui.MainMenu()), false, cmd.OutOrStdout())
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetVersionTemplate("termux-setup {{.Version}}\n")
	root.SetUsageTemplate(usagePT)
	root.CompletionOptions.HiddenDefaultCmd = true
	root.PersistentFlags().BoolP("help", "h", false, "mostra esta ajuda")
	root.Flags().BoolP("version", "v", false, "mostra a versão")
	root.SetHelpCommand(&cobra.Command{Use: "help [comando]", Short: "Ajuda sobre um comando", Hidden: true,
		Run: func(c *cobra.Command, args []string) {
			if sub, _, err := c.Root().Find(args); err == nil && sub != nil {
				_ = sub.Help()
				return
			}
			_ = c.Root().Help()
		}})
	pf := root.PersistentFlags()
	pf.BoolVar(&f.dry, "teste", false, "simulação: não instala nada (Termux de mentira em pasta temporária)")
	pf.BoolVarP(&f.auto, "sim", "y", false, "não faz perguntas (usa os padrões)")
	pf.BoolVar(&f.noAnim, "sem-animacao", false, "desliga as animações")
	for _, l := range []struct {
		p    *bool
		name string
	}{{&f.legacyTudo, "tudo"}, {&f.legacyDev, "dev"}, {&f.legacyIA, "ia"}, {&f.legacyUp, "atualizar"}, {&f.legacyUndo, "desfazer"}} {
		root.Flags().BoolVar(l.p, l.name, false, "")
		_ = root.Flags().MarkHidden(l.name)
	}

	preset := func(name, short string) *cobra.Command {
		return &cobra.Command{Use: name, Short: short, Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return installPreset(cmd, f, name) }}
	}
	root.AddCommand(
		preset("tudo", "Instalação completa (tudo + Ubuntu + IAs)"),
		preset("dev", "Dev padrão: linguagens, ferramentas e aparência"),
		preset("ia", "Ubuntu (+ ferramentas) + Claude Code, Codex, Gemini e OpenCode"),
		&cobra.Command{Use: "instalar <módulo>...", Short: "Instala módulos específicos (veja: termux-setup modulos)",
			Args: cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error { return install(cmd, f, args) }},
		&cobra.Command{Use: "atualizar", Short: "Atualiza Termux, Ubuntu e IAs", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runUpdate(cmd, f, false) }},
		&cobra.Command{Use: "ubuntu-atualizar", Short: "Atualiza só o Ubuntu e as IAs", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runUpdate(cmd, f, true) }},
		&cobra.Command{Use: "reparar", Short: "Conserta pacotes quebrados", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runFixed(cmd, f, plan.Repair()) }},
		&cobra.Command{Use: "desfazer", Short: "Volta espelho, aparência e .bashrc ao original", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runFixed(cmd, f, plan.Undo()) }},
		&cobra.Command{Use: "verificar", Short: "Mostra as versões instaladas", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return verify(cmd, f) }},
		&cobra.Command{Use: "doctor", Short: "Diagnostica o ambiente e diz como corrigir (sai com 1 se houver problema)", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return doctor(cmd, f) }},
		&cobra.Command{Use: "atualizar-app", Short: "Atualiza o próprio termux-setup (procura versão nova no GitHub)", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return selfUpdate(cmd, f) }},
		&cobra.Command{Use: "modulos", Aliases: []string{"listar"}, Short: "Lista os módulos do catálogo", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return listModules(cmd, f) }},
	)
	return root
}

const usagePT = `Uso:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if gt (len .Aliases) 0}}

Apelidos:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

Comandos:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Opções:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Opções globais:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [comando] --help" para mais detalhes.{{end}}
`

func isTTY() bool { return term.IsTerminal(os.Stdout.Fd()) && term.IsTerminal(os.Stdin.Fd()) }

func services(f *flags) (*app.Services, error) {
	if !f.dry && !sys.InTermux() {
		return nil, errors.New("este programa é só para o Termux (Android). Use --teste para simular")
	}
	return app.New(app.Config{DryRun: f.dry, NoAnim: f.noAnim || f.auto})
}

func installPreset(cmd *cobra.Command, f *flags, preset string) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	return doInstall(cmd, f, s, s.Cat.Preset(preset))
}

func install(cmd *cobra.Command, f *flags, ids []string) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	for _, id := range ids {
		if _, ok := s.Cat.Get(id); !ok {
			return fmt.Errorf("módulo desconhecido: %q (veja: termux-setup modulos)", id)
		}
	}
	return doInstall(cmd, f, s, ids)
}

func doInstall(cmd *cobra.Command, f *flags, s *app.Services, ids []string) error {
	if isTTY() {
		var first ui.Screen = ui.NewOptions(ids)
		if f.auto {
			p, err := plan.Install(s.Cat, ids, plan.Options{KeysPreset: 1}, s.Info.Arch, s.Info.SizeFactor())
			if err != nil {
				return err
			}
			first = ui.NewInstall(p)
		}
		return runTUI(s, first, f.auto, cmd.OutOrStdout())
	}
	if !f.auto {
		return errors.New("sem terminal interativo: use --sim")
	}
	p, err := plan.Install(s.Cat, ids, plan.Options{KeysPreset: 1}, s.Info.Arch, s.Info.SizeFactor())
	if err != nil {
		return err
	}
	return runText(cmd, s, p)
}

func runUpdate(cmd *cobra.Command, f *flags, ubuntuOnly bool) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	has := s.HasUbuntu(context.Background())
	p := plan.Update(has)
	if ubuntuOnly {
		if !has {
			return errors.New("Ubuntu não instalado")
		}
		p = plan.Ubuntu(true, true)
	}
	return runPlan(cmd, f, s, p)
}

func runFixed(cmd *cobra.Command, f *flags, p *plan.Plan) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	return runPlan(cmd, f, s, p)
}

func runPlan(cmd *cobra.Command, f *flags, s *app.Services, p *plan.Plan) error {
	if isTTY() {
		var first ui.Screen = ui.NewConfirm(p, nil)
		if f.auto {
			first = ui.NewInstall(p)
		}
		return runTUI(s, first, f.auto, cmd.OutOrStdout())
	}
	if !f.auto {
		return errors.New("sem terminal interativo: use --sim")
	}
	return runText(cmd, s, p)
}

// runText executa sem interface (saída redirecionada, CI, scripts).
func runText(cmd *cobra.Command, s *app.Services, p *plan.Plan) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	r := s.Engine().Run(ctx, p, &engine.TextReporter{W: cmd.OutOrStdout()}, engine.Notifier{Run: s.Runner})
	switch r.Overall {
	case engine.Complete:
		return nil
	case engine.WithFailures:
		return ExitError{1}
	default:
		return ExitError{2}
	}
}

func runTUI(s *app.Services, first ui.Screen, auto bool, out io.Writer) error {
	m := ui.New(s, first, auto)
	_, err := tea.NewProgram(m).Run()
	// Saída limpa (como nos CLIs de IA): a tela "normal" do terminal pode
	// ter sido reorganizada enquanto o app estava aberto (teclado, rotação).
	// Reseta cores, mostra o cursor, limpa a tela e volta ao topo.
	fmt.Fprint(out, "\x1b[0m\x1b[?25h\x1b[2J\x1b[H")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, " ◈ termux-setup · logs em "+s.Env.Tilde(s.Env.LogDir))
	return nil
}

func verify(cmd *cobra.Command, f *flags) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	out := cmd.OutOrStdout()
	where := ""
	s.Verify(context.Background(), func(r app.VerifyRow) {
		if r.Where != where {
			where = r.Where
			fmt.Fprintf(out, "== %s ==\n", where)
		}
		v := r.Version
		if v == "" {
			v = "não instalado"
		}
		fmt.Fprintf(out, "  %-9s %s\n", r.Name, v)
	})
	return nil
}

func listModules(cmd *cobra.Command, f *flags) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	out := cmd.OutOrStdout()
	for _, g := range s.Cat.Groups() {
		fmt.Fprintf(out, "%s\n", g.Title)
		for _, m := range s.Cat.Visible() {
			if m.Group != g.ID {
				continue
			}
			extra := ""
			if len(m.Requires) > 0 {
				extra = " (precisa: " + strings.Join(m.Requires, ", ") + ")"
			}
			fmt.Fprintf(out, "  %-8s %s%s\n", m.ID, m.Name, extra)
		}
	}
	return nil
}

// selfUpdate procura e instala a versão nova do próprio programa.
func selfUpdate(cmd *cobra.Command, f *flags) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	out := cmd.OutOrStdout()
	ctx := context.Background()
	v, err := s.Updater.Check(ctx, true)
	if err != nil {
		return fmt.Errorf("não consegui consultar o GitHub: %w", err)
	}
	if v == "" {
		fmt.Fprintf(out, "Você já está na versão mais nova (%s).\n", s.Updater.Current)
		return nil
	}
	fmt.Fprintf(out, "Nova versão: %s (você usa %s)\n", v, s.Updater.Current)
	if err := s.Updater.Apply(ctx, v, func(m string) { fmt.Fprintln(out, " ", m) }); err != nil {
		return err
	}
	fmt.Fprintf(out, "Atualizado para %s. Abra o termux-setup de novo.\n", v)
	return nil
}

// doctor roda as verificações em texto; sai com código 1 se algo falhar.
func doctor(cmd *cobra.Command, f *flags) error {
	s, err := services(f)
	if err != nil {
		return err
	}
	defer s.Close()
	out := cmd.OutOrStdout()
	fails := 0
	s.Tools.Doctor(context.Background(), func(c tools.Check) {
		mark := "✓"
		switch c.Status {
		case tools.Warn:
			mark = "!"
		case tools.Fail:
			mark, fails = "✗", fails+1
		}
		fmt.Fprintf(out, "%s %s  %s\n", mark, c.Name, c.Detail)
		if c.Fix != "" {
			fmt.Fprintf(out, "    → %s\n", c.Fix)
		}
	})
	if fails > 0 {
		return ExitError{Code: 1}
	}
	return nil
}
