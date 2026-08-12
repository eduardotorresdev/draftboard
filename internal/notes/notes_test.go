package notes_test

import (
	"bytes"
	"flag"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/webp"

	"github.com/eduardotorresdev/draftboard/internal/notes"
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Os testes usam o seam de imagem: um scene.Frame construído à mão passa por
// Planeja + render.DesenhaFrame + Desenha, e as asserções são sobre a imagem
// DECODIFICADA. Nada aqui conhece a estrutura interna do Plano.

// dirGolden guarda fixtures e goldens desta funcionalidade.
const dirGolden = "../../testdata/f4"

// atualizar regrava os goldens em vez de compará-los.
var atualizar = flag.Bool("atualizar", false, "regrava os goldens de f4")

// cinzaChrome é o valor de cinza de scene.TomChrome: o fundo do Chrome e do
// balão flutuante.
const cinzaChrome = 0x11

// notaLonga é uma frase inteira: quebra em várias linhas e faz o Chrome
// crescer.
const notaLonga = "Este cabeçalho fica fixo no topo da tela e some quando o usuário rola para baixo mais de duzentos pixels."

// larguraMaximaEsperada é o teto, em px de tinta, de uma linha de texto quebrada
// pelo layout. Vale a largura máxima fixa escolhida pelo pacote mais uma folga
// de antialiasing.
const larguraMaximaEsperada = 190

// --- os três modos ---------------------------------------------------------

func TestModoMargemEnvolveOFrameComChrome(t *testing.T) {
	f := frameComUmaNota("ok")

	plano := notes.Planeja(f, notes.Margem, 1)
	cima, direita, baixo, esquerda := plano.Margens()

	if direita <= 0 {
		t.Fatalf("modo Margem sem Chrome: margem direita = %v", direita)
	}
	if esquerda != 0 {
		t.Errorf("Chrome à esquerda inesperado: %v", esquerda)
	}

	img := desenha(t, f, notes.Margem, 1)
	querL := arredonda(esquerda+float64(f.L)+direita, 1)
	querA := arredonda(cima+float64(f.A)+baixo, 1)
	if got := img.Bounds().Dx(); got != querL {
		t.Errorf("largura da tela = %d, quer %d", got, querL)
	}
	if got := img.Bounds().Dy(); got != querA {
		t.Errorf("altura da tela = %d, quer %d", got, querA)
	}

	// O texto da Nota é claro sobre o Chrome escuro e vive fora do Frame.
	if caixa := caixaDeTinta(img, colunaDeNotas(img, f, esquerda), claro); caixa.Empty() {
		t.Fatal("nenhum texto de Nota no Chrome")
	}
}

func TestModoFlutuanteMantemAsDimensoesDoFrame(t *testing.T) {
	f := frameComUmaNota("ok")

	plano := notes.Planeja(f, notes.Flutuante, 1)
	if cima, direita, baixo, esquerda := plano.Margens(); cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
		t.Fatalf("modo Flutuante pediu Chrome: %v %v %v %v", cima, direita, baixo, esquerda)
	}

	img := desenha(t, f, notes.Flutuante, 1)
	if img.Bounds().Dx() != f.L || img.Bounds().Dy() != f.A {
		t.Fatalf("tela = %dx%d, quer %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), f.L, f.A)
	}

	// A Nota está desenhada sobre o desenho: há balão (Tom do Chrome) dentro
	// do Frame, o que nenhum Elemento produz, e texto claro dentro dele.
	if caixaDoBalao(img).Empty() {
		t.Fatal("nenhum balão de Nota sobre o desenho no modo Flutuante")
	}
	if textoNoBalao(img).Empty() {
		t.Error("nenhum texto de Nota dentro do balão no modo Flutuante")
	}
}

func TestModoDesligadoNaoDesenhaNada(t *testing.T) {
	f := frameComUmaNota(notaLonga)

	plano := notes.Planeja(f, notes.Desligado, 1)
	if cima, direita, baixo, esquerda := plano.Margens(); cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
		t.Fatalf("modo Desligado pediu Chrome: %v %v %v %v", cima, direita, baixo, esquerda)
	}

	comPlano := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	plano.Desenha(comPlano)
	semPlano := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)

	if !bytes.Equal(codifica(t, comPlano), codifica(t, semPlano)) {
		t.Error("o modo Desligado desenhou alguma coisa")
	}
}

// --- crescimento da margem -------------------------------------------------

func TestMargemCresceComTextoLongoESemTruncar(t *testing.T) {
	curta := frameComUmaNota("ok")
	longa := frameComUmaNota(notaLonga)

	_, direitaCurta, _, _ := notes.Planeja(curta, notes.Margem, 1).Margens()
	_, direitaLonga, _, _ := notes.Planeja(longa, notes.Margem, 1).Margens()

	if direitaLonga <= direitaCurta {
		t.Fatalf("Chrome não cresceu: curta = %v, longa = %v", direitaCurta, direitaLonga)
	}

	// O texto cabe inteiro: nada de tinta encostando na borda da tela.
	img := desenha(t, longa, notes.Margem, 1)
	caixa := caixaDeTinta(img, colunaDeNotas(img, longa, 0), claro)
	if caixa.Empty() {
		t.Fatal("nenhum texto de Nota no Chrome")
	}
	if caixa.Max.X >= img.Bounds().Dx()-1 {
		t.Errorf("texto truncado na borda direita: tinta vai até x=%d, tela tem %d", caixa.Max.X, img.Bounds().Dx())
	}
	if caixa.Min.Y <= 0 || caixa.Max.Y >= img.Bounds().Dy()-1 {
		t.Errorf("texto encostando na borda vertical: %v em tela de altura %d", caixa, img.Bounds().Dy())
	}
}

func TestPalavraMaiorQueALarguraMaximaAlargaOChrome(t *testing.T) {
	// Uma palavra sozinha, larga demais para a largura máxima de linha, não
	// pode ser truncada: a faixa de Chrome acompanha.
	palavrao := strings.Repeat("W", 80)

	_, direitaNormal, _, _ := notes.Planeja(frameComUmaNota(notaLonga), notes.Margem, 1).Margens()
	_, direitaPalavrao, _, _ := notes.Planeja(frameComUmaNota(palavrao), notes.Margem, 1).Margens()

	if direitaPalavrao <= direitaNormal {
		t.Fatalf("Chrome não acompanhou a palavra larga: %v <= %v", direitaPalavrao, direitaNormal)
	}

	img := desenha(t, frameComUmaNota(palavrao), notes.Margem, 1)
	caixa := caixaDeTinta(img, colunaDeNotas(img, frameComUmaNota(palavrao), 0), claro)
	if caixa.Empty() {
		t.Fatal("nenhum texto de Nota no Chrome")
	}
	if caixa.Max.X >= img.Bounds().Dx()-1 {
		t.Errorf("palavra truncada: tinta até x=%d de %d", caixa.Max.X, img.Bounds().Dx())
	}
	if blocos := blocosDeNota(img, colunaDeNotas(img, frameComUmaNota(palavrao), 0)); len(blocos) != 1 {
		t.Errorf("a palavra indivisível virou %d blocos, quer 1", len(blocos))
	}
}

func TestTextoQuebraEmMultiplasLinhas(t *testing.T) {
	curta := frameComUmaNota("ok")
	imgCurta := desenha(t, curta, notes.Margem, 1)
	umaLinha := blocosDeNota(imgCurta, colunaDeNotas(imgCurta, curta, 0))
	if len(umaLinha) != 1 {
		t.Fatalf("a Nota curta virou %d blocos, quer 1", len(umaLinha))
	}

	f := frameComUmaNota(notaLonga)
	img := desenha(t, f, notes.Margem, 1)
	blocos := blocosDeNota(img, colunaDeNotas(img, f, 0))
	if len(blocos) != 1 {
		t.Fatalf("a frase inteira virou %d blocos de Nota, quer 1", len(blocos))
	}

	// Uma frase inteira ocupa várias linhas empilhadas: o bloco é muito mais
	// alto que o de uma linha só. Truncar em uma linha derrubaria isto.
	if blocos[0].Dy() < 3*umaLinha[0].Dy() {
		t.Errorf("a frase não quebrou em várias linhas: bloco de %d px contra %d px de uma linha",
			blocos[0].Dy(), umaLinha[0].Dy())
	}
	// E quebra numa largura máxima fixa, em vez de esticar numa linha só.
	if blocos[0].Dx() > larguraMaximaEsperada {
		t.Errorf("a linha passou da largura máxima: %d px", blocos[0].Dx())
	}
}

// --- empilhamento ----------------------------------------------------------

func TestNotasNaMesmaAlturaSaoEmpilhadasSemSobreposicao(t *testing.T) {
	// Três Elementos com a MESMA altura de âncora: sem empilhamento as três
	// Notas cairiam umas sobre as outras e a tinta viraria uma banda só.
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome: "conteudo",
		Elementos: []scene.Elemento{
			{X: 10, Y: 100, L: 40, A: 40, Tom: 300, Nota: "um"},
			{X: 60, Y: 100, L: 40, A: 40, Tom: 300, Nota: "dois"},
			{X: 110, Y: 100, L: 40, A: 40, Tom: 300, Nota: "tres"},
		},
	}}}

	img := desenha(t, f, notes.Margem, 1)
	blocos := blocosDeNota(img, colunaDeNotas(img, f, 0))
	if len(blocos) != 3 {
		t.Fatalf("três Notas na mesma altura viraram %d bloco(s) de tinta, quer 3", len(blocos))
	}
	for i := 1; i < len(blocos); i++ {
		if blocos[i].Min.Y <= blocos[i-1].Max.Y {
			t.Errorf("Notas sobrepostas: bloco %d termina em %d e bloco %d começa em %d",
				i-1, blocos[i-1].Max.Y, i, blocos[i].Min.Y)
		}
	}
}

func TestNotasSaoOrdenadasPelaAlturaDaAncora(t *testing.T) {
	// A Nota de quatro linhas está ancorada EMBAIXO e é declarada PRIMEIRO;
	// a de uma linha está ancorada em cima e é declarada por último. Ordenar
	// por altura de âncora coloca a curta no topo da pilha: o bloco de cima
	// tem que ser o mais baixo dos dois. Sem ordenação a ordem seria a de
	// declaração e o bloco alto viria primeiro.
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome: "conteudo",
		Elementos: []scene.Elemento{
			{X: 10, Y: 230, L: 40, A: 40, Tom: 300, Nota: notaLonga},
			{X: 10, Y: 10, L: 40, A: 40, Tom: 300, Nota: "ok"},
		},
	}}}

	img := desenha(t, f, notes.Margem, 1)
	blocos := blocosDeNota(img, colunaDeNotas(img, f, 0))
	if len(blocos) != 2 {
		t.Fatalf("%d blocos de Nota, quer 2", len(blocos))
	}
	if blocos[0].Dy() >= blocos[1].Dy() {
		t.Fatalf("a Nota de uma linha não ficou no topo: bloco de cima tem %d px, o de baixo %d px",
			blocos[0].Dy(), blocos[1].Dy())
	}
}

// --- linha de chamada ------------------------------------------------------

func TestLinhaDeChamadaLigaNotaEElemento(t *testing.T) {
	// O Elemento fica à esquerda, longe da coluna de Notas: a linha de
	// chamada tem que atravessar o Frame claro e o vão do Chrome escuro.
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 20, Y: 120, L: 60, A: 60, Tom: 300, Nota: "ok"}},
	}}}

	img := desenha(t, f, notes.Margem, 1)

	// Dentro do Frame, à direita do Elemento, o fundo é o claro do Frame e
	// não há Elemento nenhum: qualquer tinta média ali é a linha de chamada.
	dentro := image.Rect(120, 0, f.L, f.A)
	if conta(img, dentro, medio) == 0 {
		t.Error("a linha de chamada não atravessa o Frame até o Elemento")
	}

	// No vão entre a borda do Frame e o texto o fundo é o Chrome escuro; de
	// novo, só a linha de chamada pinta ali.
	vao := image.Rect(f.L, 0, f.L+6, f.A)
	if conta(img, vao, medio) == 0 {
		t.Error("a linha de chamada não chega ao Chrome onde a Nota está")
	}
}

// --- estabilidade -----------------------------------------------------------

func TestLayoutEstavelSobReordenacaoDosElementos(t *testing.T) {
	a := scene.Elemento{X: 10, Y: 20, L: 60, A: 30, Tom: 300, Nota: "topo da tela"}
	b := scene.Elemento{X: 10, Y: 120, L: 60, A: 30, Tom: 300, Nota: notaLonga}
	c := scene.Elemento{X: 10, Y: 220, L: 60, A: 30, Tom: 300, Nota: "rodapé"}

	direta := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{
		{Nome: "conteudo", Elementos: []scene.Elemento{a, b, c}},
	}}
	// Mesma geometria, ordem de declaração embaralhada e repartida em duas
	// Camadas. Nada disso pode mexer no layout das Notas.
	embaralhada := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{
		{Nome: "conteudo", Elementos: []scene.Elemento{c}},
		{Nome: "extra", Elementos: []scene.Elemento{b, a}},
	}}

	if !bytes.Equal(anota(t, direta, notes.Margem, 1), anota(t, embaralhada, notes.Margem, 1)) {
		t.Error("reordenar os Elementos mudou o layout das Notas")
	}
}

// --- escala ----------------------------------------------------------------

func TestEscalaMultiplicaOLayoutProporcionalmente(t *testing.T) {
	f := frameComTresNotas()

	um := desenha(t, f, notes.Margem, 1)
	dois := desenha(t, f, notes.Margem, 2)

	esperadoL := 2 * um.Bounds().Dx()
	esperadoA := 2 * um.Bounds().Dy()
	// A medição do texto depende do hinting da fonte no tamanho de
	// dispositivo, então a proporção é exata a menos de alguns pixels.
	if !porPerto(dois.Bounds().Dx(), esperadoL, 0.02) {
		t.Errorf("largura na escala 2 = %d, quer ~%d", dois.Bounds().Dx(), esperadoL)
	}
	if !porPerto(dois.Bounds().Dy(), esperadoA, 0.02) {
		t.Errorf("altura na escala 2 = %d, quer ~%d", dois.Bounds().Dy(), esperadoA)
	}

	// O Chrome em px do espaço do Frame não depende da escala: é o Canvas
	// que multiplica.
	_, direitaUm, _, _ := notes.Planeja(f, notes.Margem, 1).Margens()
	_, direitaDois, _, _ := notes.Planeja(f, notes.Margem, 2).Margens()
	if !porPerto(int(direitaDois), int(direitaUm), 0.02) {
		t.Errorf("Chrome na escala 2 = %v, quer ~%v", direitaDois, direitaUm)
	}
}

// --- casos de borda --------------------------------------------------------

func TestFrameSemNotaNaoPedeChromeEmNenhumModo(t *testing.T) {
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 10, Y: 10, L: 100, A: 40, Tom: 300}},
	}}}

	for _, modo := range []notes.Modo{notes.Margem, notes.Flutuante, notes.Desligado} {
		cima, direita, baixo, esquerda := notes.Planeja(f, modo, 1).Margens()
		if cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
			t.Errorf("modo %v pediu Chrome sem Nota nenhuma: %v %v %v %v", modo, cima, direita, baixo, esquerda)
		}
		img := desenha(t, f, modo, 1)
		if img.Bounds().Dx() != f.L || img.Bounds().Dy() != f.A {
			t.Errorf("modo %v: tela = %dx%d, quer %dx%d", modo, img.Bounds().Dx(), img.Bounds().Dy(), f.L, f.A)
		}
	}
}

func TestNotaVaziaEAusenciaDeNota(t *testing.T) {
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome: "conteudo",
		Elementos: []scene.Elemento{
			{X: 10, Y: 10, L: 100, A: 40, Tom: 300, Nota: ""},
			{X: 10, Y: 80, L: 100, A: 40, Tom: 300, Nota: "   "},
		},
	}}}

	for _, modo := range []notes.Modo{notes.Margem, notes.Flutuante, notes.Desligado} {
		if cima, direita, baixo, esquerda := notes.Planeja(f, modo, 1).Margens(); cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
			t.Errorf("modo %v: Nota vazia pediu Chrome: %v %v %v %v", modo, cima, direita, baixo, esquerda)
		}
	}

	semNota := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	comNotaVazia := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	notes.Planeja(f, notes.Margem, 1).Desenha(comNotaVazia)
	if !bytes.Equal(codifica(t, semNota), codifica(t, comNotaVazia)) {
		t.Error("Nota vazia desenhou alguma coisa")
	}
}

func TestNotaEmElementoNaBordaDoFrame(t *testing.T) {
	// Elementos colados nas quatro bordas: a âncora sai do Frame e a pilha
	// transborda em cima e embaixo. O Chrome tem que crescer nos dois
	// sentidos e nenhum texto pode sair da tela.
	f := scene.Frame{Nome: "home", L: 200, A: 120, Camadas: []scene.Camada{{
		Nome: "conteudo",
		Elementos: []scene.Elemento{
			{X: 0, Y: 0, L: 200, A: 10, Tom: 300, Nota: "topo"},
			{X: 0, Y: 110, L: 200, A: 10, Tom: 300, Nota: notaLonga},
		},
	}}}

	plano := notes.Planeja(f, notes.Margem, 1)
	cima, direita, baixo, _ := plano.Margens()
	if cima <= 0 {
		t.Errorf("Chrome não cresceu para cima: %v", cima)
	}
	if baixo <= 0 {
		t.Errorf("Chrome não cresceu para baixo: %v", baixo)
	}

	img := desenha(t, f, notes.Margem, 1)
	caixa := caixaDeTinta(img, colunaDeNotas(img, f, 0), claro)
	if caixa.Empty() {
		t.Fatal("nenhum texto de Nota no Chrome")
	}
	if caixa.Min.Y <= 0 || caixa.Max.Y >= img.Bounds().Dy()-1 || caixa.Max.X >= img.Bounds().Dx()-1 {
		t.Errorf("texto cortado pela tela: %v em tela %v (chrome direito %v)", caixa, img.Bounds(), direita)
	}

	// No modo Flutuante o mesmo Frame não cresce e o balão fica dentro.
	if c, d, b, e := notes.Planeja(f, notes.Flutuante, 1).Margens(); c != 0 || d != 0 || b != 0 || e != 0 {
		t.Errorf("modo Flutuante pediu Chrome: %v %v %v %v", c, d, b, e)
	}
	flutuante := desenha(t, f, notes.Flutuante, 1)
	if flutuante.Bounds().Dx() != f.L || flutuante.Bounds().Dy() != f.A {
		t.Errorf("modo Flutuante mudou a tela: %v", flutuante.Bounds())
	}
	if textoNoBalao(flutuante).Empty() {
		t.Error("nenhum texto flutuante desenhado")
	}
}

// --- desempate da ordenação -------------------------------------------------

func TestDesempateDaOrdenacaoPelaBordaDoElemento(t *testing.T) {
	// Duas âncoras na MESMA altura: quem decide é a borda direita do
	// Elemento, da esquerda para a direita. O Elemento da esquerda leva a
	// Nota de várias linhas e um texto que ordena por último no alfabeto; o
	// da direita leva a Nota de uma linha e um texto que ordena primeiro.
	// Assim, se a borda deixasse de desempatar, o texto assumiria e a ordem
	// se inverteria.
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome: "conteudo",
		Elementos: []scene.Elemento{
			{X: 10, Y: 100, L: 40, A: 40, Tom: 300, Nota: "zzz " + notaLonga},
			{X: 300, Y: 100, L: 40, A: 40, Tom: 300, Nota: "aa"},
		},
	}}}

	img := desenha(t, f, notes.Margem, 1)
	blocos := blocosDeNota(img, colunaDeNotas(img, f, 0))
	if len(blocos) != 2 {
		t.Fatalf("%d blocos de Nota, quer 2", len(blocos))
	}
	if blocos[0].Dy() <= blocos[1].Dy() {
		t.Errorf("a Nota do Elemento mais à esquerda não veio primeiro: bloco de cima tem %d px, o de baixo %d px",
			blocos[0].Dy(), blocos[1].Dy())
	}
}

func TestDesempateDaOrdenacaoPeloTextoDaNota(t *testing.T) {
	// Geometria IDENTICA nos dois Elementos: altura e borda empatam, e só o
	// texto sobra para fechar a ordem. Sem esse último desempate a ordenação
	// fica indefinida e passa a depender da ordem de declaração.
	a := scene.Elemento{X: 10, Y: 100, L: 40, A: 40, Tom: 300, Nota: "aa"}
	b := scene.Elemento{X: 10, Y: 100, L: 40, A: 40, Tom: 300, Nota: "zzz " + notaLonga}

	direta := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{
		{Nome: "conteudo", Elementos: []scene.Elemento{a, b}},
	}}
	invertida := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{
		{Nome: "conteudo", Elementos: []scene.Elemento{b, a}},
	}}

	if !bytes.Equal(anota(t, direta, notes.Margem, 1), anota(t, invertida, notes.Margem, 1)) {
		t.Error("com âncoras idênticas a ordem passou a depender da declaração")
	}
}

// --- régua e escalas degeneradas --------------------------------------------

func TestPlanoEMedidoNaEscalaEmQueSeraPintado(t *testing.T) {
	// O texto é medido com a fonte no tamanho de dispositivo, e o hinting faz
	// a largura de uma linha não ser exatamente proporcional ao fator. Se a
	// régua ignorasse a escala, o Chrome seria idêntico em todas elas — e o
	// planejado deixaria de ser o que realmente cabe quando o texto é pintado.
	f := frameComUmaNota(notaLonga)
	_, base, _, _ := notes.Planeja(f, notes.Margem, 1).Margens()

	for _, escala := range []float64{2, 3, 5, 8} {
		if _, d, _, _ := notes.Planeja(f, notes.Margem, escala).Margens(); d != base {
			return
		}
	}
	t.Errorf("o Chrome não mudou em nenhuma escala testada: sempre %v — a régua está ignorando o fator", base)
}

func TestEscalaNaoUtilizavelNaoDerrubaOPlano(t *testing.T) {
	// A CLI recusa escala não-positiva antes de chegar aqui, mas a biblioteca
	// não pode entrar em pânico nem devolver medida infinita se alguém a
	// chamar direto: a escala degenerada cai para 1.
	f := frameComUmaNota(notaLonga)
	cimaUm, direitaUm, baixoUm, esquerdaUm := notes.Planeja(f, notes.Margem, 1).Margens()

	for _, escala := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -3} {
		cima, direita, baixo, esquerda := notes.Planeja(f, notes.Margem, escala).Margens()
		for _, m := range []float64{cima, direita, baixo, esquerda} {
			if math.IsNaN(m) || math.IsInf(m, 0) {
				t.Fatalf("escala %v produziu margem não finita: %v %v %v %v", escala, cima, direita, baixo, esquerda)
			}
		}
		if cima != cimaUm || direita != direitaUm || baixo != baixoUm || esquerda != esquerdaUm {
			t.Errorf("escala %v deu %v %v %v %v, quer o mesmo da escala 1: %v %v %v %v",
				escala, cima, direita, baixo, esquerda, cimaUm, direitaUm, baixoUm, esquerdaUm)
		}
	}
}

func TestPlanoNiloSeComportaComoPlanoVazio(t *testing.T) {
	// Um *Plano nulo é o zero natural do tipo; quem o receber de um caminho de
	// erro não pode ver a CLI cair.
	var p *notes.Plano
	if cima, direita, baixo, esquerda := p.Margens(); cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
		t.Errorf("Plano nulo pediu Chrome: %v %v %v %v", cima, direita, baixo, esquerda)
	}

	f := frameComUmaNota("ok")
	semPlano := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	comPlanoNulo := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	p.Desenha(comPlanoNulo)
	if !bytes.Equal(codifica(t, semPlano), codifica(t, comPlanoNulo)) {
		t.Error("Plano nulo desenhou alguma coisa")
	}

	// E um Canvas nulo também não derruba um Plano de verdade.
	notes.Planeja(f, notes.Margem, 1).Desenha(nil)
}

// --- piso da faixa de Chrome ------------------------------------------------

func TestFaixaDeChromeTemLarguraMinima(t *testing.T) {
	// A Nota mais curta possível mede muito menos que a faixa mínima: o
	// Chrome não encolhe até virar um filete grudado no Frame, para de
	// encolher no piso.
	const larguraMinimaDaFaixa = 48.0

	curta := frameComUmaNota("ok")
	_, direitaCurta, _, _ := notes.Planeja(curta, notes.Margem, 1).Margens()
	if direitaCurta != larguraMinimaDaFaixa {
		t.Errorf("faixa de Chrome com Nota curta = %v, quer o piso de %v", direitaCurta, larguraMinimaDaFaixa)
	}

	// E o piso é piso, não teto: uma Nota que precisa de mais que isso passa
	// direto por ele.
	_, direitaLonga, _, _ := notes.Planeja(frameComUmaNota(notaLonga), notes.Margem, 1).Margens()
	if direitaLonga <= larguraMinimaDaFaixa {
		t.Errorf("faixa de Chrome com Nota longa = %v, quer acima do piso de %v", direitaLonga, larguraMinimaDaFaixa)
	}
}

// --- modo Flutuante: encaixe no Frame ---------------------------------------

func TestBalaoFlutuanteQuebraNoQueCabeNumFrameEstreito(t *testing.T) {
	// Num Frame mais estreito que a largura máxima de linha, a quebra usa a
	// largura que existe. Sem isso o texto seria quebrado em 180 px e sairia
	// pela borda do Frame, que aqui não tem para onde crescer.
	f := flutuanteComUmaNota("estreito", 120, 200, 10, 90, 20, 20, notaLonga)

	img := desenha(t, f, notes.Flutuante, 1)
	texto := textoNoBalao(img)
	if texto.Empty() {
		t.Fatal("nenhum texto dentro do balão")
	}
	if texto.Min.X < 4 || texto.Max.X > f.L-4 {
		t.Errorf("o texto foi parar na borda do Frame: %v numa tela de largura %d", texto, f.L)
	}
}

func TestBalaoFlutuanteEPresoDentroDoFrame(t *testing.T) {
	// Elemento colado no topo e Elemento colado na base: o balão iria para
	// fora da tela, e no modo Flutuante não há margem para acompanhá-lo. Ele
	// é preso dentro do Frame, com todas as linhas visíveis.
	const nota = notaLonga

	meio := flutuanteComUmaNota("meio", 400, 200, 10, 90, 20, 6, nota)
	alturaInteira := textoNoBalao(desenha(t, meio, notes.Flutuante, 1)).Dy()
	if alturaInteira == 0 {
		t.Fatal("nenhum texto dentro do balão")
	}

	casos := []struct {
		nome string
		f    scene.Frame
	}{
		{"colado no topo", flutuanteComUmaNota("topo", 400, 200, 10, 0, 20, 6, nota)},
		{"colado na base", flutuanteComUmaNota("base", 400, 200, 10, 194, 20, 6, nota)},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			img := desenha(t, caso.f, notes.Flutuante, 1)
			texto := textoNoBalao(img)
			if texto.Empty() {
				t.Fatal("nenhum texto dentro do balão")
			}
			if texto.Min.Y < 4 || texto.Max.Y > caso.f.A-4 {
				t.Errorf("o balão escapou pela borda: %v numa tela de altura %d", texto, caso.f.A)
			}
			// Nenhuma linha foi cortada pela borda: o bloco tem a mesma
			// altura de quando a âncora está no meio do Frame. A folga de
			// 2 px absorve o arredondamento de um topo fracionário — uma
			// linha perdida custaria mais de dez.
			if texto.Dy() < alturaInteira-2 {
				t.Errorf("linhas perdidas na borda: bloco de %d px, quer ~%d px", texto.Dy(), alturaInteira)
			}
		})
	}
}

// --- goldens ---------------------------------------------------------------

// TestGoldens fixa os bytes exatos dos três modos sobre a mesma fixture. É
// também o teste de determinismo: os goldens foram gravados noutro processo, em
// outra execução, então bater byte a byte com eles é uma garantia mais forte do
// que renderizar duas vezes no mesmo processo e comparar.
func TestGoldens(t *testing.T) {
	f := frameComTresNotas()
	casos := []struct {
		nome string
		modo notes.Modo
	}{
		{"margem.webp", notes.Margem},
		{"flutuante.webp", notes.Flutuante},
		{"desligado.webp", notes.Desligado},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			comparaGolden(t, caso.nome, anota(t, f, caso.modo, 1))
		})
	}
}

// --- fixtures --------------------------------------------------------------

func frameComUmaNota(texto string) scene.Frame {
	return scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 20, Y: 30, L: 200, A: 60, Tom: 300, Nota: texto}},
	}}}
}

// flutuanteComUmaNota monta um Frame com um único Elemento anotado, em posição
// escolhida — é o que os casos do modo Flutuante precisam variar.
func flutuanteComUmaNota(nome string, l, a int, x, y, el, ea float64, nota string) scene.Frame {
	return scene.Frame{Nome: nome, L: l, A: a, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: x, Y: y, L: el, A: ea, Tom: 300, Nota: nota}},
	}}}
}

func frameComTresNotas() scene.Frame {
	return scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{
		{Nome: "fundo", Elementos: []scene.Elemento{
			{X: 20, Y: 20, L: 360, A: 50, Arredondado: true, Tom: 300, Nota: "Cabeçalho fixo"},
		}},
		{Nome: "conteudo", Elementos: []scene.Elemento{
			{X: 20, Y: 100, L: 160, A: 120, Tom: 500, Nota: notaLonga},
			{Forma: scene.Circulo, X: 300, Y: 240, L: 40, A: 40, Tom: 500, Nota: "Ação principal"},
		}},
	}}
}

// --- utilitários de imagem -------------------------------------------------

// anota roda o fluxo inteiro — Planeja, DesenhaFrame com as margens pedidas,
// Desenha — e devolve os bytes WebP.
func anota(t *testing.T, f scene.Frame, m notes.Modo, escala float64) []byte {
	t.Helper()
	plano := notes.Planeja(f, m, escala)
	cima, direita, baixo, esquerda := plano.Margens()
	tela := render.DesenhaFrame(f, escala, cima, direita, baixo, esquerda, -1)
	plano.Desenha(tela)
	return codifica(t, tela)
}

func desenha(t *testing.T, f scene.Frame, m notes.Modo, escala float64) image.Image {
	t.Helper()
	return decodifica(t, anota(t, f, m, escala))
}

func codifica(t *testing.T, c *render.Canvas) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := c.CodificaWebP(&buf); err != nil {
		t.Fatalf("codificação WebP: %v", err)
	}
	return buf.Bytes()
}

func decodifica(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := webp.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("decodificação WebP: %v", err)
	}
	return img
}

// cinzaEm devolve o valor de cinza de um pixel.
func cinzaEm(img image.Image, x, y int) uint8 {
	r, _, _, _ := img.At(x, y).RGBA()
	return uint8(r >> 8)
}

// Classificadores de tinta. O fundo do Chrome é quase preto e o do Frame quase
// branco, então uma faixa de valores basta para distinguir o que foi pintado
// por cima sem depender de antialiasing exato.
type criterio func(uint8) bool

// claro é o texto da Nota, o extremo claro da escala, pintado sobre o Chrome.
func claro(v uint8) bool { return v > 0xC0 }

// medio é a linha de chamada: nem o claro do Frame nem o escuro do Chrome.
func medio(v uint8) bool { return v > 0x30 && v < 0xC0 }

func exatamente(v uint8) criterio {
	return func(g uint8) bool { return g == v }
}

// caixaDoBalao devolve o retângulo do balão de uma Nota flutuante. O balão é
// pintado com o Tom reservado do Chrome, que nenhum Elemento pode ter, então
// ele se identifica sozinho dentro do Frame.
func caixaDoBalao(img image.Image) image.Rectangle {
	return caixaDeTinta(img, img.Bounds(), exatamente(cinzaChrome))
}

// textoNoBalao devolve a caixa do texto de uma Nota flutuante.
//
// A busca é restrita ao balão de propósito: o fundo do Frame é o mesmo extremo
// claro do texto, então procurar tinta clara na tela inteira acharia o Frame
// e não asseguraria nada. Dentro do balão, escuro, só o texto é claro.
func textoNoBalao(img image.Image) image.Rectangle {
	balao := caixaDoBalao(img)
	if balao.Empty() {
		return balao
	}
	return caixaDeTinta(img, balao, claro)
}

// colunaDeNotas é o retângulo da coluna de Notas: o Chrome à direita do Frame,
// do topo ao pé da tela.
func colunaDeNotas(img image.Image, f scene.Frame, margemEsquerda float64) image.Rectangle {
	x0 := int(margemEsquerda) + f.L
	return image.Rect(x0, 0, img.Bounds().Dx(), img.Bounds().Dy())
}

func conta(img image.Image, r image.Rectangle, c criterio) int {
	r = r.Intersect(img.Bounds())
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c(cinzaEm(img, x, y)) {
				n++
			}
		}
	}
	return n
}

// caixaDeTinta devolve a menor caixa que contém toda a tinta que satisfaz o
// critério dentro da região.
func caixaDeTinta(img image.Image, r image.Rectangle, c criterio) image.Rectangle {
	r = r.Intersect(img.Bounds())
	caixa := image.Rectangle{}
	primeiro := true
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if !c(cinzaEm(img, x, y)) {
				continue
			}
			p := image.Rect(x, y, x+1, y+1)
			if primeiro {
				caixa, primeiro = p, false
				continue
			}
			caixa = caixa.Union(p)
		}
	}
	return caixa
}

// bandasDeTinta devolve, de cima para baixo, as faixas contínuas de linhas que
// contêm tinta dentro da região. Cada linha de texto vira uma banda, porque a
// altura da caixa de linha garante que os glifos de linhas vizinhas nunca se
// toquem.
func bandasDeTinta(img image.Image, r image.Rectangle, c criterio) []image.Rectangle {
	r = r.Intersect(img.Bounds())
	var bandas []image.Rectangle
	dentro := false
	inicio := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		temTinta := conta(img, image.Rect(r.Min.X, y, r.Max.X, y+1), c) > 0
		switch {
		case temTinta && !dentro:
			dentro, inicio = true, y
		case !temTinta && dentro:
			bandas = append(bandas, image.Rect(r.Min.X, inicio, r.Max.X, y))
			dentro = false
		}
	}
	if dentro {
		bandas = append(bandas, image.Rect(r.Min.X, inicio, r.Max.X, r.Max.Y))
	}
	return bandas
}

// blocosDeNota devolve, de cima para baixo, a caixa de tinta de cada Nota
// desenhada na região.
//
// A tinta é agrupada por proximidade vertical: linhas da mesma Nota — e até
// partes do mesmo glifo, como a haste de um "d" — ficam a poucos pixels de
// distância, enquanto Notas diferentes são separadas pelo espaço entre Notas,
// bem maior. Duas Notas sobrepostas viram um bloco só, que é exatamente o que
// os testes de empilhamento procuram.
func blocosDeNota(img image.Image, r image.Rectangle) []image.Rectangle {
	const separacaoEntreNotas = 8
	var blocos []image.Rectangle
	for _, b := range bandasDeTinta(img, r, claro) {
		caixa := caixaDeTinta(img, b, claro)
		if n := len(blocos); n > 0 && caixa.Min.Y-blocos[n-1].Max.Y < separacaoEntreNotas {
			blocos[n-1] = blocos[n-1].Union(caixa)
			continue
		}
		blocos = append(blocos, caixa)
	}
	return blocos
}

func arredonda(v float64, escala float64) int {
	return int(v*escala + 0.5)
}

func porPerto(got, quer int, tolerancia float64) bool {
	d := got - quer
	if d < 0 {
		d = -d
	}
	return float64(d) <= tolerancia*float64(quer)
}

func comparaGolden(t *testing.T, nome string, obtido []byte) {
	t.Helper()
	caminho := filepath.Join(dirGolden, nome)
	if *atualizar {
		if err := os.MkdirAll(dirGolden, 0o755); err != nil {
			t.Fatalf("criando %s: %v", dirGolden, err)
		}
		if err := os.WriteFile(caminho, obtido, 0o644); err != nil {
			t.Fatalf("gravando %s: %v", caminho, err)
		}
		return
	}
	querido, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo golden %s: %v (rode com -atualizar)", caminho, err)
	}
	if !bytes.Equal(obtido, querido) {
		t.Errorf("golden %s difere: %d bytes obtidos, %d esperados", nome, len(obtido), len(querido))
	}
}
