// Package render é o Canvas de saída. Este arquivo é um esqueleto de
// compilação escrito por F1 para fiar a CLI; F3 o substitui pela implementação
// real. Nada aqui desenha.
package render

import (
	"io"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Canvas é a tela de saída. Todas as coordenadas dos métodos são em pixels do
// espaço do Frame (antes da escala); o Canvas aplica o fator internamente.
type Canvas struct{}

// NewCanvas cria a tela. l e a são as dimensões do Frame em px. As quatro
// margens são o Chrome em px do espaço do Frame; podem ser 0. escala multiplica
// tudo. O Chrome é pintado com scene.TomChrome e o Frame com scene.TomFrame.
func NewCanvas(l, a int, margemT, margemD, margemB, margemE, escala float64) *Canvas {
	return &Canvas{}
}

// DesenhaElemento pinta um Elemento resolvido, recortado ao Frame.
func (c *Canvas) DesenhaElemento(e scene.Elemento) {}

// Retangulo pinta um retângulo no plano de anotação.
func (c *Canvas) Retangulo(x, y, l, a float64, t scene.Tom) {}

// Linha pinta uma linha no plano de anotação.
func (c *Canvas) Linha(x1, y1, x2, y2, espessura float64, t scene.Tom) {}

// Texto pinta uma linha de texto no plano de anotação. y é o topo da linha.
func (c *Canvas) Texto(x, y float64, s string, tamanho float64, t scene.Tom) {}

// MedeTexto devolve as dimensões de s no tamanho dado.
func (c *Canvas) MedeTexto(s string, tamanho float64) (l, a float64) { return 0, 0 }

// QuebraTexto quebra s em linhas que cabem em larguraMax.
func (c *Canvas) QuebraTexto(s string, tamanho, larguraMax float64) []string { return nil }

// OrigemDoFrame devolve o deslocamento do canto superior esquerdo do Frame
// dentro da tela, em px do espaço do Frame (= margemE, margemT).
func (c *Canvas) OrigemDoFrame() (x, y float64) { return 0, 0 }

// CodificaWebP escreve WebP lossless. Determinístico: mesma entrada, mesmos
// bytes.
func (c *Canvas) CodificaWebP(w io.Writer) error { return nil }

// DesenhaFrame cria o Canvas, pinta o fundo e todos os Elementos das Camadas
// indicadas (todas quando ateCamada < 0), e devolve o Canvas para anotação.
func DesenhaFrame(f scene.Frame, escala float64, margemT, margemD, margemB, margemE float64, ateCamada int) *Canvas {
	return &Canvas{}
}
