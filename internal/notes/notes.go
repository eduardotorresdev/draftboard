// Package notes calcula e desenha o plano de anotação de um Frame: as Notas
// aninhadas nos Elementos, posicionadas automaticamente e ligadas por uma linha
// de chamada ao Elemento que explicam.
//
// A Nota não faz parte do desenho: não participa da Elevação e não aparece no
// export por Camada. São dois planos separados, e este é o plano de anotação.
// A âncora é implícita — é o próprio Elemento que carrega a Nota, sem
// identificador.
//
// A tela tem sempre as dimensões do Frame: o balão é preso dentro dele e nada
// cresce ao redor. Quem não quer Notas não chama Planeja.
//
// Todas as medidas deste pacote são em pixels do ESPAÇO DO FRAME, antes da
// escala. O Canvas multiplica tudo pelo fator, então o layout inteiro — corpo
// da fonte, respiros e espessura da linha de chamada — escala junto, e a
// aparência da anotação não muda com a escala.
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

// LimiteDaNota é o teto de tamanho de uma Nota, em RUNAS — o texto é português
// e um acento não pode custar dois caracteres do orçamento. Conte com
// utf8.RuneCountInString, nunca com len.
//
// São cerca de quatro linhas na largura que o balão já tem, e o teto é
// constante no binário pela mesma razão que o Tom e o corpo da fonte são: quem
// escreve precisa saber o limite enquanto escreve, e um limite derivado do
// espaço livre passaria num Frame e falharia noutro.
//
// O layout NÃO o consulta: mede sempre o texto inteiro, não trunca e não avisa.
// Quem transforma Nota longa em Erro é o diagnóstico, no caminho de validação,
// onde há como dizer ao autor o que cortar.
const LimiteDaNota = 200

// Decisões de layout. Todas em px do espaço do Frame; ver o comentário do
// pacote sobre escala.
const (
	// corpoDaFonte é o tamanho do texto da Nota. 12 px é o menor corpo que
	// continua legível na escala 1 com a Go Regular, e a Nota é anotação:
	// precisa ser lida sem competir com o desenho.
	corpoDaFonte = 12.0

	// larguraMaximaDoTexto é a largura máxima FIXA de uma linha de texto. É
	// o que faz o texto longo quebrar em várias linhas em vez de esticar o
	// balão até atravessar o Frame. ~180 px dão de 30 a 40 caracteres por
	// linha, a faixa confortável de leitura.
	larguraMaximaDoTexto = 180.0

	// respiro é a folga entre o texto e qualquer borda: a do Frame, a da
	// tela, ou a do balão.
	respiro = 8.0

	// espacoEntreBaloes é o vão vertical que a anti-colisão abre entre dois
	// balões. Encostados eles se leriam como um bloco escuro só; com o vão,
	// a fronteira entre duas Notas é visível sem precisar ler o texto.
	espacoEntreBaloes = 10.0

	// espessuraDaChamada é a espessura da linha de chamada. Fina de
	// propósito: liga sem pesar no desenho.
	espessuraDaChamada = 1.0

	// folgaDaChamada é o vão entre a ponta da linha de chamada e o texto,
	// para que a linha não encoste nos glifos.
	folgaDaChamada = 4.0

	// folgaDoBalao afasta o balão da borda do Elemento anotado. É o que
	// garante que o balão fique AO LADO da bounding box do Elemento e nunca
	// por cima dela.
	folgaDoBalao = 10.0
)

// Tons do plano de anotação.
const (
	// tomDoTexto é o extremo claro da escala. O balão é escuro, então o
	// texto contrasta com o fundo dele.
	tomDoTexto = scene.TomFrame

	// tomDoBalao é o extremo escuro reservado: nenhum Elemento pode ter esse
	// Tom, então o balão se distingue do desenho só pelo cinza, sem borda
	// nem contorno.
	tomDoBalao = scene.TomChrome

	// tomDaChamada é o meio da escala. A linha de chamada atravessa as duas
	// regiões — o Frame, claro, e o balão, escuro — então nenhum dos dois
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

// balao é o retângulo pintado ao redor do bloco de texto: o texto mais o
// respiro nos quatro lados. É ele, e não o bloco de texto, que a anti-colisão
// mantém separado — dois balões encostados já se leem como um só.
func (n nota) balao() caixa {
	return caixa{n.x - respiro, n.y - respiro, n.x + n.l + respiro, n.y + n.a + respiro}
}

// caixa é um retângulo no espaço do Frame, meio-aberto: a borda que fecha não
// pertence a ele, de modo que dois balões encostados não se cruzam.
type caixa struct{ x0, y0, x1, y1 float64 }

func (c caixa) cruza(o caixa) bool {
	return c.x0 < o.x1 && o.x0 < c.x1 && c.y0 < o.y1 && o.y0 < c.y1
}

// Plano é o layout das Notas de um Frame, calculado sem desenhar.
type Plano struct {
	alturaLinha float64
	notas       []nota
}

// Planeja resolve a posição de todas as Notas do Frame. escala é o fator da
// CLI: entra no cálculo porque o texto é medido na escala em que será pintado,
// de modo que o que foi planejado é exatamente o que cabe.
//
// Quem não quer Notas não chama Planeja: um *Plano nulo é o zero natural do
// tipo e todos os métodos o aceitam.
func Planeja(f scene.Frame, escala float64) *Plano {
	p := &Plano{}
	p.notas = colhe(f)
	if len(p.notas) == 0 {
		return p
	}
	if math.IsNaN(escala) || math.IsInf(escala, 0) || escala <= 0 {
		escala = 1
	}

	// Régua: um Canvas mínimo, na mesma escala do desenho final, usado só
	// para medir e quebrar texto. Não é desenhado nem codificado.
	regua := render.NewCanvas(1, 1, 0, 0, 0, 0, escala)
	_, p.alturaLinha = regua.MedeTexto("Mg", corpoDaFonte)

	p.dispoe(regua, f)
	return p
}

// Margens devolve as margens que a tela precisa ter ao redor do Frame, em px do
// espaço do Frame. São sempre 0,0,0,0: o balão é preso dentro do Frame e a tela
// tem as dimensões dele.
//
// O método sobrevive porque é aqui que o plano de anotação responde pelo
// tamanho da tela — quem desenha pergunta a ele em vez de assumir zero, e o dia
// em que a anotação voltar a precisar de espaço próprio há um lugar só para
// mudar.
func (p *Plano) Margens() (t, d, b, e float64) {
	return 0, 0, 0, 0
}

// Desenha pinta Notas e linhas de chamada sobre um Canvas já criado com essas
// margens. Um Plano nulo — Notas desligadas na linha de comando — não desenha
// nada, porque não há laço nenhum a percorrer.
//
// As primitivas do Canvas usam coordenadas da tela inteira, margens inclusas; o
// layout é calculado no espaço do Frame, então tudo é deslocado pela origem do
// Frame dentro da tela.
func (p *Plano) Desenha(c *render.Canvas) {
	if p == nil || c == nil {
		return
	}
	ox, oy := c.OrigemDoFrame()
	for _, n := range p.notas {
		x, y := ox+n.x, oy+n.y
		// O balão é o fundo próprio da Nota: sem ele o texto claro cairia
		// sobre o Frame claro e sumiria.
		b := n.balao()
		c.Retangulo(ox+b.x0, oy+b.y0, b.x1-b.x0, b.y1-b.y0, tomDoBalao)
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
// resultado. E a ordem importa mais do que importava: é ela que decide, na
// anti-colisão, quem chega antes e portanto quem fica com o lugar que quer.
//
// Os quatro critérios fecham ordem total sobre a geometria e o texto. Sem o
// desempate pela borda ESQUERDA, dois Elementos na mesma altura e com a mesma
// borda direita — um largo, outro estreito e encostado nele — empatam nos três
// primeiros; com o mesmo texto, empatam em todos. sort.SliceStable fecha esse
// último caso na ordem de declaração, que só chega a valer quando geometria e
// texto são idênticos e o resultado é indistinguível de qualquer jeito.
func colhe(f scene.Frame) []nota {
	var notas []nota
	for _, camada := range f.Camadas {
		for _, e := range camada.Elementos {
			// Nota vazia é ausência de Nota: não ocupa espaço nem pede
			// balão.
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
	sort.SliceStable(notas, func(i, j int) bool {
		a, b := notas[i], notas[j]
		if a.meioDoElemento != b.meioDoElemento {
			return a.meioDoElemento < b.meioDoElemento
		}
		if a.direitaDoElemento != b.direitaDoElemento {
			return a.direitaDoElemento < b.direitaDoElemento
		}
		if a.esquerdaDoElemento != b.esquerdaDoElemento {
			return a.esquerdaDoElemento < b.esquerdaDoElemento
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
