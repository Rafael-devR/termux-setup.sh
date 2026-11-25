// Package tools reúne as ferramentas do menu "Ferramentas": leituras do
// aparelho (bateria, temperatura, memória, CPU, processos, disco), testes
// (velocidade, internet, ping, estresse), diagnóstico (doctor), limpeza e
// backup. Aqui fica só a lógica; as telas ficam em internal/ui.
//
// Tudo que toca o sistema passa pelo Toolbox, que tem uma versão "de
// mentira" (modo --teste) e aceita pastas /proc e /sys trocadas nos testes.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// ErrNoAPI: falta o app Termux:API (ou o pacote termux-api).
var ErrNoAPI = errors.New("precisa do app Termux:API (F-Droid) e do pacote termux-api")

// Toolbox junta as dependências das ferramentas.
type Toolbox struct {
	Run   sys.Runner
	Env   *sys.Env
	Info  sys.Info
	Fake  bool   // modo --teste: dados inventados, nada pesado roda
	Proc  string // padrão "/proc"
	Sysfs string // padrão "/sys"
}

func (t *Toolbox) proc() string {
	if t.Proc != "" {
		return t.Proc
	}
	return "/proc"
}

func (t *Toolbox) sysfs() string {
	if t.Sysfs != "" {
		return t.Sysfs
	}
	return "/sys"
}

// termuxAPI roda um comando do Termux:API (com prazo: sem o app ele trava).
func (t *Toolbox) termuxAPI(ctx context.Context, name string, v any) error {
	if !t.Run.Has(name) {
		return ErrNoAPI
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := sys.Output(cctx, t.Run, sys.C(name))
	if err != nil {
		if cctx.Err() != nil {
			return ErrNoAPI
		}
		return err
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return ErrNoAPI
	}
	return nil
}

// ---------------------------------------------------------------- bateria

// Battery é a leitura da bateria.
type Battery struct {
	Percent int
	Temp    float64 // °C
	Status  string  // CHARGING, DISCHARGING, FULL...
	Plugged string  // UNPLUGGED, PLUGGED_AC, PLUGGED_USB...
	Health  string
}

// Charging diz se está carregando.
func (b Battery) Charging() bool { return b.Status == "CHARGING" || b.Status == "FULL" }

// ReadBattery lê a bateria pelo Termux:API.
func (t *Toolbox) ReadBattery(ctx context.Context) (Battery, error) {
	if t.Fake {
		return Battery{Percent: 64 + rand.IntN(3), Temp: 31 + rand.Float64()*3, Status: "DISCHARGING", Plugged: "UNPLUGGED", Health: "GOOD"}, nil
	}
	var r struct {
		Health      string  `json:"health"`
		Percentage  int     `json:"percentage"`
		Plugged     string  `json:"plugged"`
		Status      string  `json:"status"`
		Temperature float64 `json:"temperature"`
	}
	if err := t.termuxAPI(ctx, "termux-battery-status", &r); err != nil {
		return Battery{}, err
	}
	return Battery{Percent: r.Percentage, Temp: r.Temperature, Status: r.Status, Plugged: r.Plugged, Health: r.Health}, nil
}

// ---------------------------------------------------------------- Wi-Fi

// WiFi é a conexão Wi-Fi atual.
type WiFi struct {
	SSID      string
	RSSI      int // dBm
	LinkMbps  int
	FreqMHz   int
	IP        string
	Connected bool
}

// ReadWiFi lê a conexão pelo Termux:API.
func (t *Toolbox) ReadWiFi(ctx context.Context) (WiFi, error) {
	if t.Fake {
		return WiFi{SSID: "casa-5G", RSSI: -52 - rand.IntN(6), LinkMbps: 433, FreqMHz: 5180, IP: "192.168.0.23", Connected: true}, nil
	}
	var r struct {
		SSID  string `json:"ssid"`
		RSSI  int    `json:"rssi"`
		Link  int    `json:"link_speed_mbps"`
		Freq  int    `json:"frequency_mhz"`
		IP    string `json:"ip"`
		State string `json:"supplicant_state"`
	}
	if err := t.termuxAPI(ctx, "termux-wifi-connectioninfo", &r); err != nil {
		return WiFi{}, err
	}
	w := WiFi{SSID: strings.Trim(r.SSID, `"`), RSSI: r.RSSI, LinkMbps: r.Link, FreqMHz: r.Freq, IP: r.IP, Connected: r.State == "COMPLETED"}
	if w.SSID == "<unknown ssid>" {
		w.SSID = "(oculto pelo Android: libere a localização para o Termux:API)"
	}
	return w, nil
}

// Signal descreve o sinal Wi-Fi em palavras.
func (w WiFi) Signal() string {
	switch {
	case w.RSSI >= -55:
		return "ótimo"
	case w.RSSI >= -67:
		return "bom"
	case w.RSSI >= -75:
		return "fraco"
	}
	return "muito fraco"
}
