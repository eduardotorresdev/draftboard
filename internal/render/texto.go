package render

import (
	"strings"
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// A fonte do plano de anotação é a Go Regular embutida no binário: sem arquivo
// externo, sem cgo, e o mesmo desenho em qualquer plataforma.
var (
	fonteUmaVez sync.Once
	fonte       *truetype.Font
)

func fonteBase() *truetype.Font {
	fonteUmaVez.Do(func() {
		f, err := truetype.Parse(goregular.TTF)
		if err != nil {
			// A fonte é embutida em tempo de compilação; falhar aqui é bug.
			panic("render: fonte embutida inválida: " + err.Error())
		}
		fonte = f
	})
	return fonte
}

// Texto escreve uma linha de texto no plano de anotação. As coordenadas são
// relativas ao canto superior esquerdo da tela inteira, Chrome incluso, e y é o
// TOPO da linha, não a linha de base.
func (c *Canvas) Texto(x, y float64, s string, tamanho float64, t scene.Tom) {
	if s == "" {
		return
	}
	f := c.face(tamanho)
	subida, _ := metricas(f)

	c.dc.SetFontFace(f)
	c.dc.SetColor(cor(t))
	c.dc.DrawString(s, x*c.escala, y*c.escala+subida)
}

// MedeTexto devolve a largura e a altura de uma linha de texto, em px do espaço
// do Frame. A altura é a caixa de linha inteira — subida mais descida — para
// que empilhar linhas de MedeTexto em altura nunca as faça se tocar.
func (c *Canvas) MedeTexto(s string, tamanho float64) (l, a float64) {
	f := c.face(tamanho)
	_, altura := metricas(f)
	return c.paraOFrame(c.larguraDeDispositivo(f, s)), c.paraOFrame(altura)
}

// paraOFrame converte px de dispositivo de volta ao espaço do Frame. Escala
// não utilizável devolve 0 em vez de infinito.
func (c *Canvas) paraOFrame(v float64) float64 {
	if !finito(c.escala) || c.escala <= 0 {
		return 0
	}
	return v / c.escala
}

// QuebraTexto quebra o texto em linhas que cabem em larguraMax, quebrando
// somente entre palavras. Nunca trunca: uma palavra mais larga que larguraMax
// ocupa sozinha a sua linha e transborda, em vez de perder caracteres. Quebras
// de linha explícitas no texto original são preservadas.
func (c *Canvas) QuebraTexto(s string, tamanho, larguraMax float64) []string {
	if s == "" {
		return nil
	}
	f := c.face(tamanho)
	maxDispositivo := larguraMax * c.escala

	var linhas []string
	for _, paragrafo := range strings.Split(s, "\n") {
		palavras := strings.Fields(paragrafo)
		if len(palavras) == 0 {
			linhas = append(linhas, "")
			continue
		}
		linha := palavras[0]
		for _, palavra := range palavras[1:] {
			candidata := linha + " " + palavra
			if c.larguraDeDispositivo(f, candidata) <= maxDispositivo {
				linha = candidata
				continue
			}
			linhas = append(linhas, linha)
			linha = palavra
		}
		linhas = append(linhas, linha)
	}
	return linhas
}

// face devolve a fonte no tamanho pedido, convertido para px de dispositivo.
func (c *Canvas) face(tamanho float64) font.Face {
	td := tamanho * c.escala
	if !finito(td) || td <= 0 {
		// Tamanho não-positivo, NaN ou infinito não tem desenho possível.
		// Normalizamos para 1 — se não, a chave nunca acerta a memória (NaN
		// não é igual a si mesmo) e o mapa cresceria sem limite.
		td = 1
	}
	if f, ok := c.faces[td]; ok {
		return f
	}
	f := truetype.NewFace(fonteBase(), &truetype.Options{
		Size:    td,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	c.faces[td] = f
	return f
}

// larguraDeDispositivo mede o avanço de uma linha em px de dispositivo.
func (c *Canvas) larguraDeDispositivo(f font.Face, s string) float64 {
	d := &font.Drawer{Face: f}
	return float64(d.MeasureString(s)) / 64
}

// metricas devolve, em px de dispositivo, a subida (do topo da linha até a
// linha de base) e a altura total da caixa de linha.
func metricas(f font.Face) (subida, altura float64) {
	m := f.Metrics()
	subida = float64(m.Ascent) / 64
	descida := float64(m.Descent) / 64
	return subida, subida + descida
}
