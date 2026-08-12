// Package notes calcula e desenha o plano de anotação de um Frame: as Notas
// aninhadas nos Elementos, posicionadas automaticamente e ligadas por uma linha
// de chamada ao Elemento que explicam.
//
// A Nota não faz parte do desenho: não participa da Elevação e não aparece no
// export por Camada. São dois planos separados, e este é o plano de anotação.
// A âncora é implícita — é o próprio Elemento que carrega a Nota, sem
// identificador.
//
// Todas as medidas deste pacote são em pixels do ESPAÇO DO FRAME, antes da
// escala. O Canvas multiplica tudo pelo fator, então o layout inteiro — corpo
// da fonte, respiros, faixa de Chrome e espessura da linha de chamada — escala
// junto, e a aparência da anotação não muda com a escala.
//
// Este pacote importa internal/render. O caminho inverso nunca existe.
package notes

import (
	"math"
	"sort"
	"strings"

	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Modo é como as Notas são posicionadas na renderização. É opção da linha de
// comando, nunca do Documento.
type Modo int

const (
	// Margem posiciona as Notas no Chrome ao redor do Frame. É o padrão da
	// CLI.
	Margem Modo = iota
	// Flutuante posiciona as Notas sobre o desenho, perto da âncora.
	Flutuante
	// Desligado remove as Notas inteiras da renderização.
	Desligado
)

// Decisões de layout. Todas em px do espaço do Frame; ver o comentário do
// pacote sobre escala.
const (
	// corpoDaFonte é o tamanho do texto da Nota. 12 px é o menor corpo que
	// continua legível na escala 1 com a Go Regular, e a Nota é anotação:
	// precisa ser lida sem competir com o desenho.
	corpoDaFonte = 12.0

	// larguraMaximaDoTexto é a largura máxima FIXA de uma linha de texto. É
	// o que faz o texto longo quebrar em várias linhas em vez de esticar o
	// Chrome sem fim. ~180 px dão de 30 a 40 caracteres por linha, a faixa
	// confortável de leitura.
	larguraMaximaDoTexto = 180.0

	// larguraMinimaDaFaixa é a largura da faixa de Chrome ANTES de crescer.
	// Uma Nota curta não deixa o Chrome virar um filete grudado no Frame; a
	// faixa cresce a partir daqui, o quanto for preciso pra caber as Notas.
	larguraMinimaDaFaixa = 48.0

	// respiro é a folga entre o texto e qualquer borda: a do Frame, a da
	// tela, ou a do balão no modo Flutuante.
	respiro = 8.0

	// espacoEntreNotas separa duas Notas empilhadas. É maior que o respiro
	// para que a fronteira entre duas Notas seja mais forte que a fronteira
	// entre duas linhas da mesma Nota.
	espacoEntreNotas = 10.0

	// espessuraDaChamada é a espessura da linha de chamada. Fina de
	// propósito: liga sem pesar no desenho.
	espessuraDaChamada = 1.0

	// folgaDaChamada é o vão entre a ponta da linha de chamada e o texto,
	// para que a linha não encoste nos glifos.
	folgaDaChamada = 4.0

	// folgaFlutuante afasta o balão da borda do Elemento anotado no modo
	// Flutuante. É o que garante que o balão fique AO LADO da bounding box
	// do Elemento e nunca por cima dela.
	folgaFlutuante = 10.0
)

// Tons do plano de anotação.
const (
	// tomDoTexto é o extremo claro da escala. O Chrome é TomChrome (quase
	// preto) e o balão flutuante também, então o texto contrasta com o fundo
	// em qualquer um dos modos.
	tomDoTexto = scene.TomFrame

	// tomDoBalao é o mesmo Tom reservado do Chrome: no modo Flutuante o
	// balão é um pedaço de Chrome pousado sobre o desenho, e o Tom reservado
	// deixa claro que aquilo é anotação, não Elemento.
	tomDoBalao = scene.TomChrome

	// tomDaChamada é o meio da escala. A linha de chamada atravessa as duas
	// regiões — o Frame, claro, e o Chrome, escuro — então nenhum dos dois
	// extremos serve: só um Tom central se destaca dos dois.
	tomDaChamada = scene.Tom(500)
)

// nota é uma Nota já colhida do Elemento que a carrega, com a geometria da
// âncora e, depois do layout, a posição do seu bloco de texto.
type nota struct {
	texto string

	// Bordas da bounding box do Elemento anotado, no espaço do Frame. A
	// âncora é implícita: é a borda do Elemento voltada para a Nota, na
	// altura do meio do Elemento.
	esquerdaDoElemento, direitaDoElemento float64
	meioDoElemento                        float64

	// linhas é o texto já quebrado.
	linhas []string
	// x, y é o canto superior esquerdo do bloco de texto e l, a as suas
	// dimensões, no espaço do Frame.
	x, y, l, a float64
	// ancoraX é o ponto da borda do Elemento de onde sai a linha de chamada,
	// e chamadaX onde ela termina, junto do bloco de texto.
	ancoraX, chamadaX float64
}

// Plano é o layout das Notas de um Frame, calculado sem desenhar.
type Plano struct {
	modo        Modo
	alturaLinha float64
	notas       []nota

	margemT, margemD, margemB, margemE float64
}

// Planeja resolve a posição de todas as Notas do Frame. escala é o fator da
// CLI: entra no cálculo porque o texto é medido na escala em que será pintado,
// de modo que o que foi planejado é exatamente o que cabe.
func Planeja(f scene.Frame, m Modo, escala float64) *Plano {
	p := &Plano{modo: m}
	if m == Desligado {
		return p
	}
	p.notas = colhe(f)
	if len(p.notas) == 0 {
		// Frame sem nenhuma Nota não pede Chrome nenhum.
		return p
	}
	if math.IsNaN(escala) || math.IsInf(escala, 0) || escala <= 0 {
		escala = 1
	}

	// Régua: um Canvas mínimo, na mesma escala do desenho final, usado só
	// para medir e quebrar texto. Não é desenhado nem codificado.
	regua := render.NewCanvas(1, 1, 0, 0, 0, 0, escala)
	_, p.alturaLinha = regua.MedeTexto("Mg", corpoDaFonte)

	if m == Flutuante {
		p.disporFlutuante(regua, f)
		return p
	}
	p.disporNaMargem(regua, f)
	return p
}

// Margens devolve o Chrome necessário em px do espaço do Frame. No modo
// Flutuante e Desligado devolve 0,0,0,0, porque a tela mantém as dimensões do
// Frame.
func (p *Plano) Margens() (t, d, b, e float64) {
	if p == nil {
		return 0, 0, 0, 0
	}
	return p.margemT, p.margemD, p.margemB, p.margemE
}

// Desenha pinta Notas e linhas de chamada sobre um Canvas já criado com essas
// margens. No modo Desligado não faz nada, porque Planeja não colhe Nota
// nenhuma nesse modo e o laço abaixo não tem o que percorrer.
//
// As primitivas do Canvas usam coordenadas da tela inteira, Chrome incluso; o
// layout é calculado no espaço do Frame, então tudo é deslocado pela origem do
// Frame dentro da tela.
func (p *Plano) Desenha(c *render.Canvas) {
	if p == nil || c == nil {
		return
	}
	ox, oy := c.OrigemDoFrame()
	for _, n := range p.notas {
		x, y := ox+n.x, oy+n.y
		if p.modo == Flutuante {
			// Sobre o desenho a Nota precisa do seu próprio fundo; no
			// Chrome o fundo já é o Tom reservado.
			c.Retangulo(x-respiro, y-respiro, n.l+2*respiro, n.a+2*respiro, tomDoBalao)
		}
		c.Linha(ox+n.ancoraX, oy+n.meioDoElemento, ox+n.chamadaX, y+n.a/2, espessuraDaChamada, tomDaChamada)
		for i, linha := range n.linhas {
			c.Texto(x, y+float64(i)*p.alturaLinha, linha, corpoDaFonte, tomDoTexto)
		}
	}
}

// colhe junta as Notas de todas as Camadas do Frame e as ordena pela altura da
// âncora.
//
// A ordem é função apenas da geometria e do texto — nunca da posição do
// Elemento na lista. É isso que torna o layout estável entre edições: mexer na
// ordem de declaração sem mexer na geometria produz exatamente o mesmo
// resultado. O desempate por X e depois pelo texto fecha a ordem total, para
// que duas âncoras na mesma altura também não dependam da declaração.
func colhe(f scene.Frame) []nota {
	var notas []nota
	for _, camada := range f.Camadas {
		for _, e := range camada.Elementos {
			// Nota vazia é ausência de Nota: não ocupa espaço nem pede
			// Chrome.
			if strings.TrimSpace(e.Nota) == "" {
				continue
			}
			notas = append(notas, nota{
				texto:              e.Nota,
				esquerdaDoElemento: e.X,
				direitaDoElemento:  e.X + e.L,
				meioDoElemento:     e.Y + e.A/2,
			})
		}
	}
	sort.Slice(notas, func(i, j int) bool {
		a, b := notas[i], notas[j]
		if a.meioDoElemento != b.meioDoElemento {
			return a.meioDoElemento < b.meioDoElemento
		}
		if a.direitaDoElemento != b.direitaDoElemento {
			return a.direitaDoElemento < b.direitaDoElemento
		}
		return a.texto < b.texto
	})
	return notas
}

// quebra preenche as linhas e as dimensões do bloco de texto de uma Nota.
func (p *Plano) quebra(regua *render.Canvas, n *nota, larguraMax float64) {
	n.linhas = regua.QuebraTexto(n.texto, corpoDaFonte, larguraMax)
	n.l = 0
	for _, linha := range n.linhas {
		if l, _ := regua.MedeTexto(linha, corpoDaFonte); l > n.l {
			n.l = l
		}
	}
	// A altura de MedeTexto é a caixa de linha inteira, subida mais descida,
	// então empilhar as linhas por ela basta para que nunca se toquem.
	n.a = float64(len(n.linhas)) * p.alturaLinha
}
