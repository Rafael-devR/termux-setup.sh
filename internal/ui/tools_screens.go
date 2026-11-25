package ui

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/tools"
)

// ToolsMenu é o menu Ferramentas.
func ToolsMenu() *Menu {
	push := func(f func(c *Ctx) Screen) func(*Ctx) tea.Cmd {
		return func(c *Ctx) tea.Cmd { return Push(f(c)) }
	}
	return &Menu{
		Title: "Ferramentas",
		Pre: func(c *Ctx) []string {
			return []string{paint("Monitor do aparelho, testes e manutenção", Dim)}
		},
		Items: []MenuItem{
			{Key: "1", Group: "Monitor (ao vivo)", Label: "Bateria, temperatura, memória, CPU", Hint: "Painel que se atualiza a cada segundo, com gráficos.", Run: push(monitorTool)},
			{Key: "2", Label: "Processos (os que mais gastam)", Hint: "Lista por memória. Selecione e tecle x para encerrar um.", Run: push(procsTool)},
			{Key: "3", Label: "Wi-Fi (sinal e velocidade do link)", Hint: "Precisa do Termux:API.", Run: push(wifiTool)},
			{Key: "4", Group: "Testes", Label: "Teste de velocidade", Hint: "Ping, download e upload.", Run: push(speedTool)},
			{Key: "5", Label: "Teste de internet (onde quebra?)", Hint: "Confere IP, DNS, GitHub e os repositórios do Termux e Ubuntu.", Run: push(netTool)},
			{Key: "6", Label: "Ping contínuo", Hint: "Latência ao vivo, boa para comparar Wi-Fi e 4G.", Run: push(pingTool)},
			{Key: "7", Label: "Teste de estresse (30 s)", Hint: "Carrega a CPU e mostra a temperatura subindo. Cancele com esc.", Run: push(stressTool)},
			{Key: "8", Group: "Manutenção", Label: "Doctor (diagnóstico geral)", Hint: "Procura problemas e diz como corrigir.", Run: push(doctorTool)},
			{Key: "9", Label: "Limpeza (liberar espaço)", Hint: "Cache do apt, caches de programas e logs antigos. Backups do Desfazer ficam.", Run: push(cleanTool)},
			{Key: "i", Label: "Informações do aparelho", Hint: "Resumo para copiar e colar num pedido de ajuda.", Run: push(infoTool)},
			{Key: "b", Label: "Backup e restauração da configuração", Hint: ".bashrc, ~/.termux, ~/bin e .gitconfig.", Run: push(backupTool)},
			{Key: "0", Label: "Voltar", Run: func(*Ctx) tea.Cmd { return Pop }},
		},
	}
}

func unavailable(why string) string { return paint("indisponível: "+why, Faint) }

// ---------------------------------------------------------------- monitor

type snapshot struct {
	bat    tools.Battery
	batErr error
	temp   float64 // °C (bateria ou zona térmica)
	mem    tools.Memory
	memOK  bool
	cpu    []int
	cpuOK  bool
	disk   tools.Disk
}

func monitorTool(c *Ctx) Screen {
	tb := c.S.Tools
	var snap snapshot
	var temps, cpus, rams []float64
	got := false
	return &Tool{
		Title: "Monitor do aparelho",
		Run: func(ctx context.Context, send func(any)) {
			prev, _ := tb.ReadCPU()
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for n := 0; ; n++ {
				var s snapshot
				s.bat, s.batErr = tb.ReadBattery(ctx)
				s.temp = s.bat.Temp
				if s.batErr != nil {
					if th, ok := tb.ReadThermal(); ok {
						s.temp = th
					}
				}
				s.mem, s.memOK = tb.ReadMemory()
				if cur, ok := tb.ReadCPU(); ok {
					s.cpu, s.cpuOK = tools.CPUPercent(prev, cur), true
					prev = cur
				}
				s.disk = tb.ReadDisk()
				send(s)
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
			}
		},
		OnData: func(v any) {
			snap, got = v.(snapshot), true
			temps = push(temps, snap.temp, 40)
			rams = push(rams, float64(snap.mem.UsedPct()), 40)
			if snap.cpuOK && len(snap.cpu) > 0 {
				cpus = push(cpus, float64(snap.cpu[0]), 40)
			}
		},
		Body: func(c *Ctx, _ bool) []string {
			if !got {
				return []string{paint(spinner[c.Frame%len(spinner)]+" lendo sensores…", Dim)}
			}
			var out []string
			w := min(c.W-14, 30)
			// bateria + temperatura
			if snap.batErr != nil {
				out = append(out, panel(c, Faint, "Bateria", unavailable(snap.batErr.Error()))...)
			} else {
				st := map[string]string{"CHARGING": "carregando", "DISCHARGING": "descarregando", "FULL": "cheia", "NOT_CHARGING": "conectada"}[snap.bat.Status]
				if st == "" {
					st = strings.ToLower(snap.bat.Status)
				}
				out = append(out, panel(c, usageColor(100-snap.bat.Percent), "Bateria",
					meter(snap.bat.Percent, w, usageColor(100-snap.bat.Percent))+" "+bold(fmt.Sprintf("%d%%", snap.bat.Percent), White),
					paint(st, Dim))...)
			}
			if snap.temp > 0 {
				lo, hi := rng(temps)
				out = append(out, panel(c, tempColor(snap.temp), "Temperatura",
					bold(fmt.Sprintf("%.1f °C", snap.temp), tempColor(snap.temp))+paint(fmt.Sprintf("  mín %.0f · máx %.0f", lo, hi), Dim),
					spark(temps, w+8, lo-1, hi+1, tempColor(snap.temp)))...)
			} else {
				out = append(out, panel(c, Faint, "Temperatura", unavailable("o Android não libera (instale o Termux:API)"))...)
			}
			// memória
			if snap.memOK {
				p := snap.mem.UsedPct()
				rows := []string{meter(p, w, usageColor(p)) + " " + bold(fmt.Sprintf("%d%%", p), White),
					paint(fmt.Sprintf("%s de %s", humanMB(snap.mem.TotalMB-snap.mem.AvailMB), humanMB(snap.mem.TotalMB)), Dim)}
				if snap.mem.SwapTotalMB > 0 {
					rows = append(rows, paint(fmt.Sprintf("swap %s de %s", humanMB(snap.mem.SwapTotalMB-snap.mem.SwapFreeMB), humanMB(snap.mem.SwapTotalMB)), Faint))
				}
				out = append(out, panel(c, usageColor(p), "Memória (RAM)", rows...)...)
			} else {
				out = append(out, panel(c, Faint, "Memória (RAM)", unavailable("sem acesso a /proc/meminfo"))...)
			}
			// CPU
			if snap.cpuOK && len(snap.cpu) > 0 {
				rows := []string{meter(snap.cpu[0], w, usageColor(snap.cpu[0])) + " " + bold(fmt.Sprintf("%d%%", snap.cpu[0]), White)}
				if len(snap.cpu) > 1 {
					var cores []string
					for _, p := range snap.cpu[1:] {
						cores = append(cores, paint(fmt.Sprintf("%3d", p), usageColor(p)))
					}
					rows = append(rows, paint("núcleos ", Dim)+strings.Join(cores, " "))
				}
				out = append(out, panel(c, usageColor(snap.cpu[0]), "CPU", rows...)...)
			} else {
				out = append(out, panel(c, Faint, "CPU", unavailable("o Android bloqueia /proc/stat nesta versão"))...)
			}
			// disco
			d := snap.disk
			if d.TotalMB > 0 {
				out = append(out, panel(c, usageColor(d.UsedPct()), "Armazenamento",
					meter(d.UsedPct(), w, usageColor(d.UsedPct()))+" "+bold(fmt.Sprintf("%d%%", d.UsedPct()), White),
					paint(humanMB(d.FreeMB)+" livres de "+humanMB(d.TotalMB), Dim))...)
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- processos

func procsTool(c *Ctx) Screen {
	tb := c.S.Tools
	var ps []tools.Proc
	cur := 0
	note := ""
	return &Tool{
		Title: "Processos",
		Hints: []string{"↑↓ escolher", "x encerrar"},
		Run: func(ctx context.Context, send func(any)) {
			tick := time.NewTicker(2 * time.Second)
			defer tick.Stop()
			for {
				send(tb.TopProcs(14))
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
			}
		},
		OnData: func(v any) {
			ps = v.([]tools.Proc)
			cur = min(cur, max(0, len(ps)-1))
		},
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			switch k {
			case "up", "k":
				cur = max(0, cur-1)
			case "down", "j":
				cur = min(len(ps)-1, cur+1)
			case "x":
				if len(ps) > 0 && !c.S.Env.DryRun {
					if err := tools.Terminate(ps[cur].PID); err != nil {
						note = "não consegui encerrar " + ps[cur].Name
					} else {
						note = "pedi para " + ps[cur].Name + " encerrar"
					}
				} else {
					note = "modo teste: nada é encerrado"
				}
			default:
				return nil, false
			}
			return nil, true
		},
		Body: func(c *Ctx, _ bool) []string {
			if len(ps) == 0 {
				return []string{paint("nenhum processo visível (o Android só mostra os do Termux)", Faint)}
			}
			var out []string
			for i, p := range ps {
				row := fmt.Sprintf("%-18s %6s", fit(p.Name, 18), humanMB(p.RSSMB))
				if i == cur {
					out = append(out, band(row, c.W-2))
				} else {
					out = append(out, "  "+paint(row, Text))
				}
			}
			if note != "" {
				out = append(out, "", paint(note, Yellow))
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- Wi-Fi

func wifiTool(c *Ctx) Screen {
	tb := c.S.Tools
	var w tools.WiFi
	var err error
	got := false
	type res struct {
		w   tools.WiFi
		err error
	}
	var sig []float64
	return &Tool{
		Title: "Wi-Fi",
		Run: func(ctx context.Context, send func(any)) {
			tick := time.NewTicker(2 * time.Second)
			defer tick.Stop()
			for {
				x, e := tb.ReadWiFi(ctx)
				send(res{x, e})
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
			}
		},
		OnData: func(v any) {
			r := v.(res)
			w, err, got = r.w, r.err, true
			if err == nil {
				sig = push(sig, float64(100+w.RSSI), 40)
			}
		},
		Body: func(c *Ctx, _ bool) []string {
			switch {
			case !got:
				return []string{paint(spinner[c.Frame%len(spinner)]+" lendo…", Dim)}
			case err != nil:
				return append(panel(c, Faint, "Wi-Fi", unavailable(err.Error())), "",
					paint("Dica: instale o app Termux:API (mesma origem do Termux) e rode: pkg install termux-api", Dim))
			case !w.Connected:
				return panel(c, Yellow, "Wi-Fi", paint("sem conexão Wi-Fi no momento", Yellow))
			}
			lo, hi := rng(sig)
			return panel(c, Cyan, "Wi-Fi · "+w.SSID,
				bold(fmt.Sprintf("%d dBm", w.RSSI), White)+paint("  sinal "+w.Signal(), Dim),
				spark(sig, min(c.W-8, 40), lo-1, hi+1, Cyan),
				paint(fmt.Sprintf("link %d Mbps · %.1f GHz · IP %s", w.LinkMbps, float64(w.FreqMHz)/1000, w.IP), Dim))
		},
	}
}

// ---------------------------------------------------------------- velocidade

func speedTool(c *Ctx) Screen {
	tb := c.S.Tools
	type stage struct {
		name string
		r    tools.SpeedResult
	}
	type final struct {
		r   tools.SpeedResult
		err error
	}
	var st stage
	var fin *final
	return &Tool{
		Title: "Teste de velocidade",
		Hints: []string{"r repetir"},
		Run: func(ctx context.Context, send func(any)) {
			r, err := tb.SpeedTest(ctx, func(s string, p tools.SpeedResult) { send(stage{s, p}) })
			if ctx.Err() == nil {
				send(final{r, err})
			}
		},
		OnData: func(v any) {
			switch x := v.(type) {
			case stage:
				st = x
			case final:
				fin = &x
			}
		},
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			if k == "r" && t.done {
				st, fin = stage{}, nil
				return t.Restart(), true
			}
			return nil, false
		},
		Body: func(c *Ctx, done bool) []string {
			if fin != nil && fin.err != nil {
				return append(panel(c, Red, "Falhou", paint(fin.err.Error(), Text)), "", paint("Veja também: Ferramentas > Teste de internet", Dim))
			}
			r := st.r
			if fin != nil {
				r = fin.r
			}
			label := map[string]string{"ping": "medindo o ping…", "download": "medindo o download…", "upload": "medindo o upload…"}[st.name]
			sp := ""
			if fin == nil {
				sp = paint(spinner[c.Frame%len(spinner)]+" "+label, Dim)
			}
			num := func(v float64, unit string, on bool) string {
				if !on {
					return paint("—", Faint)
				}
				return bold(fmt.Sprintf("%.1f", v), White) + paint(" "+unit, Dim)
			}
			stg := map[string]int{"ping": 0, "download": 1, "upload": 2, "fim": 3, "": -1}[st.name]
			out := panel(c, Cyan, "Resultado",
				paint("Ping      ", Dim)+num(r.PingMs, "ms", stg >= 1)+paint(fmt.Sprintf("  (jitter %.1f ms)", r.JitterMs), Faint),
				paint("Download  ", Dim)+num(r.DownMbps, "Mbps", stg >= 2),
				paint("Upload    ", Dim)+num(r.UpMbps, "Mbps", stg >= 3 || fin != nil))
			if sp != "" {
				out = append(out, "", sp)
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- internet

func checkLines(c *Ctx, checks []tools.Check) []string {
	var out []string
	for _, k := range checks {
		mark, col := bold("✓", Green), Text
		switch k.Status {
		case tools.Warn:
			mark, col = bold("!", Yellow), Text
		case tools.Fail:
			mark, col = bold("✗", Red), Text
		}
		out = append(out, mark+" "+paint(k.Name, col)+paint("  "+k.Detail, Dim))
		if k.Fix != "" {
			for _, l := range wrap(k.Fix, c.W-8) {
				out = append(out, "  "+paint(l, Yellow))
			}
		}
	}
	return out
}

// wrap quebra um texto em linhas de até w colunas.
func wrap(s string, w int) []string {
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		if cur != "" && width(cur)+1+width(word) > w {
			lines = append(lines, cur)
			cur = word
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += word
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func netTool(c *Ctx) Screen {
	tb := c.S.Tools
	var checks []tools.Check
	return &Tool{
		Title: "Teste de internet",
		Hints: []string{"r repetir"},
		Run: func(ctx context.Context, send func(any)) {
			tb.NetCheck(ctx, func(k tools.Check) { send(k) })
		},
		OnData: func(v any) { checks = append(checks, v.(tools.Check)) },
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			if k == "r" && t.done {
				checks = nil
				return t.Restart(), true
			}
			return nil, false
		},
		Body: func(c *Ctx, done bool) []string {
			out := checkLines(c, checks)
			if !done {
				out = append(out, paint(spinner[c.Frame%len(spinner)]+" testando…", Dim))
			} else {
				bad := 0
				for _, k := range checks {
					if k.Status == tools.Fail {
						bad++
					}
				}
				if bad == 0 {
					out = append(out, "", bold("Tudo certo com a internet.", Green))
				}
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- ping

func pingTool(c *Ctx) Screen {
	tb := c.S.Tools
	var samples []float64
	return &Tool{
		Title: "Ping contínuo",
		Run: func(ctx context.Context, send func(any)) {
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				ms, err := tb.PingOnce(ctx)
				if err != nil {
					ms = 0
				}
				send(ms)
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
			}
		},
		OnData: func(v any) { samples = push(samples, v.(float64), 300) },
		Body: func(c *Ctx, _ bool) []string {
			if len(samples) == 0 {
				return []string{paint(spinner[c.Frame%len(spinner)]+" medindo…", Dim)}
			}
			st := tools.Summarize(samples)
			last := samples[len(samples)-1]
			lo, hi := rng(samples[max(0, len(samples)-48):])
			head := bold(fmt.Sprintf("%.0f ms", last), White)
			if last <= 0 {
				head = bold("sem resposta", Red)
			}
			return panel(c, Cyan, "Cloudflare 1.1.1.1",
				head,
				spark(samples, min(c.W-8, 48), lo-1, hi+1, Cyan),
				paint(fmt.Sprintf("mín %.0f · média %.0f · máx %.0f ms · jitter %.0f", st.Min, st.Avg, st.Max, st.Jit), Dim),
				paint(fmt.Sprintf("%d enviados · %d perdidos", st.Sent, st.Lost), Dim))
		},
	}
}

// ---------------------------------------------------------------- estresse

func stressTool(c *Ctx) Screen {
	tb := c.S.Tools
	const total = 30
	var temps []float64
	var first, peak float64
	sec := 0
	return &Tool{
		Title: "Teste de estresse",
		Hints: []string{"r repetir"},
		Run: func(ctx context.Context, send func(any)) {
			tb.Stress(ctx, total*time.Second, func(s tools.StressSample) { send(s) })
		},
		OnData: func(v any) {
			s := v.(tools.StressSample)
			sec = s.Second
			if s.TempC > 0 {
				if first == 0 {
					first = s.TempC
				}
				peak = max(peak, s.TempC)
				temps = push(temps, s.TempC, 40)
			}
		},
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			if k == "r" && t.done {
				temps, first, peak, sec = nil, 0, 0, 0
				return t.Restart(), true
			}
			return nil, false
		},
		Body: func(c *Ctx, done bool) []string {
			w := min(c.W-14, 30)
			p := min(100, sec*100/total)
			if done {
				p = 100
			}
			rows := []string{meter(p, w, Cyan) + " " + paint(fmt.Sprintf("%d/%d s", min(sec, total), total), Dim)}
			if len(temps) > 0 {
				lo, hi := rng(temps)
				rows = append(rows, bold(fmt.Sprintf("%.1f °C", temps[len(temps)-1]), tempColor(temps[len(temps)-1])),
					spark(temps, min(c.W-8, 40), lo-1, hi+1, tempColor(peak)))
			} else {
				rows = append(rows, unavailable("sem leitura de temperatura (instale o Termux:API)"))
			}
			out := panel(c, Cyan, fmt.Sprintf("Carregando %d núcleos", runtime.NumCPU()), rows...)
			if done && peak > 0 {
				verdict := bold("Temperatura normal para uso pesado.", Green)
				switch {
				case peak >= 45:
					verdict = bold("Esquentou muito: evite instalar coisas pesadas com carregador na tomada e capa.", Red)
				case peak >= 40:
					verdict = bold("Esquentou um pouco: dá para usar, mas deixe esfriar entre tarefas longas.", Yellow)
				}
				out = append(out, "", paint(fmt.Sprintf("subiu %.1f °C (de %.1f para %.1f)", peak-first, first, peak), Dim), verdict)
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- doctor

func doctorTool(c *Ctx) Screen {
	tb := c.S.Tools
	var checks []tools.Check
	return &Tool{
		Title: "Doctor",
		Hints: []string{"r repetir"},
		Run: func(ctx context.Context, send func(any)) {
			tb.Doctor(ctx, func(k tools.Check) { send(k) })
		},
		OnData: func(v any) { checks = append(checks, v.(tools.Check)) },
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			if k == "r" && t.done {
				checks = nil
				return t.Restart(), true
			}
			return nil, false
		},
		Body: func(c *Ctx, done bool) []string {
			out := checkLines(c, checks)
			if !done {
				return append(out, paint(spinner[c.Frame%len(spinner)]+" verificando…", Dim))
			}
			fails, warns := 0, 0
			for _, k := range checks {
				switch k.Status {
				case tools.Fail:
					fails++
				case tools.Warn:
					warns++
				}
			}
			switch {
			case fails > 0:
				out = append(out, "", bold(fmt.Sprintf("%d problema(s) e %d aviso(s). Siga as dicas em amarelo.", fails, warns), Red))
			case warns > 0:
				out = append(out, "", bold(fmt.Sprintf("Sem problemas graves, %d aviso(s).", warns), Yellow))
			default:
				out = append(out, "", bold("Tudo certo!", Green))
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- limpeza

func cleanTool(c *Ctx) Screen {
	tb := c.S.Tools
	var items []tools.CleanItem
	sel := map[string]bool{}
	cur := 0
	scanned := false
	result := ""
	type cleaned struct{ freed int64 }
	return &Tool{
		Title: "Limpeza",
		Hints: []string{"espaço marca", "⏎ limpar"},
		Run: func(ctx context.Context, send func(any)) {
			send(tb.CleanScan(ctx))
		},
		OnData: func(v any) {
			switch x := v.(type) {
			case []tools.CleanItem:
				items, scanned = x, true
				for _, it := range items {
					sel[it.ID] = it.Bytes > 0
				}
			case cleaned:
				result = "Liberado: " + humanBytes(x.freed)
			}
		},
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			if !scanned {
				return nil, false
			}
			switch k {
			case "up", "k":
				cur = max(0, cur-1)
			case "down", "j":
				cur = min(len(items)-1, cur+1)
			case "space", " ":
				sel[items[cur].ID] = !sel[items[cur].ID]
			case "enter":
				var ids []string
				for _, it := range items {
					if sel[it.ID] {
						ids = append(ids, it.ID)
					}
				}
				if len(ids) == 0 {
					return nil, true
				}
				if c.S.Env.DryRun {
					result = "Modo teste: nada foi apagado."
					return nil, true
				}
				freed, err := tb.Clean(context.Background(), ids)
				if err != nil {
					result = "Erro: " + err.Error()
				} else {
					result = "Liberado: " + humanBytes(freed)
				}
				items = tb.CleanScan(context.Background())
				for _, it := range items {
					sel[it.ID] = false
				}
			default:
				return nil, false
			}
			return nil, true
		},
		Body: func(c *Ctx, _ bool) []string {
			if !scanned {
				return []string{paint(spinner[c.Frame%len(spinner)]+" medindo o que dá para limpar…", Dim)}
			}
			var out []string
			var total int64
			for i, it := range items {
				mark := paint("○", Faint)
				if sel[it.ID] {
					mark = bold("◉", Green)
					total += it.Bytes
				}
				row := mark + " " + fit(it.Label, c.W-16) + "  " + humanBytes(it.Bytes)
				if i == cur {
					out = append(out, band(row, c.W-2))
				} else {
					out = append(out, "  "+paint(row, Text))
				}
			}
			out = append(out, "", paint("Marcado: "+humanBytes(total), Dim),
				paint("Não mexe nos backups do Desfazer nem no Ubuntu.", Faint))
			if result != "" {
				out = append(out, "", bold(result, Green))
			}
			return out
		},
	}
}

func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/(1<<10))
	}
	return fmt.Sprintf("%d B", b)
}

// ---------------------------------------------------------------- informações

func deviceReport(c *Ctx) []string {
	i := c.S.Info
	host, _ := os.Hostname()
	disk := c.S.Tools.ReadDisk()
	mem, _ := c.S.Tools.ReadMemory()
	lines := []string{
		"termux-setup " + app.Version,
		"Aparelho  : " + i.Model,
		"Android   : " + i.Android,
		"Arquitetura: " + i.Arch + " · " + fmt.Sprint(runtime.NumCPU()) + " núcleos",
		"Memória   : " + humanMB(mem.TotalMB),
		"Disco     : " + humanMB(disk.FreeMB) + " livres de " + humanMB(disk.TotalMB),
		"Espelho   : " + c.S.Mirror(),
		"Host      : " + host,
		"Go        : " + runtime.Version(),
	}
	if a := os.Getenv("TERMUX_APK_RELEASE"); a != "" {
		lines = append(lines, "Origem    : "+a)
	}
	if v := os.Getenv("TERMUX_VERSION"); v != "" {
		lines = append(lines, "Termux    : "+v)
	}
	return lines
}

func infoTool(c *Ctx) Screen {
	var lines []string
	note := ""
	return &Tool{
		Title:  "Informações do aparelho",
		Hints:  []string{"c copiar"},
		Run:    func(ctx context.Context, send func(any)) { send(deviceReport(c)) },
		OnData: func(v any) { lines = v.([]string) },
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			if k != "c" || len(lines) == 0 {
				return nil, false
			}
			if c.S.Env.DryRun || !c.S.Runner.Has("termux-clipboard-set") {
				note = "copiar precisa do Termux:API (termux-clipboard-set)"
				return nil, true
			}
			cmd := sys.Cmd{Name: "termux-clipboard-set", Stdin: strings.Join(lines, "\n")}
			if err := c.S.Runner.Run(context.Background(), cmd, nil); err != nil {
				note = "não consegui copiar"
			} else {
				note = "copiado!"
			}
			return nil, true
		},
		Body: func(c *Ctx, _ bool) []string {
			var out []string
			for _, l := range lines {
				k, v, ok := strings.Cut(l, ":")
				if ok && !strings.HasPrefix(l, "termux-setup") {
					out = append(out, paint(fmt.Sprintf("%-12s", k), Dim)+paint(strings.TrimSpace(v), Text))
				} else {
					out = append(out, bold(l, White))
				}
			}
			if note != "" {
				out = append(out, "", paint(note, Yellow))
			}
			return out
		},
	}
}

// ---------------------------------------------------------------- backup

func backupTool(c *Ctx) Screen {
	tb := c.S.Tools
	note := ""
	var list []string
	return &Tool{
		Title: "Backup da configuração",
		Hints: []string{"b criar backup", "r restaurar o último"},
		Run:   func(ctx context.Context, send func(any)) { send(tb.Backups()) },
		OnData: func(v any) {
			if l, ok := v.([]string); ok {
				list = l
			}
		},
		OnKey: func(c *Ctx, k string, t *Tool) (tea.Cmd, bool) {
			switch k {
			case "b":
				if c.S.Env.DryRun {
					note = "Modo teste: nada foi gravado."
					return nil, true
				}
				f, err := tb.Backup()
				if err != nil {
					note = "Erro: " + err.Error()
				} else {
					note = "Backup salvo: " + c.S.Env.Tilde(f)
				}
				list = tb.Backups()
			case "r":
				if len(list) == 0 {
					note = "Ainda não há backup."
					return nil, true
				}
				if c.S.Env.DryRun {
					note = "Modo teste: nada foi restaurado."
					return nil, true
				}
				n, err := tb.Restore(list[0])
				if err != nil {
					note = "Erro: " + err.Error()
				} else {
					note = fmt.Sprintf("Restaurados %d arquivos. Reabra o Termux para valer.", n)
				}
			default:
				return nil, false
			}
			return nil, true
		},
		Body: func(c *Ctx, _ bool) []string {
			out := []string{paint("Guarda .bashrc, ~/.termux, ~/bin e .gitconfig.", Dim), ""}
			if len(list) == 0 {
				out = append(out, paint("nenhum backup ainda", Faint))
			}
			for i, f := range list {
				if i >= 5 {
					break
				}
				out = append(out, paint(fit(c.S.Env.Tilde(f), c.W-4), Text))
			}
			if note != "" {
				out = append(out, "", bold(note, Cyan))
			}
			return out
		},
	}
}
