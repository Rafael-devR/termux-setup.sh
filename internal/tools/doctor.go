package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/ubuntu"
)

// Doctor confere o ambiente e devolve uma lista de verificações. Cada
// resultado chega por emit assim que fica pronto.
func (t *Toolbox) Doctor(ctx context.Context, emit func(Check)) {
	// internet (resumida)
	netOK := true
	t.NetCheck(ctx, func(c Check) {
		if c.Status != OK {
			netOK = false
			emit(c)
		}
	})
	if netOK {
		emit(Check{Name: "Internet, DNS e repositórios", Detail: "tudo respondendo"})
	}
	if ctx.Err() != nil {
		return
	}

	d := t.ReadDisk()
	switch {
	case d.FreeMB < 500:
		emit(Check{Name: "Espaço livre", Status: Fail, Detail: fmt.Sprintf("%d MB", d.FreeMB), Fix: "Abra Ferramentas > Limpeza ou apague arquivos. Instalar precisa de pelo menos 1 GB."})
	case d.FreeMB < 2048:
		emit(Check{Name: "Espaço livre", Status: Warn, Detail: fmt.Sprintf("%d MB", d.FreeMB), Fix: "Pouco espaço para instalar tudo. Veja Ferramentas > Limpeza."})
	default:
		emit(Check{Name: "Espaço livre", Detail: fmt.Sprintf("%.1f GB", float64(d.FreeMB)/1024)})
	}

	switch src := os.Getenv("TERMUX_APK_RELEASE"); src {
	case "GOOGLE_PLAY":
		emit(Check{Name: "Origem do Termux", Status: Fail, Detail: "Play Store", Fix: "A versão da Play Store é antiga e dá erro de chave (NO_PUBKEY). Instale o Termux pelo F-Droid ou GitHub."})
	case "":
		emit(Check{Name: "Origem do Termux", Status: Warn, Detail: "não consegui identificar", Fix: "Se der erro de chave GPG no apt, reinstale o Termux pelo F-Droid."})
	default:
		emit(Check{Name: "Origem do Termux", Detail: strings.ToLower(strings.ReplaceAll(src, "_", "-"))})
	}

	bin := t.Env.BinDir()
	inPath := false
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == bin {
			inPath = true
		}
	}
	if inPath {
		emit(Check{Name: "~/bin no PATH", Detail: "ok"})
	} else {
		emit(Check{Name: "~/bin no PATH", Status: Warn, Detail: "fora do PATH", Fix: "Rode: source ~/.bashrc (ou reabra o Termux)."})
	}

	if _, err := os.Stat(filepath.Join(t.Env.Home, "storage")); err == nil {
		emit(Check{Name: "Acesso ao armazenamento", Detail: "liberado"})
	} else {
		emit(Check{Name: "Acesso ao armazenamento", Status: Warn, Detail: "não liberado", Fix: "Rode: termux-setup-storage"})
	}

	if _, err := t.ReadBattery(ctx); err != nil {
		emit(Check{Name: "Termux:API", Status: Warn, Detail: "não respondeu", Fix: "Instale o app Termux:API (mesma origem do Termux) e o pacote termux-api. Libera bateria, Wi-Fi e notificações."})
	} else {
		b, _ := t.ReadBattery(ctx)
		c := Check{Name: "Bateria", Detail: fmt.Sprintf("%d%% · %.0f °C", b.Percent, b.Temp)}
		if b.Percent < 20 && !b.Charging() {
			c.Status, c.Fix = Warn, "Bateria baixa: ligue o carregador antes de uma instalação longa."
		}
		if b.Temp >= 42 {
			c.Status, c.Fix = Warn, "Aparelho quente: deixe esfriar antes de instalar coisas pesadas."
		}
		emit(c)
	}

	if t.Fake || ubuntu.Has(ctx, t.Run) {
		emit(Check{Name: "Ubuntu (proot)", Detail: "instalado"})
	} else {
		emit(Check{Name: "Ubuntu (proot)", Status: Warn, Detail: "não instalado", Fix: "Menu: Ubuntu & IA > Instalar tudo."})
	}

	if t.Info.Is64() {
		emit(Check{Name: "Arquitetura", Detail: t.Info.Arch})
	} else {
		emit(Check{Name: "Arquitetura", Status: Warn, Detail: t.Info.Arch + " (32 bits)", Fix: "Claude Code e Codex só rodam em 64 bits."})
	}
}
