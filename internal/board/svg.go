package board

import (
	"bufio"
	"fmt"
	"html"
	"math"
	"strconv"

	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// escreveSVG desenha a Prancheta inteira: os Frames em ordem de declaração e,
// por cima de todos eles, as Ligações.
func escreveSVG(b *bufio.Writer, d *scene.Documento, posicoes []posicao, ligacoes []ligacao) {
	l, a := mundo(d, posicoes)
	fmt.Fprintf(b, "<svg id=\"mundo\" xmlns=\"http://www.w3.org/2000/svg\" data-largura=\"%s\" data-altura=\"%s\">\n",
		num(l), num(a))
	fmt.Fprintf(b, "<defs>\n")
	fmt.Fprintf(b, "<marker id=\"ponta\" viewBox=\"0 0 10 10\" refX=\"9\" refY=\"5\" markerWidth=\"7\" markerHeight=\"7\" orient=\"auto-start-reverse\">\n")
	fmt.Fprintf(b, "<path d=\"M0,0 L10,5 L0,10 z\" fill=\"%s\"/>\n</marker>\n", cor(scene.TomChrome))
	for i, f := range d.Frames {
		fmt.Fprintf(b, "<clipPath id=\"corte-%d\"><rect x=\"0\" y=\"0\" width=\"%s\" height=\"%s\"/></clipPath>\n",
			i, num(float64(f.L)), num(float64(f.A)))
	}
	fmt.Fprintf(b, "</defs>\n")
	fmt.Fprintf(b, "<g id=\"camera\">\n")

	for i, f := range d.Frames {
		escreveFrame(b, i, f, posicoes[i])
	}

	fmt.Fprintf(b, "<g id=\"ligacoes\">\n")
	for _, lg := range ligacoes {
		escreveLigacao(b, d, posicoes, lg)
	}
	fmt.Fprintf(b, "</g>\n</g>\n</svg>\n")
}

// escreveFrame desenha um Frame: o título acima, o fundo, e os Elementos de
// cada Camada na ordem de pintura, recortados na borda do Frame.
func escreveFrame(b *bufio.Writer, indice int, f scene.Frame, p posicao) {
	fmt.Fprintf(b, "<g class=\"frame\" data-frame=\"%d\" data-nome=\"%s\" data-x=\"%s\" data-y=\"%s\" data-l=\"%d\" data-a=\"%d\" transform=\"translate(%s,%s)\">\n",
		indice, escapa(f.Nome), num(p.X), num(p.Y), f.L, f.A, num(p.X), num(p.Y))
	fmt.Fprintf(b, "<text class=\"titulo\" x=\"0\" y=\"-14\">%s <tspan class=\"dim\">%d&times;%d</tspan></text>\n",
		escapa(f.Nome), f.L, f.A)
	fmt.Fprintf(b, "<rect class=\"realce\" x=\"-6\" y=\"-6\" width=\"%s\" height=\"%s\"/>\n",
		num(float64(f.L)+12), num(float64(f.A)+12))
	fmt.Fprintf(b, "<g clip-path=\"url(#corte-%d)\">\n", indice)
	fmt.Fprintf(b, "<rect x=\"0\" y=\"0\" width=\"%s\" height=\"%s\" fill=\"%s\"/>\n",
		num(float64(f.L)), num(float64(f.A)), cor(scene.TomFrame))
	// rotulos numera os recortes de Rótulo dentro do Frame: cada um precisa do
	// seu, porque a área recortada é a do Elemento.
	rotulos := 0
	for _, c := range f.Camadas {
		for _, e := range c.Elementos {
			escreveElemento(b, indice, &rotulos, c.Nome, e)
		}
	}
	fmt.Fprintf(b, "</g>\n</g>\n")
}

// escreveElemento desenha um Elemento. As peças internas de um Controle são
// desenhadas — elas existem no desenho — mas não recebem clique: o Controle é
// fechado, e quem clica nele seleciona o Controle, não o seu miolo.
func escreveElemento(b *bufio.Writer, frame int, rotulos *int, camada string, e scene.Elemento) {
	atributos := fmt.Sprintf(" data-caminho=\"%s\" data-camada=\"%s\" data-forma=\"%s\" data-tom=\"%d\" data-elev=\"%d\"",
		escapa(e.Caminho), escapa(camada), e.Forma, int(e.Tom), e.Elevacao)
	atributos += fmt.Sprintf(" data-geo=\"%s,%s %s&times;%s\"", num(e.X), num(e.Y), num(e.L), num(e.A))
	if e.Controle != "" {
		atributos += fmt.Sprintf(" data-controle=\"%s\"", escapa(e.Controle))
	}
	if e.Detalhe != "" {
		atributos += fmt.Sprintf(" data-detalhe=\"%s\"", escapa(e.Detalhe))
	}
	if e.Origem != "" {
		atributos += fmt.Sprintf(" data-origem=\"%s\"", escapa(e.Origem))
	}
	if e.Nota != "" {
		atributos += fmt.Sprintf(" data-nota=\"%s\"", escapa(e.Nota))
	}
	if e.Destino != "" {
		atributos += fmt.Sprintf(" data-para=\"%s\"", escapa(e.Destino))
	}
	classe := "peca"
	if e.Interno {
		classe += " interno"
	}
	if e.Nota != "" {
		classe += " anotado"
	}
	if e.Destino != "" {
		classe += " gatilho"
	}

	abre := func(tag, geometria string) {
		fmt.Fprintf(b, "<%s class=\"%s\" %s fill=\"%s\"%s>", tag, classe, geometria, cor(e.Tom), atributos)
		if e.Nota != "" {
			fmt.Fprintf(b, "<title>%s</title>", escapa(e.Nota))
		}
		fmt.Fprintf(b, "</%s>\n", tag)
	}

	switch e.Forma {
	case scene.Circulo:
		abre("circle", fmt.Sprintf("cx=\"%s\" cy=\"%s\" r=\"%s\"",
			num(e.X+e.L/2), num(e.Y+e.A/2), num(e.L/2)))
	case scene.Texto:
		if e.Rotulo == "" {
			return
		}
		tamanho := render.TamanhoDoRotulo(e.A)
		if !finito(tamanho) || tamanho <= 0 {
			return
		}
		x, ancora := e.X, "start"
		if e.Alinhamento == scene.AoCentro {
			x, ancora = e.X+e.L/2, "middle"
		}
		// O Rótulo é recortado na sua própria área, e não só na borda do
		// Frame: é o que o raster já faz com mascaraDaArea. Sem isto, um
		// Rótulo mais largo que o bloco sai cortado no WebP e inteiro na
		// Prancheta, por cima dos vizinhos — o mesmo Documento com dois
		// desenhos.
		corte := fmt.Sprintf("rotulo-%d-%d", frame, *rotulos)
		*rotulos++
		fmt.Fprintf(b, "<clipPath id=\"%s\"><rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\"/></clipPath>\n",
			corte, num(e.X), num(e.Y), num(e.L), num(e.A))
		fmt.Fprintf(b, "<text class=\"%s rotulo\" x=\"%s\" y=\"%s\" font-size=\"%s\" text-anchor=\"%s\" fill=\"%s\" clip-path=\"url(#%s)\"%s>%s</text>\n",
			classe, num(x), num(e.Y+e.A/2), num(tamanho), ancora, cor(e.Tom), corte, atributos, escapa(e.Rotulo))
	default:
		geometria := fmt.Sprintf("x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\"",
			num(e.X), num(e.Y), num(e.L), num(e.A))
		if e.Arredondado {
			geometria += fmt.Sprintf(" rx=\"%s\"", num(render.Raio(e.L, e.A)))
		}
		abre("rect", geometria)
	}
}

// escreveLigacao desenha uma Ligação: uma curva que sai da borda do Elemento
// gatilho e chega na borda do Frame de destino, com ponta de seta.
func escreveLigacao(b *bufio.Writer, d *scene.Documento, posicoes []posicao, lg ligacao) {
	origem := posicoes[lg.de]
	alvo := posicoes[lg.para]
	frameAlvo := d.Frames[lg.para]

	// Coordenadas do gatilho na Prancheta.
	gx, gy := origem.X+lg.x, origem.Y+lg.y
	centroGatilhoY := gy + lg.a/2
	alvoX, alvoL := alvo.X, float64(frameAlvo.L)
	alvoY, alvoA := alvo.Y, float64(frameAlvo.A)

	var x1, y1, x2, y2, c1x, c2x float64
	if lg.de == lg.para {
		// A auto-Ligação não tem para onde ir: vira um laço à direita do
		// próprio Frame.
		x1, y1 = gx+lg.l, centroGatilhoY
		x2, y2 = alvoX+alvoL, centroGatilhoY+lg.a+40
		laco := intervaloH / 2
		fmt.Fprintf(b, "<path class=\"ligacao\" d=\"M %s,%s C %s,%s %s,%s %s,%s\" marker-end=\"url(#ponta)\" data-de=\"%d\" data-para=\"%d\" data-caminho=\"%s\"/>\n",
			num(x1), num(y1), num(x1+laco), num(y1), num(x2+laco), num(y2), num(x2), num(y2),
			lg.de, lg.para, escapa(lg.caminho))
		return
	}

	if alvoX+alvoL/2 >= gx+lg.l/2 {
		// O alvo está à direita: sai pela direita do gatilho, entra pela
		// esquerda do Frame.
		x1, y1 = gx+lg.l, centroGatilhoY
		x2, y2 = alvoX, alvoY+alvoA/2
	} else {
		x1, y1 = gx, centroGatilhoY
		x2, y2 = alvoX+alvoL, alvoY+alvoA/2
	}
	// Os pontos de controle são horizontais: a curva sai e chega perpendicular
	// à borda, que é o que faz duas Ligações vizinhas se distinguirem.
	folga := math.Max(math.Abs(x2-x1)*0.45, 60)
	if x2 >= x1 {
		c1x, c2x = x1+folga, x2-folga
	} else {
		c1x, c2x = x1-folga, x2+folga
	}

	fmt.Fprintf(b, "<path class=\"ligacao\" d=\"M %s,%s C %s,%s %s,%s %s,%s\" marker-end=\"url(#ponta)\" data-de=\"%d\" data-para=\"%d\" data-caminho=\"%s\"/>\n",
		num(x1), num(y1), num(c1x), num(y1), num(c2x), num(y2), num(x2), num(y2),
		lg.de, lg.para, escapa(lg.caminho))
}

// cor devolve o cinza de um Tom no formato hexadecimal do CSS. A escala é a
// mesma do raster: a Prancheta não inventa cor nenhuma.
func cor(t scene.Tom) string {
	c := t.Cinza()
	return fmt.Sprintf("#%02x%02x%02x", c, c, c)
}

// num formata uma coordenada com no máximo duas casas. O arredondamento existe
// para que a saída não carregue ruído de ponto flutuante e seja byte a byte
// igual entre execuções.
func num(v float64) string {
	if !finito(v) {
		return "0"
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

func finito(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// escapa neutraliza o texto vindo do YAML antes de ele entrar no documento.
func escapa(s string) string {
	return html.EscapeString(s)
}
