package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// Theme é um conjunto completo de cores. As cores têm PAPÉIS fixos (o design
// system): quem desenha usa o papel (Panel, Dim, Primary...) e nunca uma cor
// solta. Trocar de tema troca todos os papéis de uma vez.
//
//	Superfícies  BG < Panel < Raised            (fundo, painel, cartão)
//	Texto        White > Text > Dim > Faint > Ghost
//	Seleção      Select (preenchimento claro) + Ink (texto escuro sobre ele)
//	Acentos      Primary (destaque da marca), Secondary (barras e cartões),
//	             Section (títulos de grupo)
//	Estado       Green (ok), Yellow (aviso), Red (erro)
//	Marca        Grad (degradê do logo e da barra), Matrix (chuva da abertura)
type Theme struct {
	ID, Name, Desc string

	BG, Panel, Raised       string
	White, Text, Dim, Faint string
	Ghost                   string
	Select, Ink             string
	Primary, Secondary      string
	Section                 string
	Green, Yellow, Red      string
	Matrix                  string
	Grad                    [5]string
}

// Themes são os temas disponíveis (o primeiro é o padrão).
var Themes = []Theme{
	{
		ID: "meia-noite", Name: "Meia-noite", Desc: "azul-tinta profundo, acento índigo e azul-céu",
		BG: "#0B0E1A", Panel: "#11152A", Raised: "#1A2040",
		White: "#EEF0FA", Text: "#C3C8DE", Dim: "#8790B0", Faint: "#5B6385", Ghost: "#2C3254",
		Select: "#9DB0FF", Ink: "#0B0E1A",
		Primary: "#8B7BFF", Secondary: "#5CC8F5", Section: "#A9B8FF",
		Green: "#5FD39A", Yellow: "#F5C26B", Red: "#FF6B81",
		Matrix: "#3F8FB5",
		Grad:   [5]string{"#5CC8F5", "#7C9CFF", "#8B7BFF", "#B47CF0", "#E27BD0"},
	},
	{
		ID: "floresta", Name: "Floresta", Desc: "verde-musgo escuro, acento esmeralda e areia",
		BG: "#0A1210", Panel: "#0F1B17", Raised: "#17291F",
		White: "#ECF4EF", Text: "#BFD0C6", Dim: "#82998B", Faint: "#55695C", Ghost: "#26372D",
		Select: "#9BE8C3", Ink: "#08130F",
		Primary: "#3DDC97", Secondary: "#E9C46A", Section: "#8FDDB8",
		Green: "#5FD39A", Yellow: "#F0C75E", Red: "#F2788A",
		Matrix: "#2E9E78",
		Grad:   [5]string{"#2BB5A0", "#3DDC97", "#9BE07A", "#D4DE6A", "#E9C46A"},
	},
	{
		ID: "brasa", Name: "Brasa", Desc: "marrom-carvão quente, acento laranja e rosé",
		BG: "#120F0D", Panel: "#1A1613", Raised: "#271F1A",
		White: "#F4ECE4", Text: "#D2C4B8", Dim: "#9C8C7E", Faint: "#6B5E53", Ghost: "#3A312A",
		Select: "#FFC29A", Ink: "#160F0A",
		Primary: "#FF8A4C", Secondary: "#F2A7A0", Section: "#F0B48A",
		Green: "#8CCB8A", Yellow: "#F5C26B", Red: "#FF6F6F",
		Matrix: "#B5703F",
		Grad:   [5]string{"#FF6B6B", "#FF8A4C", "#F5A25D", "#F5C26B", "#FFE0A3"},
	},
	{
		ID: "carvao", Name: "Carvão", Desc: "grafite neutro, acento latão (o mais discreto)",
		BG: "#0E0F11", Panel: "#16181B", Raised: "#1F2226",
		White: "#ECEAE4", Text: "#BDBAB2", Dim: "#8F8D85", Faint: "#615F5A", Ghost: "#34353A",
		Select: "#D6D1C7", Ink: "#101113",
		Primary: "#C9A66B", Secondary: "#8FA9B8", Section: "#B3A98F",
		Green: "#86B08F", Yellow: "#D1AE6B", Red: "#C77F78",
		Matrix: "#5F8F6E",
		Grad:   [5]string{"#6E7B86", "#9FB0BC", "#D9D3C7", "#C9A66B", "#A9845A"},
	},
}

// Paleta ativa (papéis). São variáveis: SetTheme as troca em tempo de execução.
var (
	BG, Panel, Raised, SelBG       fx.Color
	White, Text, Dim, Faint, Ghost fx.Color
	Select, Ink                    fx.Color
	Accent, Cyan, Purple           fx.Color // Primary, Secondary, Section
	Green, Yellow, Red             fx.Color
	Matrix                         fx.Color
	Grad                           fx.Gradient
	currentTheme                   = Themes[0].ID
)

func init() { SetTheme(Themes[0].ID) }

// FindTheme acha um tema pelo id (ou nome); ok=false se não existir.
func FindTheme(id string) (Theme, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, t := range Themes {
		if t.ID == id || strings.ToLower(t.Name) == id {
			return t, true
		}
	}
	return Theme{}, false
}

// CurrentTheme devolve o id do tema ativo.
func CurrentTheme() string { return currentTheme }

// SetTheme ativa um tema; id desconhecido é ignorado (devolve false).
func SetTheme(id string) bool {
	t, ok := FindTheme(id)
	if !ok {
		return false
	}
	h := fx.Hex
	BG, Panel, Raised, SelBG = h(t.BG), h(t.Panel), h(t.Raised), h(t.Raised)
	White, Text, Dim, Faint, Ghost = h(t.White), h(t.Text), h(t.Dim), h(t.Faint), h(t.Ghost)
	Select, Ink = h(t.Select), h(t.Ink)
	Accent, Cyan, Purple = h(t.Primary), h(t.Secondary), h(t.Section)
	Green, Yellow, Red = h(t.Green), h(t.Yellow), h(t.Red)
	Matrix = h(t.Matrix)
	Grad = fx.Gradient{h(t.Grad[0]), h(t.Grad[1]), h(t.Grad[2]), h(t.Grad[3]), h(t.Grad[4])}
	currentTheme = t.ID
	return true
}

// themeFile é onde a escolha fica guardada.
func themeFile(home string) string { return filepath.Join(home, ".config", "termux-setup", "tema") }

// LoadTheme aplica o tema salvo (ou o de TSETUP_TEMA, que vence).
func LoadTheme(home string) {
	if v := os.Getenv("TSETUP_TEMA"); v != "" && SetTheme(v) {
		return
	}
	if b, err := os.ReadFile(themeFile(home)); err == nil {
		SetTheme(string(b))
	}
}

// SaveTheme guarda o tema ativo para os próximos usos.
func SaveTheme(home string) error {
	f := themeFile(home)
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, []byte(currentTheme+"\n"), 0o644)
}
