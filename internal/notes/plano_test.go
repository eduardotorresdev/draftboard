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

// barraInferior monta um Frame de celular com n Elementos lado a lado colados
// na base — uma barra de abas típica. Todas as âncoras ficam na mesma altura, a
// poucos pixels do fim da tela: não há para onde descer, e a tela vazia está
// toda acima.
func barraInferior(n int) scene.Frame {
	f := scene.Frame{Nome: "home", L: 375, A: 812}
	c := scene.Camada{Nome: "abas"}
	for i := 0; i < n; i++ {
		c.Elementos = append(c.Elementos, scene.Elemento{
			X: 20 + float64(i)*58, Y: 740, L: 48, A: 48, Tom: 300,
			Nota: fmt.Sprintf("Aba %d", i+1),
		})
	}
	f.Camadas = []scene.Camada{c}
	return f
}

// colunaEstreita monta um Frame mais estreito que o balão pede — os dois lados
// do Elemento são descartados pela largura antes de qualquer descida, e o
// posicionamento cai inteiro no ramo de último recurso.
func colunaEstreita(n int) scene.Frame {
	f := scene.Frame{Nome: "estreito", L: 60, A: 400}
	c := scene.Camada{Nome: "conteudo"}
	for i := 0; i < n; i++ {
		c.Elementos = append(c.Elementos, scene.Elemento{
			X: 8, Y: 8 + float64(i)*32, L: 44, A: 24, Tom: 300,
			Nota: fmt.Sprintf("Nota %d", i+1),
		})
	}
	f.Camadas = []scene.Camada{c}
	return f
}

// cruzamentos devolve os pares de balões que se sobrepõem.
func cruzamentos(p *Plano) [][2]int {
	var pares [][2]int
	for i := range p.notas {
		for j := i + 1; j < len(p.notas); j++ {
			if p.notas[i].balao().cruza(p.notas[j].balao()) {
				pares = append(pares, [2]int{i, j})
			}
		}
	}
	return pares
}

func TestBaloesNaoSeCruzam(t *testing.T) {
	// Um balão de uma linha tem cerca de 30 px de altura; com as âncoras a
	// 15 px umas das outras, a posição desejada de todos eles se sobrepõe e
	// só o desvio da anti-colisão separa os retângulos.
	casos := []struct {
		nome    string
		f       scene.Frame
		quantas int
	}{
		{"2 notas", colunaAnotada(2, 15), 2},
		{"3 notas", colunaAnotada(3, 15), 3},
		{"10 notas", colunaAnotada(10, 15), 10},
		// Barra inferior: as seis âncoras nascem na mesma altura, a 60 px
		// do fim da tela. Descer é impossível, e antes de subir ser
		// tentado o layout aceitava a sobreposição com 700 px de tela
		// vazia logo acima.
		{"barra inferior", barraInferior(6), 6},
		// Frame estreito: nenhum dos dois lados passa no portão de
		// largura, então a anti-colisão inteira era pulada e os quatro
		// balões saíam empilhados no mesmo lugar. Somados eles têm 300 px
		// de altura num Frame de 400: há espaço para todos.
		{"frame estreito", colunaEstreita(4), 4},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			p := Planeja(caso.f, 1)
			if len(p.notas) != caso.quantas {
				t.Fatalf("%d Notas colhidas, quer %d", len(p.notas), caso.quantas)
			}
			for _, par := range cruzamentos(p) {
				i, j := par[0], par[1]
				t.Errorf("balões %d e %d se cruzam: %+v e %+v",
					i, j, p.notas[i].balao(), p.notas[j].balao())
			}
		})
	}
}

// TestBalaoNaoSaiDoFrame cobre também o ramo de último recurso, que é onde as
// duas garantias caem: Frame estreito, coluna saturada — mais Notas do que a
// tela comporta — e balão mais alto que o Frame inteiro.
//
// A promessa tem dois níveis, porque abaixo de 2*respiro + alturaDeLinha
// nenhuma posição cabe: o balão que cabe fica dentro da tela; o que não cabe
// encosta no canto de origem e transborda só o que é geometricamente
// inevitável.
func TestBalaoNaoSaiDoFrame(t *testing.T) {
	casos := []struct {
		nome string
		f    scene.Frame
	}{
		{"coluna folgada", colunaAnotada(10, 15)},
		{"frame estreito", colunaEstreita(4)},
		{"coluna saturada", saturada(20)},
		{"barra inferior", barraInferior(6)},
		{"balão mais alto que o Frame", umNotaoNumFrameBaixo()},
		{"frame minúsculo", frameMinusculo()},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			fl, fa := float64(caso.f.L), float64(caso.f.A)
			p := Planeja(caso.f, 1)
			if len(p.notas) == 0 {
				t.Fatal("nenhuma Nota colhida")
			}
			for i, n := range p.notas {
				b := n.balao()
				// Preso à tela: o balão nunca começa antes da
				// origem, caiba ele ou não.
				if b.x0 < 0 || b.y0 < 0 {
					t.Errorf("balão %d começa fora da tela: %+v", i, b)
				}
				if b.x1-b.x0 <= fl && b.x1 > fl {
					t.Errorf("balão %d de %v px de largura escapou do Frame de %v px: %+v", i, b.x1-b.x0, fl, b)
				}
				if b.y1-b.y0 <= fa && b.y1 > fa {
					t.Errorf("balão %d de %v px de altura escapou do Frame de %v px: %+v", i, b.y1-b.y0, fa, b)
				}
			}
		})
	}
}

// TestBalaoGuardaRespiroContraABordaDaTela: o balão é texto mais respiro, e o
// limite é dele, não do bloco de texto. Presar o texto deixava o balão colado
// na moldura, com folga zero — e nos quatro lados, porque a saturação acontece
// nos dois extremos de cada eixo.
func TestBalaoGuardaRespiroContraABordaDaTela(t *testing.T) {
	// Um Elemento em cada canto do Frame, um por vez: é a única forma de
	// saturar cada um dos quatro limites.
	cantos := []struct {
		nome string
		x, y float64
	}{
		{"canto superior esquerdo", 0, 0},
		{"canto superior direito", 340, 0},
		{"canto inferior esquerdo", 0, 280},
		{"canto inferior direito", 340, 280},
	}
	for _, canto := range cantos {
		t.Run(canto.nome, func(t *testing.T) {
			f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
				Nome: "conteudo",
				Elementos: []scene.Elemento{{
					X: canto.x, Y: canto.y, L: 60, A: 20, Tom: 300,
					Nota: repetePalavras(60),
				}},
			}}}
			b := Planeja(f, 1).notas[0].balao()
			if b.x0 < respiro || b.y0 < respiro || b.x1 > 400-respiro || b.y1 > 300-respiro {
				t.Errorf("balão %+v encostou na borda da tela de 400x300: quer %v px de folga nos quatro lados", b, respiro)
			}
		})
	}
}

// TestCoordenadaAbsurdaNaoChegaAoCanvas: o diagnóstico deixa passar como aviso
// o Elemento que cai fora do Frame, e um Elemento assim pode declarar dimensão
// absurda. Se a âncora e a linha de chamada saírem daí sem limite, o
// rasterizador varre a distância célula a célula e o processo nunca volta — o
// que se observa na CLI como um `render` que não termina.
func TestCoordenadaAbsurdaNaoChegaAoCanvas(t *testing.T) {
	casos := []struct {
		nome       string
		x, y, l, a float64
	}{
		{"largura absurda", 5, 5, 1e300, 5},
		{"altura absurda", 5, 5, 5, 1e300},
		{"origem absurda à direita", 1e300, 5, 5, 5},
		{"origem absurda à esquerda", -1e300, 5, 5, 5},
		{"origem absurda embaixo", 5, 1e300, 5, 5},
	}
	const fl, fa = 400.0, 300.0
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			f := scene.Frame{Nome: "home", L: int(fl), A: int(fa), Camadas: []scene.Camada{{
				Nome: "conteudo",
				Elementos: []scene.Elemento{{
					X: caso.x, Y: caso.y, L: caso.l, A: caso.a, Tom: 300, Nota: "Cartão",
				}},
			}}}
			n := Planeja(f, 1).notas[0]
			b := n.balao()
			medidas := map[string]float64{
				"x": n.x, "y": n.y, "ancoraX": n.ancoraX, "chamadaX": n.chamadaX,
				"meioDoElemento": n.meioDoElemento,
				"balao.x0":       b.x0, "balao.y0": b.y0, "balao.x1": b.x1, "balao.y1": b.y1,
			}
			for nome, v := range medidas {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Fatalf("%s não é finito: %v", nome, v)
				}
				// Uma folga de um Frame inteiro para cada lado é
				// generosa de sobra: o que se recusa aqui é a
				// ordem de grandeza que mata o rasterizador.
				if v < -fl-fa || v > 2*(fl+fa) {
					t.Errorf("%s = %v, fora do alcance da tela de %vx%v", nome, v, fl, fa)
				}
			}
		})
	}
}

// saturada é uma coluna com Notas demais para a tela: os balões somados passam
// da altura do Frame, então a sobreposição é inevitável e o que se cobra é só
// que ninguém caia fora da tela.
func saturada(n int) scene.Frame {
	f := colunaAnotada(n, 12)
	f.A = 300
	return f
}

// umNotaoNumFrameBaixo: um balão de quatro linhas num Frame de 60 px de altura.
// Nenhuma posição o faz caber; ele encosta no topo e o texto é cortado.
func umNotaoNumFrameBaixo() scene.Frame {
	return scene.Frame{Nome: "baixo", L: 400, A: 60, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 20, Y: 20, L: 60, A: 20, Tom: 300, Nota: repetePalavras(160)}},
	}}}
}

// frameMinusculo é menor que um balão de uma linha só: 20x20 contra os ~30 px
// de 2*respiro mais a altura de linha.
func frameMinusculo() scene.Frame {
	return scene.Frame{Nome: "minusculo", L: 20, A: 20, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 0, Y: 0, L: 5, A: 5, Tom: 300, Nota: "Oi"}},
	}}}
}

// TestDesempateDaOrdenacaoPelaBordaDireitaDoElemento fecha o segundo critério
// de colhe, o que separa dois Elementos na mesma altura.
//
// Os dois começam na mesma borda esquerda e levam o mesmo texto: a altura da
// âncora e a borda esquerda empatam, e só a borda DIREITA — a largura — os
// separa. Ela decide o layout porque as duas bordas direitas estão a menos de
// um balão de distância: os dois querem a direita, na mesma altura, e quem
// chega depois é empurrado para a esquerda do seu Elemento. Sem o desempate,
// quem chega antes passa a ser quem foi declarado antes, e a imagem muda sem a
// geometria ter mudado.
func TestDesempateDaOrdenacaoPelaBordaDireitaDoElemento(t *testing.T) {
	estreito := scene.Elemento{X: 100, Y: 0, L: 50, A: 20, Tom: 300, Nota: "Nota A"}
	largo := scene.Elemento{X: 100, Y: 0, L: 80, A: 20, Tom: 300, Nota: "Nota A"}

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

	// E o desempate escolhe pela geometria: quem termina mais à esquerda é
	// sempre o primeiro atendido, e portanto quem fica com a direita.
	if direta.notas[0].direitaDoElemento != estreito.X+estreito.L {
		t.Errorf("a primeira Nota veio do Elemento que termina em X=%v, quer o que termina em X=%v",
			direta.notas[0].direitaDoElemento, estreito.X+estreito.L)
	}
	// A fixture só prova alguma coisa se os dois disputarem mesmo o lugar:
	// balões idênticos passariam no teste sem exercitar o desempate.
	if direta.notas[0].balao() == direta.notas[1].balao() {
		t.Fatalf("os dois balões caíram no mesmo lugar (%+v): a fixture não força disputa", direta.notas[0].balao())
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
