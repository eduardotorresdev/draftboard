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
// relativas ao canto superior esquerdo da tela inteira, e não ao Frame, e y é o
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

// limiteDaFonte é o maior tamanho de fonte, em px de dispositivo, que se pode
// pedir ao freetype.
//
// `truetype.NewFace` não aloca a máscara de UM glifo: aloca a de 512, o tamanho
// do cache de glifos da face. O custo é `512 * (k*S)²` bytes, com S o tamanho da
// fonte e k a proporção entre a caixa do maior glifo e o corpo — cerca de 1,3 na
// Go Regular. Sem teto, a régua das Notas de um `--scale 10000` pede uma fonte
// de cem mil px e o processo morre por falta de memória ANTES de o teto de área
// do §8 chegar a recusar a tela.
//
// 256 px deixa o cache em ~56 MB, a mesma ordem de grandeza da tela que o teto
// de área já permite. Nenhum desenho legítimo chega perto: a Nota é um corpo
// fixo pequeno, e o Rótulo de Controle acima disso exigiria um Controle de
// centenas de px de altura em escala alta, que o teto de área já recusa.
const limiteDaFonte = 256

// face devolve a fonte no tamanho pedido, convertido para px de dispositivo.
func (c *Canvas) face(tamanho float64) font.Face {
	td := tamanho * c.escala
	if !finito(td) || td <= 0 {
		// Tamanho não-positivo, NaN ou infinito não tem desenho possível.
		// Normalizamos para 1 — se não, a chave nunca acerta a memória (NaN
		// não é igual a si mesmo) e o mapa cresceria sem limite.
		td = 1
	}
	if td > limiteDaFonte {
		td = limiteDaFonte
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

// fracaoDoRotulo é a altura da fonte do Rótulo como fração da altura da área
// que a resolução lhe reservou. É constante de propósito: o tamanho do texto é
// derivado da estrutura, nunca declarado, pela mesma razão que o Tom é.
const fracaoDoRotulo = 0.45

// TamanhoDoRotulo devolve a altura da fonte do Rótulo de um Elemento de altura
// a, em px do espaço do Frame. É exportada pela mesma razão que Raio: a
// Prancheta desenha o mesmo Rótulo em SVG e não pode ter uma segunda regra.
func TamanhoDoRotulo(a float64) float64 {
	return fracaoDoRotulo * a
}

// desenhaRotulo pinta o Rótulo de um Elemento de Forma Texto dentro da área do
// próprio Elemento, no espaço do Frame. As coordenadas chegam já em px de
// dispositivo.
//
// A resolução entrega a área e o alinhamento, nunca a largura do texto: medir
// glifos exige a fonte, e manter a fonte fora da resolução é o que impede o
// freetype de virar dependência de quem só calcula geometria.
func (c *Canvas) desenhaRotulo(e scene.Elemento, x, y, l, a float64) {
	if e.Rotulo == "" {
		return
	}
	tamanho := fracaoDoRotulo * e.A
	if !finito(tamanho) || tamanho <= 0 || tamanho*c.escala > c.limiteDeTracado() {
		return
	}

	f := c.face(tamanho)
	subida, altura := metricas(f)

	// A linha fica centralizada na vertical em qualquer alinhamento: só o eixo
	// horizontal é escolha do Controle.
	base := y + (a-altura)/2 + subida
	inicio := x
	if e.Alinhamento == scene.AoCentro {
		inicio = x + (l-c.larguraDeDispositivo(f, e.Rotulo))/2
	}

	_ = c.dc.SetMask(c.mascaraDaArea(x, y, l, a))
	c.dc.SetFontFace(f)
	c.dc.SetColor(cor(e.Tom))
	c.dc.DrawString(e.Rotulo, inicio, base)
	c.dc.ResetClip()
}
