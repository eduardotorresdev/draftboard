package notes

import (
	"fmt"
	"math"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Estes testes olham os retângulos do Plano por dentro, e não a imagem, porque
// é sobre eles que a anti-colisão promete: "dois balões não se cruzam" é uma
// afirmação de geometria, e pedi-la à imagem só provaria que dois blocos de
// tinta clara ficaram separados — o que também acontece com balões sobrepostos,
// desde que as linhas de texto caiam em alturas diferentes. As garantias
// visuais continuam sendo testadas pela imagem, em notes_test.go.

// colunaAnotada monta um Frame com n Elementos empilhados numa coluna central,
// com as âncoras a passo px umas das outras. Passo menor que a altura de um
// balão é o que força a anti-colisão a trabalhar.
func colunaAnotada(n int, passo float64) scene.Frame {
	f := scene.Frame{Nome: "home", L: 400, A: 600}
	c := scene.Camada{Nome: "conteudo"}
	for i := 0; i < n; i++ {
		c.Elementos = append(c.Elementos, scene.Elemento{
			X: 160, Y: 25 + float64(i)*passo, L: 80, A: 30, Tom: 300,
			Nota: fmt.Sprintf("Nota %d", i+1),
		})
	}
	f.Camadas = []scene.Camada{c}
	return f
}

func TestBaloesNaoSeCruzam(t *testing.T) {
	// Um balão de uma linha tem cerca de 30 px de altura; com as âncoras a
	// 15 px umas das outras, a posição desejada de todos eles se sobrepõe e
	// só o desvio da anti-colisão separa os retângulos.
	for _, quantas := range []int{2, 3, 10} {
		t.Run(fmt.Sprintf("%d notas", quantas), func(t *testing.T) {
			p := Planeja(colunaAnotada(quantas, 15), 1)
			if len(p.notas) != quantas {
				t.Fatalf("%d Notas colhidas, quer %d", len(p.notas), quantas)
			}
			for i := range p.notas {
				for j := i + 1; j < len(p.notas); j++ {
					a, b := p.notas[i].balao(), p.notas[j].balao()
					if a.cruza(b) {
						t.Errorf("balões %d e %d se cruzam: %+v e %+v", i, j, a, b)
					}
				}
			}
		})
	}
}

func TestBalaoNaoSaiDoFrame(t *testing.T) {
	f := colunaAnotada(10, 15)
	p := Planeja(f, 1)
	fl, fa := float64(f.L), float64(f.A)
	for i, n := range p.notas {
		b := n.balao()
		if b.x0 < 0 || b.y0 < 0 || b.x1 > fl || b.y1 > fa {
			t.Errorf("balão %d escapou do Frame de %vx%v: %+v", i, fl, fa, b)
		}
	}
}

// TestOrdemDeDeclaracaoNaoMudaOPlano fecha o empate que a ordenação por altura,
// borda direita e texto não resolve.
//
// Os dois Elementos terminam na mesma borda direita, na mesma altura, e levam o
// mesmo texto: os três primeiros critérios empatam. Só a borda ESQUERDA os
// separa — e ela decide o layout, porque o Elemento largo não tem espaço à
// esquerda para o seu balão e o estreito tem. Quem for atendido primeiro fica
// com a direita; o outro escolhe entre descer e ir para a esquerda. Sem o
// desempate, a ordem de declaração passaria a escolher, e a imagem mudaria sem
// a geometria ter mudado.
func TestOrdemDeDeclaracaoNaoMudaOPlano(t *testing.T) {
	largo := scene.Elemento{X: 0, Y: 0, L: 300, A: 20, Tom: 300, Nota: "A"}
	estreito := scene.Elemento{X: 200, Y: 0, L: 100, A: 20, Tom: 300, Nota: "A"}

	monta := func(es ...scene.Elemento) scene.Frame {
		return scene.Frame{Nome: "home", L: 400, A: 200, Camadas: []scene.Camada{
			{Nome: "conteudo", Elementos: es},
		}}
	}

	direta := Planeja(monta(largo, estreito), 1)
	invertida := Planeja(monta(estreito, largo), 1)

	if len(direta.notas) != 2 || len(invertida.notas) != 2 {
		t.Fatalf("Notas colhidas: %d e %d, quer 2 e 2", len(direta.notas), len(invertida.notas))
	}
	for i := range direta.notas {
		a, b := direta.notas[i], invertida.notas[i]
		if a.balao() != b.balao() || a.ancoraX != b.ancoraX {
			t.Errorf("Nota %d mudou de lugar com a declaração invertida: %+v (âncora %v) contra %+v (âncora %v)",
				i, a.balao(), a.ancoraX, b.balao(), b.ancoraX)
		}
	}

	// E o desempate escolhe pela geometria, não pela sorte: a Nota do
	// Elemento que começa mais à esquerda é sempre a primeira atendida.
	if direta.notas[0].esquerdaDoElemento != largo.X {
		t.Errorf("a primeira Nota veio do Elemento em X=%v, quer o de X=%v", direta.notas[0].esquerdaDoElemento, largo.X)
	}
}

// TestLimiteDaNotaNaoEntraNoLayout: o teto é declarado para o diagnóstico, e
// medir por ele aqui cortaria a Nota longa sem dizer nada a quem a escreveu.
func TestLimiteDaNotaNaoEntraNoLayout(t *testing.T) {
	if LimiteDaNota != 200 {
		t.Errorf("LimiteDaNota = %d, quer 200", LimiteDaNota)
	}

	// Duas Notas de uma palavra só, repetida: uma exatamente no teto, outra
	// bem acima dele. Se o layout truncasse no teto, as duas dariam o mesmo
	// balão — é justamente esse corte silencioso que não pode existir aqui.
	curta := repetePalavras(LimiteDaNota)
	longa := repetePalavras(LimiteDaNota * 3)

	altura := func(texto string) float64 {
		p := Planeja(umaNota(texto), 1)
		if len(p.notas) != 1 {
			t.Fatalf("%d Notas colhidas, quer 1", len(p.notas))
		}
		b := p.notas[0].balao()
		return b.y1 - b.y0
	}
	if a, b := altura(curta), altura(longa); b <= a {
		t.Errorf("Nota de %d runas deu balão de %v px e a de %d runas deu %v px: o layout parou no teto",
			len([]rune(longa)), b, len([]rune(curta)), a)
	}
}

// TestPlanoEMedidoNaEscalaEmQueSeraPintado: o texto é medido com a fonte no
// tamanho de dispositivo, e o hinting faz a largura de uma linha não ser
// exatamente proporcional ao fator. Se a régua ignorasse a escala, o bloco de
// texto teria exatamente a mesma largura em todas elas — e o planejado
// deixaria de ser o que realmente cabe quando o texto é pintado.
func TestPlanoEMedidoNaEscalaEmQueSeraPintado(t *testing.T) {
	f := umaNota(repetePalavras(40))
	base := Planeja(f, 1).notas[0].l

	for _, escala := range []float64{2, 3, 5, 8} {
		if Planeja(f, escala).notas[0].l != base {
			return
		}
	}
	t.Errorf("a largura do texto não mudou em nenhuma escala testada: sempre %v — a régua está ignorando o fator", base)
}

func TestEscalaNaoUtilizavelNaoDerrubaOPlano(t *testing.T) {
	// A CLI recusa escala não-positiva antes de chegar aqui, mas a
	// biblioteca não pode entrar em pânico nem devolver medida infinita se
	// alguém a chamar direto: a escala degenerada cai para 1.
	f := umaNota(repetePalavras(40))
	querido := Planeja(f, 1).notas[0].balao()

	for _, escala := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -3} {
		b := Planeja(f, escala).notas[0].balao()
		for _, v := range []float64{b.x0, b.y0, b.x1, b.y1} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("escala %v produziu balão não finito: %+v", escala, b)
			}
		}
		if b != querido {
			t.Errorf("escala %v deu %+v, quer o mesmo da escala 1: %+v", escala, b, querido)
		}
	}
}

func umaNota(texto string) scene.Frame {
	return scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 20, Y: 30, L: 200, A: 60, Tom: 300, Nota: texto}},
	}}}
}

// repetePalavras devolve um texto de runas runas, quebrável em qualquer ponto.
func repetePalavras(runas int) string {
	const palavra = "nota "
	s := ""
	for len([]rune(s)) < runas {
		s += palavra
	}
	return string([]rune(s)[:runas])
}
