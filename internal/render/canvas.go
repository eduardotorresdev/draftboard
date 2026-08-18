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
	"image/draw"
	"io"
	"math"

	"github.com/HugoSmits86/nativewebp"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// LimiteDeArea é o número máximo de pixels da tela de saída.
const LimiteDeArea = 64 << 20 // 67 108 864 px (~256 MB de RGBA)

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
//
// Um Canvas NÃO é seguro para uso concorrente: use um Canvas por goroutine.
// Os métodos mutam o contexto de desenho e a memória de fontes sem qualquer
// sincronização.
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

	// faces memoiza as fontes já construídas, indexadas pelo tamanho em px de
	// dispositivo, para que o mesmo tamanho não seja reconstruído a cada linha.
	faces map[float64]font.Face
}

// NewCanvas cria a tela. l e a são as dimensões do Frame em px. As quatro
// margens são o Chrome em px do espaço do Frame; podem ser 0. escala multiplica
// tudo. O Chrome é pintado com scene.TomChrome e o Frame com scene.TomFrame.
//
// As dimensões finais em pixels são o produto arredondado para o inteiro mais
// próximo, de modo que fatores de escala não-inteiros funcionem.
//
// A tela satura em LimiteDeArea pixels: pedida uma área maior, a escala efetiva
// é reduzida até caber, preservando a proporção, em vez de alocar sem teto. A
// CLI recusa esse caso antes de chegar aqui (ver CONTRACT.md §5b); a saturação
// existe para que a biblioteca jamais entre em pânico nem vá para o swap.
func NewCanvas(l, a int, margemT, margemD, margemB, margemE, escala float64) *Canvas {
	fl, fa := float64(l), float64(a)
	larguraTotal := margemE + fl + margemD
	alturaTotal := margemT + fa + margemB

	escala = escalaQueCabeNoTeto(larguraTotal, alturaTotal, escala)
	telaL, telaA := dimensoesDaTela(larguraTotal, alturaTotal, escala)

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
//
// A bounding box é recortada ao retângulo do Frame ANTES de rasterizar. Além de
// evitar trabalho invisível, isso é o que impede o laço de CPU sem fim do
// rasterizador quando a extensão do Elemento passa de 2^25 px de dispositivo.
func (c *Canvas) DesenhaElemento(e scene.Elemento) {
	if !finito(e.X) || !finito(e.Y) || !finito(e.L) || !finito(e.A) {
		return
	}
	if e.L <= 0 || e.A <= 0 {
		// Elemento de área zero não pinta nada.
		return
	}

	x := (c.margemE + e.X) * c.escala
	y := (c.margemT + e.Y) * c.escala
	l := e.L * c.escala
	a := e.A * c.escala

	fx0, fy0, fx1, fy1 := c.retanguloDoFrame()
	// Elemento inteiramente fora do Frame: a máscara não deixaria passar nada.
	if x >= fx1 || y >= fy1 || x+l <= fx0 || y+a <= fy0 {
		return
	}

	// O Rótulo não é um caminho a preencher: sai por glifos, com máscara
	// própria, e por isso não passa pelo Fill comum às outras Formas.
	if e.Forma == scene.Texto {
		c.desenhaRotulo(e, x, y, l, a)
		return
	}

	// A máscara tem sempre o tamanho da tela, então SetMask não pode falhar.
	_ = c.dc.SetMask(c.mascaraDoFrame())
	c.dc.SetColor(cor(e.Tom))

	switch e.Forma {
	case scene.Circulo:
		c.tracaCirculo(x, y, l, a, fx0, fy0, fx1, fy1)
	default:
		c.tracaRetangulo(e, x, y, l, a, fx0, fy0, fx1, fy1)
	}

	c.dc.Fill()
	c.dc.ResetClip()
}

// mascaraDaArea devolve a máscara do retângulo dado, já interseccionado com o
// Frame. É o recorte do Rótulo: texto mais largo que a área que lhe coube é
// cortado nela, e nunca vaza para a Superfície vizinha nem para o Chrome.
func (c *Canvas) mascaraDaArea(x, y, l, a float64) *image.Alpha {
	larg, alt := c.dc.Width(), c.dc.Height()
	fx0, fy0, fx1, fy1 := c.retanguloDoFrame()
	m := image.NewAlpha(image.Rect(0, 0, larg, alt))
	r := image.Rect(
		pixelSeguro(math.Max(x, fx0), larg), pixelSeguro(math.Max(y, fy0), alt),
		pixelSeguro(math.Min(x+l, fx1), larg), pixelSeguro(math.Min(y+a, fy1), alt),
	)
	draw.Draw(m, r, image.NewUniform(color.Alpha{A: 0xFF}), image.Point{}, draw.Src)
	return m
}

// tracaRetangulo traça o Retângulo já recortado ao Frame. O recorte é exato:
// um Retângulo cortado continua um Retângulo. No caso arredondado a caixa é
// folgada pelo raio, de modo que os cantos criados pelo corte caem fora do
// Frame e nunca chegam a aparecer.
func (c *Canvas) tracaRetangulo(e scene.Elemento, x, y, l, a, fx0, fy0, fx1, fy1 float64) {
	r, folga := 0.0, 0.0
	if e.Arredondado {
		r = c.raio(e)
		// A folga é o raio: os cantos que o corte cria ficam a pelo menos um
		// raio da borda do Frame e portanto nunca aparecem. O pixel a mais é
		// margem para a fronteira que cai em fração de pixel.
		folga = r + 1
	}

	x0, y0 := math.Max(x, fx0-folga), math.Max(y, fy0-folga)
	x1, y1 := math.Min(x+l, fx1+folga), math.Min(y+a, fy1+folga)

	if r > 0 {
		c.dc.DrawRoundedRectangle(x0, y0, x1-x0, y1-y0, r)
		return
	}
	c.dc.DrawRectangle(x0, y0, x1-x0, y1-y0)
}

// tracaCirculo traça o Círculo. O Círculo é definido por um único diâmetro e a
// resolução garante L == A; tomamos o menor dos dois lados para que ele
// continue redondo mesmo num Frame não-quadrado, jamais virando elipse.
func (c *Canvas) tracaCirculo(x, y, l, a, fx0, fy0, fx1, fy1 float64) {
	cx, cy := x+l/2, y+a/2
	r := math.Min(l, a) / 2

	// O rasterizador percorre a extensão inteira do caminho, linha a linha, e o
	// ponto fixo do freetype ainda satura em 2^25 px. Um Círculo muito maior que
	// a tela é traçado como polígono limitado à faixa do Frame: o desenho onde
	// ele aparece é o mesmo, e o custo passa a ser proporcional ao Frame em vez
	// de ao diâmetro.
	if r > c.limiteDeTracado() {
		c.tracaCirculoGigante(cx, cy, r, fx0, fy0, fx1, fy1)
		return
	}
	c.dc.DrawCircle(cx, cy, r)
}

// limiteDeTracado é a maior extensão que vale a pena entregar ao rasterizador:
// além disso o caminho é muito maior que a tela e só gera trabalho invisível.
func (c *Canvas) limiteDeTracado() float64 {
	maior := c.dc.Width()
	if c.dc.Height() > maior {
		maior = c.dc.Height()
	}
	return 4 * float64(maior)
}

// tracaCirculoGigante aproxima um Círculo enorme por um polígono restrito à
// faixa de linhas do Frame. A borda é amostrada a cada linha da tela e os
// pontos são presos às bordas do Frame, então dentro do Frame o resultado é
// indistinguível do Círculo real — a flecha de cada segmento de uma linha de
// altura é 1/(8r) de pixel.
func (c *Canvas) tracaCirculoGigante(cx, cy, r, fx0, fy0, fx1, fy1 float64) {
	topo := math.Max(fy0-1, cy-r)
	base := math.Min(fy1+1, cy+r)
	if topo > base {
		return
	}
	esqLim, dirLim := fx0-1, fx1+1

	n := int(math.Ceil(base-topo)) + 1
	if n < 2 {
		n = 2
	}

	direita := make([][2]float64, n)
	esquerda := make([][2]float64, n)
	for i := 0; i < n; i++ {
		y := topo + (base-topo)*float64(i)/float64(n-1)
		dy := y - cy
		meia := 0.0
		if s := r*r - dy*dy; s > 0 {
			meia = math.Sqrt(s)
		}
		direita[i] = [2]float64{preso(cx+meia, esqLim, dirLim), y}
		esquerda[i] = [2]float64{preso(cx-meia, esqLim, dirLim), y}
	}

	c.dc.MoveTo(esquerda[0][0], esquerda[0][1])
	for _, p := range direita {
		c.dc.LineTo(p[0], p[1])
	}
	for i := len(esquerda) - 1; i >= 0; i-- {
		c.dc.LineTo(esquerda[i][0], esquerda[i][1])
	}
	// Fechar é redundante — gg.Fill fecha subcaminhos abertos por conta
	// própria — mas deixa o polígono explícito na leitura.
	c.dc.ClosePath()
}

// preso limita v ao intervalo [min, max].
func preso(v, min, max float64) float64 {
	if !finito(v) {
		if math.IsInf(v, -1) {
			return min
		}
		return max
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
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

// Raio devolve o raio de canto de um Retângulo arredondado de l por a, em px do
// espaço do Frame — antes de qualquer fator de escala. É exportada porque a
// Prancheta desenha os mesmos Retângulos em SVG: a regra do canto vive num
// lugar só, e raster e vetor nunca divergem. Ver raioBase e raioFracaoMaxima.
func Raio(l, a float64) float64 {
	r := raioBase
	if limite := math.Min(l, a) * raioFracaoMaxima; limite < r {
		r = limite
	}
	return r
}

// raio devolve o raio de canto de um Retângulo arredondado, em px de
// dispositivo.
func (c *Canvas) raio(e scene.Elemento) float64 {
	return Raio(e.L, e.A) * c.escala
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
// primeira chamada. O recorte é um retângulo alinhado aos eixos, então basta
// preencher o alfa direto: não há caminho a rasterizar, e a fronteira sai dura
// mesmo quando as margens caem em fração de pixel.
func (c *Canvas) mascaraDoFrame() *image.Alpha {
	if c.recorte == nil {
		larg, alt := c.dc.Width(), c.dc.Height()
		m := image.NewAlpha(image.Rect(0, 0, larg, alt))
		x0, y0, x1, y1 := c.retanguloDoFrame()
		r := image.Rect(
			pixelSeguro(x0, larg), pixelSeguro(y0, alt),
			pixelSeguro(x1, larg), pixelSeguro(y1, alt),
		)
		draw.Draw(m, r, image.NewUniform(color.Alpha{A: 0xFF}), image.Point{}, draw.Src)
		c.recorte = m
	}
	return c.recorte
}

// escalaQueCabeNoTeto reduz a escala até que a tela caiba em LimiteDeArea,
// preservando a proporção. Escala que já cabe volta intacta.
func escalaQueCabeNoTeto(largura, altura, escala float64) float64 {
	// NaN e escala não-positiva não têm tela possível: a conversão para pixels
	// resolve devolvendo a tela mínima. +Inf, esse sim, precisa saturar — é o
	// caminho que levaria a alocar sem teto.
	if math.IsNaN(escala) || escala <= 0 || largura <= 0 || altura <= 0 {
		return escala
	}
	maxEscala := math.Sqrt(LimiteDeArea / (largura * altura))
	if escala > maxEscala {
		return maxEscala
	}
	return escala
}

// ladoDaTela converte uma medida do espaço do Frame em pixels da tela: o
// inteiro mais próximo, nunca menor que 1 nem maior que o teto de área — um
// lado sozinho já não pode passar disso. Medida não-numérica vira 1.
func ladoDaTela(v float64) int {
	if math.IsNaN(v) || v <= 1 {
		return 1
	}
	if v >= LimiteDeArea {
		return LimiteDeArea
	}
	return int(math.Round(v))
}

// dimensoesDaTela converte as medidas do espaço do Frame em pixels da tela.
// Além do arredondamento, garante — aconteça o que acontecer com a escala —
// que a tela caiba em LimiteDeArea, preservando a proporção.
func dimensoesDaTela(largura, altura, escala float64) (l, a int) {
	l, a = ladoDaTela(largura*escala), ladoDaTela(altura*escala)

	if l > LimiteDeArea/a {
		fator := math.Sqrt(float64(LimiteDeArea) / (float64(l) * float64(a)))
		l, a = int(float64(l)*fator), int(float64(a)*fator)
		if l < 1 {
			l = 1
		}
		if a < 1 {
			a = 1
		}
	}
	return l, a
}

// cor devolve o cinza opaco de um Tom. A imagem inteira é escala de cinza: os
// três canais recebem o mesmo valor.
func cor(t scene.Tom) color.Color {
	v := t.Cinza()
	return color.NRGBA{R: v, G: v, B: v, A: 0xFF}
}

// pixelSeguro arredonda uma coordenada de dispositivo para um índice de pixel
// dentro de [0, max]. Coordenada não-numérica vira 0.
func pixelSeguro(v float64, max int) int {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}
	if v >= float64(max) {
		return max
	}
	return int(math.Round(v))
}

// finito diz se o valor é um número utilizável como coordenada.
func finito(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
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
