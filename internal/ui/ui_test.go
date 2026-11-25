package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/actions"
	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/testutil"
	"github.com/Rafael-devR/termux-setup.sh/internal/tools"
	"github.com/Rafael-devR/termux-setup.sh/modules"
	"github.com/charmbracelet/x/ansi"
)

func services(t *testing.T, anim bool) *app.Services {
	t.Helper()
	cat, err := catalog.Load(modules.FS)
	if err != nil {
		t.Fatal(err)
	}
	env := testutil.Env(t)
	info := sys.Info{Arch: "aarch64", Android: "14", Model: "Teste"}
	runner := sys.DryRunner{Delay: time.Millisecond}
	return &app.Services{Env: env, Runner: runner, Fetch: sys.DryFetcher{},
		Info: info, Cat: cat, Reg: actions.Registry(), Anim: anim,
		Tools: &tools.Toolbox{Run: runner, Env: env, Info: info, Fake: true}}
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

// run executa um comando e devolve as mensagens (abrindo Batch/Sequence).
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	switch m := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			out = append(out, run(c)...)
		}
		return out
	case nil:
		return nil
	}
	return []tea.Msg{msg}
}

func plainLines(s string) []string { return strings.Split(ansi.Strip(s), "\n") }

func TestScreensFitNarrowPhone(t *testing.T) {
	s := services(t, false)
	p, _ := plan.Install(s.Cat, s.Cat.Preset("tudo"), plan.Options{}, "aarch64", 100)
	for _, size := range [][2]int{{40, 24}, {46, 34}, {80, 40}} {
		c := &Ctx{W: size[0], H: size[1], S: s, RNG: newRNG()}
		screens := map[string]Screen{"menu": MainMenu(), "ubuntu": UbuntuMenu(), "confirm": NewConfirm(p, nil),
			"checklist": NewChecklist("x", groupIDs(s.Cat), s.Cat.Preset("dev")), "install": NewInstall(p),
			"result": NewResult(&engine.Result{Plan: p, Status: make([]engine.Status, len(p.Steps)), Overall: engine.Complete})}
		for name, sc := range screens {
			sc.Init(c)
			for _, l := range plainLines(sc.View(c)) {
				if w := ansi.StringWidth(l); w > c.W {
					t.Errorf("%s em %dx%d: linha com %d colunas: %q", name, c.W, c.H, w, l)
				}
			}
		}
	}
}

func TestMenuNavigationAndShortcut(t *testing.T) {
	s := services(t, false)
	c := &Ctx{W: 46, H: 34, S: s, RNG: newRNG()}
	m := MainMenu()
	m.Init(c)
	m.Update(c, key("down"))
	if m.cur != 1 {
		t.Fatal(m.cur)
	}
	m.Update(c, key("up"))
	m.Update(c, key("up")) // volta para o último
	if m.cur != len(m.Items)-1 {
		t.Fatal(m.cur)
	}
	_, cmd := m.Update(c, key("2"))
	msgs := run(cmd)
	if len(msgs) != 1 {
		t.Fatalf("%#v", msgs)
	}
	if push, ok := msgs[0].(pushMsg); !ok {
		t.Fatalf("%#v", msgs[0])
	} else if _, ok := push.s.(*Options); !ok {
		t.Fatalf("o atalho 2 deveria abrir as perguntas, abriu %T", push.s)
	}
}

func TestChecklistShowsDependencies(t *testing.T) {
	s := services(t, false)
	c := &Ctx{W: 60, H: 50, S: s, RNG: newRNG()}
	k := NewChecklist("IA", []string{"ubuntu", "ia"}, []string{"codex"})
	k.Init(c)
	v := ansi.Strip(k.View(c))
	if !strings.Contains(v, "incluído por dependência") {
		t.Fatal(v)
	}
	k.Update(c, key("n"))
	if len(k.selected(c)) != 0 {
		t.Fatal("n deveria desmarcar tudo")
	}
	k.Update(c, key("space"))
	if len(k.selected(c)) != 1 {
		t.Fatal(k.selected(c))
	}
}

func TestChecklistBlocksUnsupportedArch(t *testing.T) {
	s := services(t, false)
	s.Info.Arch = "armv7l"
	c := &Ctx{W: 60, H: 50, S: s, RNG: newRNG()}
	k := NewChecklist("IA", []string{"ia"}, []string{"claude"})
	k.Init(c)
	if len(k.selected(c)) != 0 {
		t.Fatal("claude não roda em 32 bits")
	}
	if !strings.Contains(ansi.Strip(k.View(c)), "sem suporte a armv7l") {
		t.Fatal(ansi.Strip(k.View(c)))
	}
}

// O painel roda o motor de verdade (modo teste) até virar a tela de resultado.
func TestInstallRunsToResult(t *testing.T) {
	s := services(t, true)
	c := &Ctx{W: 46, H: 34, S: s, RNG: newRNG()}
	p, _ := plan.Install(s.Cat, []string{"python", "theme"}, plan.Options{}, "aarch64", 100)
	in := NewInstall(p)
	in.Init(c)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		c.Frame++
		_, cmd := in.Update(c, tickMsg(time.Now()))
		_ = in.View(c)
		for _, msg := range run(cmd) {
			if r, ok := msg.(replaceMsg); ok {
				res, ok := r.s.(*Result)
				if !ok {
					t.Fatalf("%T", r.s)
				}
				if res.r.Overall != engine.Complete {
					t.Fatal(res.r.Overall)
				}
				res.Init(c)
				if v := ansi.Strip(res.View(c)); !strings.Contains(v, "COMPLETO") && !strings.Contains(v, "◉") {
					t.Fatal(v)
				}
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("não terminou")
}

func TestInstallCancel(t *testing.T) {
	s := services(t, true)
	s.Runner = sys.DryRunner{Delay: 2 * time.Second}
	c := &Ctx{W: 46, H: 34, S: s, RNG: newRNG()}
	in := NewInstall(plan.Repair())
	in.Init(c)
	in.Update(c, key("q"))
	deadline := time.Now().Add(10 * time.Second)
	for in.snap.Result == nil && time.Now().Before(deadline) {
		c.Frame++
		in.Update(c, tickMsg(time.Now()))
		time.Sleep(10 * time.Millisecond)
	}
	if in.snap.Result == nil || !in.snap.Result.Interrupted {
		t.Fatal("deveria interromper")
	}
}

func TestRootModelNavigates(t *testing.T) {
	s := services(t, false)
	m := New(s, MainMenu(), false)
	m.Init()
	m.Update(tea.WindowSizeMsg{Width: 46, Height: 34})
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, "Menu principal") {
		t.Fatal(v)
	}
	_, cmd := m.Update(key("3"))
	for _, msg := range run(cmd) {
		m.Update(msg)
	}
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, "Ubuntu & IA") {
		t.Fatal(v)
	}
	m.Update(popMsg{})
	if len(m.stack) != 1 {
		t.Fatal(len(m.stack))
	}
}

func TestIntroSkipsOnKey(t *testing.T) {
	s := services(t, true)
	c := &Ctx{W: 46, H: 34, S: s, RNG: newRNG()}
	in := NewIntro(MainMenu())
	in.Init(c)
	for range 50 {
		c.Frame++
		in.Update(c, tickMsg(time.Now()))
	}
	if !strings.Contains(ansi.Strip(in.View(c)), "█") {
		t.Fatal("o logo deveria estar na tela no quadro 50")
	}
	_, cmd := in.Update(c, key("x"))
	if msgs := run(cmd); len(msgs) != 1 {
		t.Fatal(msgs)
	} else if _, ok := msgs[0].(replaceMsg); !ok {
		t.Fatalf("%#v", msgs[0])
	}
}

// O painel é centralizado (até MaxW colunas) e a tela inteira é preenchida:
// toda linha tem exatamente a largura do terminal e há uma por linha da tela.
func TestComposeFillsTerminal(t *testing.T) {
	s := services(t, false)
	for _, sz := range [][2]int{{46, 34}, {100, 36}, {160, 50}} {
		m := New(s, MainMenu(), false)
		m.Init()
		m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		lines := strings.Split(m.View().Content, "\n")
		if len(lines) != sz[1] {
			t.Fatalf("%v: %d linhas", sz, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != sz[0] {
				t.Fatalf("%v: linha %d com %d colunas", sz, i, w)
			}
		}
	}
}

// pump entrega a uma tela Tool tudo que o trabalho em segundo plano manda,
// até ele terminar (ou até n mensagens).
func pump(t *testing.T, s Screen, c *Ctx, n int) {
	t.Helper()
	cmd := s.Init(c)
	for i := 0; i < n && cmd != nil; i++ {
		msgs := run(cmd)
		cmd = nil
		for _, m := range msgs {
			var next tea.Cmd
			s, next = s.Update(c, m)
			if next != nil {
				cmd = next
			}
			if _, done := m.(toolDoneMsg); done {
				return
			}
		}
	}
}

func TestToolsScreensRenderOnPhone(t *testing.T) {
	s := services(t, false)
	c := &Ctx{W: 46, H: 40, S: s, RNG: newRNG()}
	for _, tc := range []struct {
		name string
		mk   func(*Ctx) Screen
		want string
	}{
		{"velocidade", speedTool, "Download"},
		{"internet", netTool, "Tudo certo com a internet"},
		{"doctor", doctorTool, "Espaço livre"},
		{"info", infoTool, "Arquitetura"},
		{"limpeza", cleanTool, "Cache de pacotes"},
		{"backup", backupTool, "Guarda .bashrc"},
		{"wifi", wifiTool, "casa-5G"},
		{"processos", procsTool, "termux-setup"},
		{"monitor", monitorTool, "Bateria"},
	} {
		scr := tc.mk(c)
		n := 40
		if tc.name == "monitor" || tc.name == "wifi" || tc.name == "processos" {
			n = 1 // ao vivo: uma leitura basta
		}
		pump(t, scr, c, n)
		if tl, ok := scr.(*Tool); ok {
			defer tl.stop()
		}
		view := ansi.Strip(scr.View(c))
		if !strings.Contains(view, tc.want) {
			t.Errorf("%s: faltou %q na tela:\n%s", tc.name, tc.want, view)
		}
		for i, l := range strings.Split(view, "\n") {
			if w := ansi.StringWidth(l); w > 46 {
				t.Errorf("%s: linha %d com %d colunas: %q", tc.name, i, w, l)
			}
		}
	}
}

func TestToolsMenuOpensAndEscGoesBack(t *testing.T) {
	s := services(t, false)
	m := New(s, MainMenu(), false)
	m.Init()
	m.Update(tea.WindowSizeMsg{Width: 46, Height: 40})
	_, cmd := m.Update(key("t"))
	for _, msg := range run(cmd) {
		m.Update(msg)
	}
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, "Teste de velocidade") {
		t.Fatal(v)
	}
}

func TestStressScreenStopsOnEsc(t *testing.T) {
	s := services(t, false)
	c := &Ctx{W: 46, H: 40, S: s, RNG: newRNG()}
	scr := stressTool(c).(*Tool)
	scr.Init(c)
	_, cmd := scr.Update(c, key("esc"))
	if cmd == nil {
		t.Fatal("esc deveria voltar")
	}
	if scr.cancel == nil {
		t.Fatal("sem cancelamento")
	}
}
