package notes_test

import (
	"bytes"
	"flag"
	"image"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/webp"

	"github.com/eduardotorresdev/draftboard/internal/notes"
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Os testes usam o seam de imagem: um scene.Frame construído à mão passa por
// Planeja + render.DesenhaFrame + Desenha, e as asserções são sobre a imagem
// DECODIFICADA. Nada aqui conhece a estrutura interna do Plano — o que ela
// promete em geometria é provado em plano_test.go, do lado de dentro.

// dirGolden guarda fixtures e goldens desta funcionalidade.
const dirGolden = "../../testdata/f4"

// atualizar regrava os goldens em vez de compará-los.
var atualizar = flag.Bool("atualizar", false, "regrava os goldens de f4")

// cinzaDoBalao é o valor de cinza de scene.TomChrome, o Tom reservado do balão.
// Nenhum Elemento pode alcançá-lo, então ele identifica o balão sozinho.
const cinzaDoBalao = 0x11

// notaLonga é uma frase inteira: quebra em várias linhas e faz o balão crescer.
const notaLonga = "Este cabeçalho fica fixo no topo da tela e some quando o usuário rola para baixo mais de duzentos pixels."

// larguraMaximaEsperada é o teto, em px de tinta, de uma linha de texto quebrada
// pelo layout. Vale a largura máxima fixa escolhida pelo pacote mais uma folga
// de antialiasing.
const larguraMaximaEsperada = 190

// --- a Nota sobre o desenho -------------------------------------------------

func TestNotaEDesenhadaNumBalaoDentroDoFrame(t *testing.T) {
	f := frameComUmaNota("ok")

	if cima, direita, baixo, esquerda := notes.Planeja(f, 1).Margens(); cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
		t.Fatalf("o plano de anotação pediu margem: %v %v %v %v", cima, direita, baixo, esquerda)
	}

	img := desenha(t, f, 1)
	if img.Bounds().Dx() != f.L || img.Bounds().Dy() != f.A {
		t.Fatalf("tela = %dx%d, quer %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), f.L, f.A)
	}

	// Há balão — Tom reservado, que nenhum Elemento produz — dentro do
	// Frame, e texto claro dentro dele.
	if caixaDoBalao(img).Empty() {
		t.Fatal("nenhum balão de Nota sobre o desenho")
	}
	if textoNoBalao(img).Empty() {
		t.Error("nenhum texto de Nota dentro do balão")
	}
}

func TestSemPlanoNadaEDesenhado(t *testing.T) {
	// Quem não pede Notas passa um *Plano nulo adiante: é o caminho normal
	// da CLI sem `--notes`, não um caso de erro.
	f := frameComUmaNota(notaLonga)
	var p *notes.Plano

	if cima, direita, baixo, esquerda := p.Margens(); cima != 0 || direita != 0 || baixo != 0 || esquerda != 0 {
		t.Errorf("Plano nulo pediu margem: %v %v %v %v", cima, direita, baixo, esquerda)
	}

	comPlano := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	p.Desenha(comPlano)
	semPlano := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	if !bytes.Equal(codifica(t, comPlano), codifica(t, semPlano)) {
		t.Error("o Plano nulo desenhou alguma coisa")
	}

	// E um Canvas nulo também não derruba um Plano de verdade.
	notes.Planeja(f, 1).Desenha(nil)
}

// --- quebra de linha --------------------------------------------------------

func TestTextoQuebraEmMultiplasLinhas(t *testing.T) {
	curta := frameComUmaNota("ok")
	umaLinha := textoNoBalao(desenha(t, curta, 1))
	if umaLinha.Empty() {
		t.Fatal("nenhum texto dentro do balão da Nota curta")
	}

	longa := textoNoBalao(desenha(t, frameComUmaNota(notaLonga), 1))
	if longa.Empty() {
		t.Fatal("nenhum texto dentro do balão da Nota longa")
	}

	// Uma frase inteira ocupa várias linhas empilhadas: o bloco é muito mais
	// alto que o de uma linha só. Truncar em uma linha derrubaria isto.
	if longa.Dy() < 3*umaLinha.Dy() {
		t.Errorf("a frase não quebrou em várias linhas: bloco de %d px contra %d px de uma linha",
			longa.Dy(), umaLinha.Dy())
	}
	// E quebra numa largura máxima fixa, em vez de esticar numa linha só.
	if longa.Dx() > larguraMaximaEsperada {
		t.Errorf("a linha passou da largura máxima: %d px", longa.Dx())
	}
}

// --- anti-colisão -----------------------------------------------------------

func TestNotasVizinhasNaoViramUmBlocoDeTintaSo(t *testing.T) {
	// Três Elementos com âncoras a 10 px umas das outras — menos que a
	// altura de um balão — e colados na borda ESQUERDA do Frame, onde não
	// sobra espaço para nenhum balão. Os três disputam a mesma coluna à
	// direita, e é a anti-colisão que os separa; sem ela o texto viraria uma
	// mancha só.
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome: "conteudo",
		Elementos: []scene.Elemento{
			{X: 0, Y: 100, L: 60, A: 40, Tom: 300, Nota: "um"},
			{X: 0, Y: 110, L: 60, A: 40, Tom: 300, Nota: "dois"},
			{X: 0, Y: 120, L: 60, A: 40, Tom: 300, Nota: "tres"},
		},
	}}}

	img := desenha(t, f, 1)
	if blocos := blocosDeNota(img, img.Bounds()); len(blocos) != 3 {
		t.Errorf("três Notas viraram %d bloco(s) de tinta, quer 3", len(blocos))
	}
}

// --- linha de chamada ------------------------------------------------------

func TestLinhaDeChamadaLigaNotaEElemento(t *testing.T) {
	// O Elemento fica à esquerda e o balão vai para a direita dele: entre os
	// dois há uma faixa de Frame claro onde não existe Elemento nenhum, e
	// qualquer tinta média ali é a linha de chamada.
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 20, Y: 120, L: 60, A: 60, Tom: 300, Nota: "ok"}},
	}}}

	img := desenha(t, f, 1)
	balao := caixaDoBalao(img)
	if balao.Empty() {
		t.Fatal("nenhum balão desenhado")
	}
	vao := image.Rect(81, 0, balao.Min.X, f.A)
	if conta(img, vao, medio) == 0 {
		t.Error("a linha de chamada não liga o Elemento ao balão")
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

	if !bytes.Equal(anota(t, direta, 1), anota(t, embaralhada, 1)) {
		t.Error("reordenar os Elementos mudou o layout das Notas")
	}
}

// --- escala ----------------------------------------------------------------

func TestEscalaMultiplicaOLayoutProporcionalmente(t *testing.T) {
	f := frameComUmaNota(notaLonga)

	um := caixaDoBalao(desenha(t, f, 1))
	dois := caixaDoBalao(desenha(t, f, 2))
	if um.Empty() || dois.Empty() {
		t.Fatal("nenhum balão desenhado")
	}

	// A medição do texto depende do hinting da fonte no tamanho de
	// dispositivo, então a proporção é exata a menos de alguns pixels.
	if !porPerto(dois.Dx(), 2*um.Dx(), 0.05) {
		t.Errorf("largura do balão na escala 2 = %d, quer ~%d", dois.Dx(), 2*um.Dx())
	}
	if !porPerto(dois.Dy(), 2*um.Dy(), 0.05) {
		t.Errorf("altura do balão na escala 2 = %d, quer ~%d", dois.Dy(), 2*um.Dy())
	}
	if !porPerto(dois.Min.X, 2*um.Min.X, 0.05) {
		t.Errorf("o balão na escala 2 começa em x=%d, quer ~%d", dois.Min.X, 2*um.Min.X)
	}
}

// --- casos de borda --------------------------------------------------------

func TestFrameSemNotaNaoDesenhaNada(t *testing.T) {
	f := scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 10, Y: 10, L: 100, A: 40, Tom: 300}},
	}}}

	img := desenha(t, f, 1)
	if img.Bounds().Dx() != f.L || img.Bounds().Dy() != f.A {
		t.Errorf("tela = %dx%d, quer %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), f.L, f.A)
	}
	if !caixaDoBalao(img).Empty() {
		t.Error("Frame sem Nota nenhuma desenhou balão")
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

	semNota := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	comNotaVazia := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	notes.Planeja(f, 1).Desenha(comNotaVazia)
	if !bytes.Equal(codifica(t, semNota), codifica(t, comNotaVazia)) {
		t.Error("Nota vazia desenhou alguma coisa")
	}
}

func TestBalaoQuebraNoQueCabeNumFrameEstreito(t *testing.T) {
	// Num Frame mais estreito que a largura máxima de linha, a quebra usa a
	// largura que existe. Sem isso o texto seria quebrado em 180 px e sairia
	// pela borda do Frame, que aqui não tem para onde crescer.
	f := comUmaNotaEm("estreito", 120, 200, 10, 90, 20, 20, notaLonga)

	texto := textoNoBalao(desenha(t, f, 1))
	if texto.Empty() {
		t.Fatal("nenhum texto dentro do balão")
	}
	if texto.Min.X < 4 || texto.Max.X > f.L-4 {
		t.Errorf("o texto foi parar na borda do Frame: %v numa tela de largura %d", texto, f.L)
	}
}

func TestBalaoEPresoDentroDoFrame(t *testing.T) {
	// Elemento colado no topo e Elemento colado na base: o balão iria para
	// fora da tela, e aqui não há margem para acompanhá-lo. Ele é preso
	// dentro do Frame, com todas as linhas visíveis.
	meio := comUmaNotaEm("meio", 400, 200, 10, 90, 20, 6, notaLonga)
	alturaInteira := textoNoBalao(desenha(t, meio, 1)).Dy()
	if alturaInteira == 0 {
		t.Fatal("nenhum texto dentro do balão")
	}

	casos := []struct {
		nome string
		f    scene.Frame
	}{
		{"colado no topo", comUmaNotaEm("topo", 400, 200, 10, 0, 20, 6, notaLonga)},
		{"colado na base", comUmaNotaEm("base", 400, 200, 10, 194, 20, 6, notaLonga)},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			img := desenha(t, caso.f, 1)
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

// TestGoldens fixa os bytes exatos das duas saídas possíveis sobre a mesma
// fixture: com Notas e sem elas. É também o teste de determinismo: os goldens
// foram gravados noutro processo, em outra execução, então bater byte a byte
// com eles é uma garantia mais forte do que renderizar duas vezes no mesmo
// processo e comparar.
func TestGoldens(t *testing.T) {
	f := frameAnotado()

	t.Run("flutuante.webp", func(t *testing.T) {
		comparaGolden(t, "flutuante.webp", anota(t, f, 1))
	})
	t.Run("desligado.webp", func(t *testing.T) {
		var p *notes.Plano
		tela := render.DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
		p.Desenha(tela)
		comparaGolden(t, "desligado.webp", codifica(t, tela))
	})
}

// --- fixtures --------------------------------------------------------------

func frameComUmaNota(texto string) scene.Frame {
	return scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: 20, Y: 30, L: 200, A: 60, Tom: 300, Nota: texto}},
	}}}
}

// comUmaNotaEm monta um Frame com um único Elemento anotado, em posição
// escolhida — é o que os casos de encaixe no Frame precisam variar.
func comUmaNotaEm(nome string, l, a int, x, y, el, ea float64, nota string) scene.Frame {
	return scene.Frame{Nome: nome, L: l, A: a, Camadas: []scene.Camada{{
		Nome:      "conteudo",
		Elementos: []scene.Elemento{{X: x, Y: y, L: el, A: ea, Tom: 300, Nota: nota}},
	}}}
}

// frameAnotado é a fixture dos goldens: quatro Notas que exercitam o layout
// inteiro numa imagem só — texto curto e texto que quebra em quatro linhas,
// balão à direita e balão à esquerda (o do Círculo não cabe à direita), e duas
// âncoras próximas o bastante para que a anti-colisão tenha de empurrar uma
// delas para baixo.
func frameAnotado() scene.Frame {
	return scene.Frame{Nome: "home", L: 400, A: 300, Camadas: []scene.Camada{
		{Nome: "fundo", Elementos: []scene.Elemento{
			{X: 20, Y: 20, L: 360, A: 50, Arredondado: true, Tom: 300, Nota: "Cabeçalho fixo"},
		}},
		{Nome: "conteudo", Elementos: []scene.Elemento{
			{X: 20, Y: 100, L: 160, A: 120, Tom: 500, Nota: notaLonga},
			{X: 40, Y: 150, L: 100, A: 40, Tom: 700, Nota: "Lista de itens"},
			{Forma: scene.Circulo, X: 300, Y: 240, L: 40, A: 40, Tom: 500, Nota: "Ação principal"},
		}},
	}}
}

// --- utilitários de imagem -------------------------------------------------

// anota roda o fluxo inteiro — Planeja, DesenhaFrame com as margens pedidas,
// Desenha — e devolve os bytes WebP.
func anota(t *testing.T, f scene.Frame, escala float64) []byte {
	t.Helper()
	plano := notes.Planeja(f, escala)
	cima, direita, baixo, esquerda := plano.Margens()
	tela := render.DesenhaFrame(f, escala, cima, direita, baixo, esquerda, -1)
	plano.Desenha(tela)
	return codifica(t, tela)
}

func desenha(t *testing.T, f scene.Frame, escala float64) image.Image {
	t.Helper()
	return decodifica(t, anota(t, f, escala))
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

// Classificadores de tinta. O fundo do balão é quase preto e o do Frame quase
// branco, então uma faixa de valores basta para distinguir o que foi pintado
// por cima sem depender de antialiasing exato.
type criterio func(uint8) bool

// claro é o texto da Nota, o extremo claro da escala, pintado sobre o balão.
func claro(v uint8) bool { return v > 0xC0 }

// medio é a linha de chamada: nem o claro do Frame nem o escuro do balão.
func medio(v uint8) bool { return v > 0x30 && v < 0xC0 }

func exatamente(v uint8) criterio {
	return func(g uint8) bool { return g == v }
}

// caixaDoBalao devolve o retângulo que contém todos os balões desenhados. O
// balão é pintado com o Tom reservado, que nenhum Elemento pode ter, então ele
// se identifica sozinho dentro do Frame.
func caixaDoBalao(img image.Image) image.Rectangle {
	return caixaDeTinta(img, img.Bounds(), exatamente(cinzaDoBalao))
}

// textoNoBalao devolve a caixa do texto de uma Nota.
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
// desenhada na região. A busca é feita dentro dos balões: fora deles o fundo
// claro do Frame se confundiria com o texto.
//
// A tinta é agrupada por proximidade vertical: linhas da mesma Nota — e até
// partes do mesmo glifo, como a haste de um "d" — ficam a poucos pixels de
// distância, enquanto Notas diferentes são separadas pelo vão entre balões,
// bem maior. Duas Notas sobrepostas viram um bloco só, que é exatamente o que
// o teste de anti-colisão procura.
func blocosDeNota(img image.Image, r image.Rectangle) []image.Rectangle {
	const separacaoEntreNotas = 8
	balao := caixaDoBalao(img)
	if balao.Empty() {
		return nil
	}
	var blocos []image.Rectangle
	for _, b := range bandasDeTinta(img, r.Intersect(balao), claro) {
		caixa := caixaDeTinta(img, b, claro)
		if n := len(blocos); n > 0 && caixa.Min.Y-blocos[n-1].Max.Y < separacaoEntreNotas {
			blocos[n-1] = blocos[n-1].Union(caixa)
			continue
		}
		blocos = append(blocos, caixa)
	}
	return blocos
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
