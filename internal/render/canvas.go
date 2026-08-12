// Package render rasteriza um Frame já resolvido: recebe geometria em pixels
// absolutos, com Elevação e Tom já calculados, e produz uma imagem WebP
// lossless em escala de cinza.
//
// A tela é composta por duas regiões: o Frame, pintado com scene.TomFrame, e o
// Chrome ao redor dele, pintado com scene.TomChrome. Elementos vivem no espaço
// do Frame e são recortados nele; as primitivas do plano de anotação vivem no
// espaço da tela inteira e alcançam o Chrome.
//
// Este pacote nunca importa internal/notes.
package render

import (
	"image"
	"image/color"
	"io"
	"math"

	"github.com/HugoSmits86/nativewebp"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

const (
	// raioBase é o raio dos cantos arredondados de um Retângulo, em pixels do
	// espaço do Frame (antes da escala). É constante em toda a tela: dois
	// Retângulos de tamanhos diferentes recebem exatamente o mesmo raio, e só
	// o limite abaixo pode reduzi-lo. Como todo o desenho é multiplicado pelo
	// fator de escala, o raio escala junto e a aparência não muda com a escala.
	raioBase = 8.0

	// raioFracaoMaxima limita o raio a esta fração do menor lado do Retângulo.
	// Um Retângulo vira pílula quando o raio chega à METADE do menor lado;
	// parando em um quarto, o canto continua visivelmente arredondado e um
	// Retângulo pequeno nunca vira pílula por acidente.
	raioFracaoMaxima = 0.25
)

// Canvas é a tela de saída. Todas as coordenadas dos métodos são em pixels do
// espaço do Frame (antes da escala); o Canvas aplica o fator internamente.
type Canvas struct {
	dc     *gg.Context
	escala float64

	// l e a são as dimensões do Frame em px do espaço do Frame.
	l, a float64
	// margens delimitam o Chrome, em px do espaço do Frame.
	margemT, margemD, margemB, margemE float64

	// recorte é a máscara do retângulo do Frame. É calculada uma única vez e
	// reaproveitada por todo DesenhaElemento.
	recorte *image.Alpha

	// faces memoriza as fontes já construídas, indexadas pelo tamanho em px de
	// dispositivo, para que o mesmo tamanho não seja reconstruído a cada linha.
	faces map[float64]font.Face
}

// NewCanvas cria a tela. l e a são as dimensões do Frame em px. As quatro
// margens são o Chrome em px do espaço do Frame; podem ser 0. escala multiplica
// tudo. O Chrome é pintado com scene.TomChrome e o Frame com scene.TomFrame.
//
// As dimensões finais em pixels são o produto arredondado para o inteiro mais
// próximo, de modo que fatores de escala não-inteiros funcionem.
func NewCanvas(l, a int, margemT, margemD, margemB, margemE, escala float64) *Canvas {
	fl, fa := float64(l), float64(a)

	telaL := arredonda((margemE + fl + margemD) * escala)
	telaA := arredonda((margemT + fa + margemB) * escala)
	// Uma tela sem pixel nenhum não é codificável; garantimos ao menos 1x1.
	if telaL < 1 {
		telaL = 1
	}
	if telaA < 1 {
		telaA = 1
	}

	c := &Canvas{
		dc:      gg.NewContext(telaL, telaA),
		escala:  escala,
		l:       fl,
		a:       fa,
		margemT: margemT,
		margemD: margemD,
		margemB: margemB,
		margemE: margemE,
		faces:   make(map[float64]font.Face),
	}

	// O Chrome cobre a tela inteira e o Frame é pintado por cima dele: a área
	// das margens fica com TomChrome e a área do Frame com TomFrame.
	c.dc.SetColor(cor(scene.TomChrome))
	c.dc.Clear()

	x0, y0, x1, y1 := c.retanguloDoFrame()
	c.dc.SetColor(cor(scene.TomFrame))
	c.dc.DrawRectangle(x0, y0, x1-x0, y1-y0)
	c.dc.Fill()

	return c
}

// DesenhaElemento pinta um Elemento resolvido, recortado ao Frame. O Elemento
// é sólido, no Tom que já traz consigo, e é posicionado no espaço do Frame —
// portanto deslocado pelas margens esquerda e superior. Um Elemento que
// ultrapassa a borda é cortado e nunca invade o Chrome.
func (c *Canvas) DesenhaElemento(e scene.Elemento) {
	if e.L <= 0 || e.A <= 0 {
		// Elemento de área zero não pinta nada.
		return
	}

	x := (c.margemE + e.X) * c.escala
	y := (c.margemT + e.Y) * c.escala
	l := e.L * c.escala
	a := e.A * c.escala

	// A máscara tem sempre o tamanho da tela, então SetMask não pode falhar.
	_ = c.dc.SetMask(c.mascaraDoFrame())
	c.dc.SetColor(cor(e.Tom))

	switch e.Forma {
	case scene.Circulo:
		// O Círculo é definido por um único diâmetro e a resolução garante
		// L == A. Tomamos o menor dos dois lados para que ele continue redondo
		// mesmo num Frame não-quadrado, jamais virando elipse.
		d := math.Min(l, a)
		c.dc.DrawCircle(x+l/2, y+a/2, d/2)
	default:
		if e.Arredondado {
			c.dc.DrawRoundedRectangle(x, y, l, a, c.raio(e))
		} else {
			c.dc.DrawRectangle(x, y, l, a)
		}
	}

	c.dc.Fill()
	c.dc.ResetClip()
}

// Retangulo pinta um retângulo sólido no plano de anotação. As coordenadas são
// relativas ao canto superior esquerdo da tela inteira, Chrome incluso.
func (c *Canvas) Retangulo(x, y, l, a float64, t scene.Tom) {
	if l <= 0 || a <= 0 {
		return
	}
	c.dc.SetColor(cor(t))
	c.dc.DrawRectangle(x*c.escala, y*c.escala, l*c.escala, a*c.escala)
	c.dc.Fill()
}

// Linha traça uma linha reta no plano de anotação. As coordenadas e a espessura
// são relativas à tela inteira, Chrome incluso.
func (c *Canvas) Linha(x1, y1, x2, y2, espessura float64, t scene.Tom) {
	if espessura <= 0 {
		return
	}
	c.dc.SetColor(cor(t))
	c.dc.SetLineWidth(espessura * c.escala)
	c.dc.DrawLine(x1*c.escala, y1*c.escala, x2*c.escala, y2*c.escala)
	c.dc.Stroke()
}

// OrigemDoFrame devolve o deslocamento do canto superior esquerdo do Frame
// dentro da tela, em px do espaço do Frame.
func (c *Canvas) OrigemDoFrame() (x, y float64) {
	return c.margemE, c.margemT
}

// CodificaWebP escreve a tela como WebP lossless. É determinístico: a mesma
// entrada produz bytes idênticos.
func (c *Canvas) CodificaWebP(w io.Writer) error {
	// O codificador ignora o erro de escrita, então guardamos o primeiro por
	// conta própria para não relatar sucesso sobre um destino quebrado.
	e := &escritor{w: w}
	if err := nativewebp.Encode(e, c.dc.Image(), nil); err != nil {
		return err
	}
	return e.err
}

// DesenhaFrame cria o Canvas, pinta o fundo e todos os Elementos das Camadas de
// índice 0 até ateCamada inclusive, na ordem declarada; ateCamada < 0 significa
// todas as Camadas. É isso que sustenta o export por Camada cumulativo: cada
// imagem contém a Camada pedida e todas abaixo dela.
func DesenhaFrame(f scene.Frame, escala float64, margemT, margemD, margemB, margemE float64, ateCamada int) *Canvas {
	c := NewCanvas(f.L, f.A, margemT, margemD, margemB, margemE, escala)

	ate := ateCamada
	if ate < 0 || ate >= len(f.Camadas) {
		ate = len(f.Camadas) - 1
	}
	for i := 0; i <= ate; i++ {
		for _, e := range f.Camadas[i].Elementos {
			c.DesenhaElemento(e)
		}
	}
	return c
}

// raio devolve o raio de canto de um Retângulo arredondado, em px de
// dispositivo. Ver raioBase e raioFracaoMaxima.
func (c *Canvas) raio(e scene.Elemento) float64 {
	r := raioBase
	if limite := math.Min(e.L, e.A) * raioFracaoMaxima; limite < r {
		r = limite
	}
	return r * c.escala
}

// retanguloDoFrame devolve o retângulo do Frame na tela, em px de dispositivo.
func (c *Canvas) retanguloDoFrame() (x0, y0, x1, y1 float64) {
	x0 = c.margemE * c.escala
	y0 = c.margemT * c.escala
	x1 = (c.margemE + c.l) * c.escala
	y1 = (c.margemT + c.a) * c.escala
	return x0, y0, x1, y1
}

// mascaraDoFrame devolve a máscara de recorte do Frame, construindo-a na
// primeira chamada.
func (c *Canvas) mascaraDoFrame() *image.Alpha {
	if c.recorte == nil {
		m := gg.NewContext(c.dc.Width(), c.dc.Height())
		x0, y0, x1, y1 := c.retanguloDoFrame()
		m.SetColor(color.Opaque)
		m.DrawRectangle(x0, y0, x1-x0, y1-y0)
		m.Fill()
		c.recorte = m.AsMask()
	}
	return c.recorte
}

// cor devolve o cinza opaco de um Tom. A imagem inteira é escala de cinza: os
// três canais recebem o mesmo valor.
func cor(t scene.Tom) color.Color {
	v := t.Cinza()
	return color.NRGBA{R: v, G: v, B: v, A: 0xFF}
}

// arredonda converte px de dispositivo fracionários no inteiro mais próximo.
// É a regra única de arredondamento de toda a rasterização.
func arredonda(v float64) int {
	return int(math.Round(v))
}

// escritor guarda o primeiro erro de escrita, já que o codificador WebP não os
// propaga.
type escritor struct {
	w   io.Writer
	err error
}

func (e *escritor) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	n, err := e.w.Write(p)
	if err != nil {
		e.err = err
	}
	return n, err
}
