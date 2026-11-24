package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// Install é o painel de instalação.
type Install struct {
	p      *plan.Plan
	prog   *engine.Progress
	cancel context.CancelFunc
	snap   engine.Snapshot

	lastCur    int
	stepAt     int     // quadro em que o passo atual começou (decifrar título)
	flash      []int   // quadro em que cada passo terminou (✓ acende)
	shakeUntil int     // quadro até onde a tela treme (falha)
	activity   float64 // energia do espectro (linhas de log por segundo, suavizado)
	lastLines  int
	doneAt     int // quadro em que o motor terminou
	cancelled  bool

	// HUD (amostrado 2x por segundo)
	hudAt     int
	cacheB    int64
	rate      []float64 // MB/s
	mem       sys.Mem
	memOK     bool
	free      int64
	cacheDirs []string
}

// NewInstall cria o painel para o plano.
func NewInstall(p *plan.Plan) *Install { return &Install{p: p, lastCur: -1} }

func (in *Install) NoReveal() {}

func (in *Install) Init(c *Ctx) tea.Cmd {
	in.prog = &engine.Progress{}
	in.flash = make([]int, len(in.p.Steps))
	ctx, cancel := context.WithCancel(context.Background())
	in.cancel = cancel
	e := c.S.Engine()
	in.prog.Start(in.p) // o painel já tem o que mostrar antes do motor começar
	go e.Run(ctx, in.p, in.prog, engine.Notifier{Run: c.S.Runner})
	in.cacheDirs = []string{
		filepath.Join(c.S.Env.Prefix, "var/cache/apt/archives"),
		filepath.Join(c.S.Env.Prefix, "var/cache/apt/archives/partial"),
		filepath.Join(c.S.Env.Prefix, "var/lib/proot-distro/installed-rootfs/ubuntu/var/cache/apt/archives"),
		filepath.Join(c.S.Env.Prefix, "var/lib/proot-distro/installed-rootfs/ubuntu/var/cache/apt/archives/partial"),
	}
	in.sampleHUD(c)
	in.snap = in.prog.Snapshot()
	return nil
}

func (in *Install) Animating(*Ctx) bool { return true }

func (in *Install) sampleHUD(c *Ctx) {
	var b int64
	for _, d := range in.cacheDirs {
		b += sys.DirBytes(d)
	}
	if in.hudAt > 0 {
		dt := float64(c.Frame-in.hudAt) / FPS
		mbps := 0.0
		if dt > 0 && b > in.cacheB {
			mbps = float64(b-in.cacheB) / 1e6 / dt
		}
		if c.S.Env.DryRun { // simulação: sinal plausível para o painel ter vida
			mbps = 0.4 + 2.5*in.activity*c.RNG.Float64()
		}
		in.rate = append(in.rate, mbps)
		if len(in.rate) > 16 {
			in.rate = in.rate[1:]
		}
	}
	in.cacheB, in.hudAt = b, c.Frame
	in.mem, in.memOK = sys.ReadMem()
	in.free = c.S.FreeMB()
}

func (in *Install) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if !in.cancelled && in.snap.Result == nil {
				in.cancelled = true
				in.cancel()
			}
		case "enter":
			if in.snap.Result != nil {
				return in, Replace(NewResult(in.snap.Result))
			}
		}
	case tickMsg:
		in.snap = in.prog.Snapshot()
		s := in.snap
		if s.Current != in.lastCur {
			if in.lastCur >= 0 {
				in.flash[in.lastCur] = c.Frame
			}
			in.lastCur, in.stepAt = s.Current, c.Frame
		}
		for i, st := range s.Status {
			if st == engine.Failed && in.flash[i] == 0 {
				in.flash[i] = c.Frame
				in.shakeUntil = c.Frame + 8
			}
		}
		// energia = linhas novas no log (suavizado)
		inst := float64(s.NLines-in.lastLines) / 3
		in.lastLines = s.NLines
		in.activity = in.activity*0.85 + min(1, inst)*0.15 + 0.02
		if c.Frame-in.hudAt >= FPS/2 {
			in.sampleHUD(c)
		}
		if s.Result != nil {
			if in.doneAt == 0 {
				in.doneAt = c.Frame
				for i := range in.flash {
					if in.flash[i] == 0 {
						in.flash[i] = c.Frame
					}
				}
			}
			if c.Frame-in.doneAt > FPS*3/2 || c.Auto && c.Frame-in.doneAt > FPS {
				return in, Replace(NewResult(s.Result))
			}
		}
	}
	return in, nil
}

// percent pondera os passos pelo tempo estimado; o passo atual conta
// pelo tempo decorrido (no máximo 90% dele).
func (in *Install) percent() int {
	s := in.snap
	if s.Result != nil {
		return 100
	}
	total, done := 0.0, 0.0
	for i, st := range in.p.Steps {
		w := float64(max(1, st.Seconds))
		total += w
		switch {
		case in.st(i) != engine.Pending && in.st(i) != engine.Running:
			done += w
		case i == s.Current:
			done += min(w*0.9, time.Since(s.StepStart).Seconds())
		}
	}
	return min(99, int(done/total*100))
}

func (in *Install) View(c *Ctx) string {
	s := in.snap
	n := len(in.p.Steps)
	pct := in.percent()
	elapsed := int(time.Since(s.Start).Seconds())
	if s.Result != nil {
		elapsed = int(s.Result.Elapsed().Seconds())
	}
	title := in.p.Title
	if in.cancelled && s.Result == nil {
		title = "Interrompendo"
	}
	lines := header(c, title)
	lines[1] = fit(lines[1], c.W-7) + " " + paint(mmss(elapsed), Dim)

	// ---- dígitos grandes + espectro
	digits := fx.HalfBlocks(fx.Bitmap(fmt.Sprintf("%d%%", pct)))
	glitch := c.Frame < in.shakeUntil || (c.Frame-in.stepAt < 3 && in.stepAt > 0)
	dw := width(digits[0])
	spw := max(0, c.W-dw-6)
	spec := fx.Spectrum(spw, 3, float64(c.Frame)/FPS*2, in.activity, c.RNG)
	lines = append(lines, "")
	for i, d := range digits {
		off := " "
		if glitch && i == c.Frame%3 {
			off = "  "
		}
		col := grad(d, float64(c.Frame)/90, false)
		if c.Frame < in.shakeUntil {
			col = paint(d, Red)
		}
		lines = append(lines, off+col+"   "+gradRev(spec[i]))
	}
	lines = append(lines, "")

	// ---- barra com brilho
	bw := max(10, c.W-8) // + " 100%"
	lines = append(lines, " "+in.bar(c, pct, bw))

	// ---- passo atual
	cur := min(s.Current, n-1)
	label := in.p.Steps[cur].Label
	if s.Result != nil {
		label = "Finalizando"
	}
	stepTxt := fx.Decrypt(label, float64(c.Frame-in.stepAt)/8, c.RNG)
	spin := paint(spinner[c.Frame%len(spinner)], Grad.At(float64(c.Frame%40)/40))
	if s.Result != nil {
		spin = bold("◉", Green)
	}
	meta := paint(fmt.Sprintf("%d/%d", min(cur+1, n), n), Dim)
	if eta := in.eta(pct, elapsed); eta != "" {
		meta += paint(" · ~"+eta, Faint)
	}
	lines = append(lines, " "+spin+" "+bold(fit(stepTxt, c.W-width(meta)-6), White)+"  "+meta)
	lines = append(lines, "")

	// ---- quanto espaço sobra para log e linha do tempo
	free := c.H - len(lines) - 4 // HUD + vazio + pontos + margem
	logRows := min(6, max(2, free/2))
	tlRows := max(0, free-logRows-2)

	// ---- caixa de atividade (log rolando)
	addr := fmt.Sprintf("0x%04X", (s.NLines*40503)&0xFFFF)
	boxW := c.W - 2
	top := "┌─ " + "ATIVIDADE" + " "
	top = paint(top, Faint) + paint(strings.Repeat("─", max(0, boxW-width(top)-len(addr)-2)), Ghost) + " " + paint(addr, Faint) + paint("┐", Faint)
	lines = append(lines, " "+top)
	logs := s.Lines
	if len(logs) > logRows {
		logs = logs[len(logs)-logRows:]
	}
	for i := 0; i < logRows; i++ {
		txt := ""
		if j := i - (logRows - len(logs)); j >= 0 {
			txt = logs[j]
		}
		age := float64(logRows-1-i) / float64(max(1, logRows-1)) // 0 = mais nova
		col := fx.Lerp(White, Ghost, age*0.9)
		if i == logRows-1 && txt != "" {
			col = Cyan
		}
		lines = append(lines, " "+paint("│", Faint)+" "+pad(paint(fit(txt, boxW-4), col), boxW-3)+paint("│", Faint))
	}
	lines = append(lines, " "+paint("└"+strings.Repeat("─", max(0, boxW-2))+"┘", Faint))

	// ---- HUD
	lines = append(lines, " "+in.hud(c))
	lines = append(lines, "")

	// ---- linha do tempo
	if tlRows > 0 {
		first := max(0, min(cur-tlRows/2, n-tlRows))
		for i := first; i < min(n, first+tlRows); i++ {
			lines = append(lines, in.stepLine(c, i, cur))
		}
	}
	// ---- corrente de pontos (todos os passos)
	var chain strings.Builder
	for i := 0; i < n && i < c.W-2; i++ {
		switch st := in.st(i); {
		case st == engine.OK:
			chain.WriteString(paint("●", Grad.At(float64(i)/float64(max(1, n-1)))))
		case st == engine.Failed:
			chain.WriteString(paint("●", Red))
		case i == cur && s.Result == nil:
			chain.WriteString(bold("◉", White))
		default:
			chain.WriteString(paint("○", Ghost))
		}
	}
	lines = append(lines, " "+chain.String())
	if in.cancelled && s.Result == nil {
		lines = append(lines, " "+paint("interrompendo o passo atual…", Yellow))
	} else if s.Result == nil {
		lines = append(lines, footer(c.W, "q interrompe"))
	}
	return strings.Join(lines, "\n")
}

// st devolve o status do passo i (Pending se o retrato ainda não tem).
func (in *Install) st(i int) engine.Status {
	if i < len(in.snap.Status) {
		return in.snap.Status[i]
	}
	return engine.Pending
}

func (in *Install) eta(pct, elapsed int) string {
	if pct < 8 || pct >= 100 {
		return ""
	}
	return mmss(elapsed * (100 - pct) / pct)
}

func (in *Install) bar(c *Ctx, pct, w int) string {
	filled := pct * w / 100
	shine := -1
	if filled > 2 {
		shine = c.Frame % (filled + 10)
	}
	var b strings.Builder
	for i := 0; i < w; i++ {
		switch {
		case i < filled && (i == shine || i == shine-1):
			b.WriteString(paint("█", White))
		case i < filled:
			b.WriteString(paint("█", Grad.At(float64(i)/float64(w))))
		case i == filled && pct < 100:
			b.WriteString(paint("▓", Accent))
		default:
			b.WriteString(paint("░", Ghost))
		}
	}
	return b.String() + " " + bold(fmt.Sprintf("%3d%%", pct), White)
}

func (in *Install) hud(c *Ctx) string {
	rate := 0.0
	if len(in.rate) > 0 {
		rate = in.rate[len(in.rate)-1]
	}
	hist := in.rate
	if len(hist) > 8 {
		hist = hist[len(hist)-8:]
	}
	parts := []string{paint("↓ ", Cyan) + bold(fmt.Sprintf("%.1fMB/s", rate), White) + " " + gradRev(fx.Spark(append(hist, 0)))}
	if in.memOK {
		u := in.mem.UsedPct()
		col := Green
		if u > 85 {
			col = Red
		} else if u > 70 {
			col = Yellow
		}
		parts = append(parts, paint("RAM ", Dim)+bold(itoa(u)+"%", col))
	}
	parts = append(parts, paint("disco ", Dim)+bold(humanMB(in.free), White))
	return fit(strings.Join(parts, paint(" · ", Ghost)), c.W-2)
}

func (in *Install) stepLine(c *Ctx, i, cur int) string {
	s := in.snap
	st := in.st(i)
	label := fit(in.p.Steps[i].Label, c.W-12)
	dur := ""
	if i < len(s.Durations) && s.Durations[i] > 0 {
		dur = paint(fmt.Sprintf(" %ds", int(s.Durations[i].Seconds())), Faint)
	}
	glow := 0.0 // ✓ acende e apaga
	if f := in.flash[i]; f > 0 && c.Frame-f < 12 {
		glow = 1 - float64(c.Frame-f)/12
	}
	switch {
	case st == engine.OK:
		return " " + bold("✓", fx.Lerp(Green, White, glow)) + " " + paint(label, fx.Lerp(Dim, White, glow)) + dur
	case st == engine.Failed:
		return " " + bold("✗", Red) + " " + paint(label, fx.Lerp(Text, Red, glow)) + dur
	case st == engine.Interrupted:
		return " " + bold("■", Yellow) + " " + paint(label, Yellow)
	case i == cur && s.Result == nil:
		return band(bold("▌", Accent)+bold("▶", Grad.At(float64(c.Frame%30)/30))+" "+bold(label, White)+
			paint(fmt.Sprintf(" %ds", int(time.Since(s.StepStart).Seconds())), Dim), c.W)
	default:
		return " " + paint("· "+label, Faint)
	}
}
