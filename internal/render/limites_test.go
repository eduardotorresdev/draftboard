package render

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Q1 — Texto vive no espaço da TELA INTEIRA, não no do Frame: com Chrome, a
// coordenada 5,5 cai no Chrome e não é deslocada pelas margens.
func TestTextoUsaOEspacoDaTelaInteira(t *testing.T) {
	const margem = 60

	comTexto := NewCanvas(200, 120, margem, margem, margem, margem, 1)
	comTexto.Texto(5, 5, "MMMM", 20, scene.TomFrame)
	img := decodifica(t, codifica(t, comTexto))

	// O texto tem de ter pintado dentro do Chrome, acima e à esquerda do Frame.
	pintouNoChrome := false
	for y := 0; y < margem; y++ {
		for x := 0; x < margem; x++ {
			if tomEm(img, x, y) != scene.TomChrome.Cinza() {
				pintouNoChrome = true
			}
		}
	}
	if !pintouNoChrome {
		t.Errorf("Texto(5,5) não pintou no Chrome: a coordenada foi deslocada pelas margens")
	}

	// E o Frame precisa ter ficado intocado.
	semTexto := NewCanvas(200, 120, margem, margem, margem, margem, 1)
	limpo := decodifica(t, codifica(t, semTexto))
	for y := margem; y < margem+120; y++ {
		for x := margem; x < margem+200; x++ {
			if tomEm(img, x, y) != tomEm(limpo, x, y) {
				t.Fatalf("Texto(5,5) pintou dentro do Frame em (%d,%d)", x, y)
			}
		}
	}
}

// Q2 — MedeTexto devolve a caixa de linha inteira (subida + descida). É o que
// garante que F4 empilhe Notas por altura sem elas se tocarem: entre um bloco
// com descidas e outro com subidas tem de sobrar vão em branco.
func TestMedeTextoDevolveCaixaDeLinhaInteira(t *testing.T) {
	const tamanho = 24
	const topo = 10.0

	c := NewCanvas(300, 120, 0, 0, 0, 0, 1)
	_, altura := c.MedeTexto("gggg", tamanho)

	// "gggg" só tem descidas; "HHHH" só tem subidas. Empilhados por altura,
	// eles não podem se tocar.
	c.Texto(10, topo, "gggg", tamanho, scene.TomChrome)
	c.Texto(10, topo+altura, "HHHH", tamanho, scene.TomChrome)
	img := decodifica(t, codifica(t, c))

	fundo := scene.TomFrame.Cinza()
	linhaTemTinta := func(y int) bool {
		for x := 0; x < 300; x++ {
			if tomEm(img, x, y) != fundo {
				return true
			}
		}
		return false
	}

	primeira, ultima := -1, -1
	for y := 0; y < 120; y++ {
		if linhaTemTinta(y) {
			if primeira < 0 {
				primeira = y
			}
			ultima = y
		}
	}
	if primeira < 0 {
		t.Fatalf("nada foi pintado")
	}

	// Conta o maior vão em branco entre a primeira e a última linha com tinta:
	// é a separação entre o bloco das descidas e o das subidas.
	maiorVao, vao := 0, 0
	for y := primeira; y <= ultima; y++ {
		if linhaTemTinta(y) {
			vao = 0
			continue
		}
		vao++
		if vao > maiorVao {
			maiorVao = vao
		}
	}
	if maiorVao < 2 {
		t.Errorf("vão entre os blocos = %d linhas; MedeTexto não está devolvendo "+
			"subida + descida, então as Notas de F4 se tocariam", maiorVao)
	}
}

// Q4 — o diâmetro do Círculo é o MENOR lado da bounding box, então uma caixa
// deformada continua produzindo um Círculo e nunca uma elipse.
func TestCirculoUsaOMenorLadoDaBoundingBox(t *testing.T) {
	e := scene.Elemento{
		Caminho: "deformado", Forma: scene.Circulo,
		X: 20, Y: 20, L: 80, A: 20,
		Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}
	c := NewCanvas(200, 100, 0, 0, 0, 0, 1)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	alvo := e.Tom.Cinza()
	// Linha e coluna que passam pelo centro da bounding box.
	cx, cy := 20+40, 20+10
	largura, altura := 0, 0
	for x := 0; x < 200; x++ {
		if tomEm(img, x, cy) == alvo {
			largura++
		}
	}
	for y := 0; y < 100; y++ {
		if tomEm(img, cx, y) == alvo {
			altura++
		}
	}
	if largura != altura {
		t.Errorf("bounding box 80x20 virou elipse: largura %d, altura %d", largura, altura)
	}
	// O diâmetro é o menor lado, 20 — não o maior, 80.
	if largura < 16 || largura > 20 {
		t.Errorf("diâmetro pintado = %d, esperado perto de 20 (o menor lado)", largura)
	}
}

// Q5 — a ordem real de uso de F4: DesenhaFrame primeiro, anotação depois. O
// recorte do Frame não pode sobreviver ao DesenhaElemento e engolir as Notas.
func TestAnotacaoDepoisDeDesenharOFrame(t *testing.T) {
	const margem = 40
	c := DesenhaFrame(frameExemplo(), 1, margem, margem, margem, margem, -1)

	// Tudo isto cai no Chrome, fora do retângulo do Frame.
	c.Retangulo(4, 4, 20, 10, scene.TomFrame)
	c.Linha(4, 30, 30, 30, 2, scene.TomFrame)
	c.Texto(4, 165, "Nota", 14, scene.TomFrame)
	img := decodifica(t, codifica(t, c))

	claro := scene.TomFrame.Cinza()
	if got := tomEm(img, 10, 8); got != claro {
		t.Errorf("Retangulo de anotação sumiu: pixel = %#x, quer %#x", got, claro)
	}
	if got := tomEm(img, 10, 30); got != claro {
		t.Errorf("Linha de chamada sumiu: pixel = %#x, quer %#x", got, claro)
	}

	pintouTexto := false
	for y := 165; y < 185; y++ {
		for x := 0; x < margem; x++ {
			if tomEm(img, x, y) == claro {
				pintouTexto = true
			}
		}
	}
	if !pintouTexto {
		t.Errorf("Texto da Nota sumiu do Chrome")
	}
}

// §5b — a tela satura em LimiteDeArea em vez de alocar sem teto. Conferido na
// aritmética, sem alocar: um Frame 1280x800 com --scale 100 pediria 41 GB.
func TestDimensoesSaturamNoLimiteDeArea(t *testing.T) {
	casos := []struct {
		nome            string
		largura, altura float64
		escala          float64
		saturou         bool
	}{
		{"escala 1 cabe", 1280, 800, 1, false},
		{"escala 8 cabe", 1280, 800, 8, false},
		{"escala 100 satura", 1280, 800, 100, true},
		{"escala 1000 satura", 1280, 800, 1000, true},
		{"escala 10000 satura", 1280, 800, 10000, true},
		{"escala infinita satura", 1280, 800, math.Inf(1), true},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			escala := escalaQueCabeNoTeto(caso.largura, caso.altura, caso.escala)
			l, a := dimensoesDaTela(caso.largura, caso.altura, escala)

			if l < 1 || a < 1 {
				t.Fatalf("dimensões degeneradas: %dx%d", l, a)
			}
			if l > LimiteDeArea/a {
				t.Errorf("área %d x %d = %d passa do teto %d", l, a, l*a, LimiteDeArea)
			}
			if caso.saturou {
				if escala >= caso.escala {
					t.Errorf("escala %v não foi reduzida (ficou %v)", caso.escala, escala)
				}
				// Saturou perto do teto, não muito abaixo dele.
				if l*a < LimiteDeArea/2 {
					t.Errorf("saturou em %d px, bem abaixo do teto %d", l*a, LimiteDeArea)
				}
			} else if escala != caso.escala {
				t.Errorf("escala %v foi reduzida para %v sem necessidade", caso.escala, escala)
			}

			// A proporção do Frame é preservada pela saturação.
			querProporcao := caso.largura / caso.altura
			temProporcao := float64(l) / float64(a)
			if diferencaRelativa(temProporcao, querProporcao) > 0.01 {
				t.Errorf("proporção %v, quer %v", temProporcao, querProporcao)
			}
		})
	}
}

// §5b — extensão acima de 2^25 px de dispositivo travava o rasterizador do
// freetype num laço de CPU sem fim. O recorte da bounding box fecha isso, nas
// três formas.
func TestElementoGiganteNaoTravaORasterizador(t *testing.T) {
	const gigante = 4e7 // acima de 2^25 = 33 554 432

	casos := []struct {
		nome string
		e    scene.Elemento
	}{
		{"retangulo", scene.Elemento{
			Forma: scene.Retangulo, X: -gigante / 2, Y: -gigante / 2, L: gigante, A: gigante,
			Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
		{"retangulo arredondado", scene.Elemento{
			Forma: scene.Retangulo, X: -gigante / 2, Y: -gigante / 2, L: gigante, A: gigante,
			Arredondado: true, Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
		{"circulo", scene.Elemento{
			Forma: scene.Circulo, X: -gigante / 2, Y: -gigante / 2, L: gigante, A: gigante,
			Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
		{"circulo com a borda cruzando o Frame", scene.Elemento{
			Forma: scene.Circulo, X: -gigante + 100, Y: -gigante / 2, L: gigante, A: gigante,
			Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			feito := make(chan struct{})
			go func() {
				defer close(feito)
				c := NewCanvas(200, 120, 10, 10, 10, 10, 1)
				c.DesenhaElemento(caso.e)
			}()
			select {
			case <-feito:
			case <-time.After(10 * time.Second):
				t.Fatalf("DesenhaElemento não retornou em 10s: o rasterizador travou")
			}
		})
	}
}

// O Elemento gigante ainda cobre o Frame — o recorte não pode apagá-lo.
func TestElementoGiganteAindaCobreOFrame(t *testing.T) {
	const gigante = 4e7
	const margem = 10

	casos := []scene.Forma{scene.Retangulo, scene.Circulo}
	for _, forma := range casos {
		t.Run(forma.String(), func(t *testing.T) {
			e := scene.Elemento{
				Forma: forma, X: -gigante / 2, Y: -gigante / 2, L: gigante, A: gigante,
				Elevacao: 1, Tom: scene.TomDaElevacao(1),
			}
			c := NewCanvas(200, 120, margem, margem, margem, margem, 1)
			c.DesenhaElemento(e)
			img := decodifica(t, codifica(t, c))

			if got, quer := tomEm(img, margem+100, margem+60), e.Tom.Cinza(); got != quer {
				t.Errorf("centro do Frame = %#x, quer o Tom do Elemento %#x", got, quer)
			}
			if got, quer := tomEm(img, 1, 1), scene.TomChrome.Cinza(); got != quer {
				t.Errorf("Chrome = %#x, quer %#x: o Elemento gigante vazou", got, quer)
			}
		})
	}
}

// Elemento com coordenada não-numérica não trava nem entra em pânico.
func TestElementoComCoordenadaNaoNumerica(t *testing.T) {
	naN := math.NaN()
	inf := math.Inf(1)
	casos := []scene.Elemento{
		{Forma: scene.Retangulo, X: naN, Y: 0, L: 10, A: 10},
		{Forma: scene.Retangulo, X: 0, Y: 0, L: inf, A: 10},
		{Forma: scene.Circulo, X: 0, Y: naN, L: 10, A: 10},
	}
	c := NewCanvas(50, 50, 0, 0, 0, 0, 1)
	antes := codifica(t, c)
	for _, e := range casos {
		c.DesenhaElemento(e)
	}
	if depois := codifica(t, c); len(depois) != len(antes) {
		t.Errorf("Elemento não-numérico pintou algo")
	}
}

// Q9 — MedeTexto não devolve infinito quando a escala é degenerada.
func TestMedeTextoComEscalaDegenerada(t *testing.T) {
	for _, escala := range []float64{0, -1} {
		c := NewCanvas(100, 100, 0, 0, 0, 0, escala)
		l, a := c.MedeTexto("abc", 12)
		if !finito(l) || !finito(a) {
			t.Errorf("escala %v: MedeTexto = %v,%v, quer números finitos", escala, l, a)
		}
	}
}

// Q9 — tamanho de fonte não-numérico não faz a memória de faces crescer sem
// limite nem quebra o desenho.
func TestFaceComTamanhoNaoNumerico(t *testing.T) {
	c := NewCanvas(100, 100, 0, 0, 0, 0, 1)
	for i := 0; i < 10; i++ {
		c.face(math.NaN())
	}
	if len(c.faces) != 1 {
		t.Errorf("memória de faces tem %d entradas depois de 10 tamanhos NaN, quer 1", len(c.faces))
	}
}

// Q9 / §5b — o contrato exige que o tipo Canvas documente que não é seguro
// para uso concorrente. Lemos o doc comment do próprio fonte.
func TestCanvasDocumentaQueNaoEConcorrente(t *testing.T) {
	fset := token.NewFileSet()
	arquivo, err := parser.ParseFile(fset, "canvas.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("lendo canvas.go: %v", err)
	}

	doc := ""
	ast.Inspect(arquivo, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.TYPE {
			return true
		}
		for _, spec := range decl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "Canvas" {
				doc = decl.Doc.Text()
			}
		}
		return true
	})

	if doc == "" {
		t.Fatalf("o tipo Canvas não tem doc comment")
	}
	if !strings.Contains(strings.ToLower(doc), "não é seguro para uso concorrente") {
		t.Errorf("o doc do tipo Canvas precisa dizer que não é seguro para uso "+
			"concorrente; diz apenas:\n%s", doc)
	}
}

// Q6 — com margem e escala fracionárias a fronteira do Frame cai no meio de um
// pixel. A máscara é preenchida direto no alfa, sem rasterizador, então a
// fronteira sai dura e um Elemento que transborda não deixa nenhum resíduo
// meio-tom no Chrome.
func TestRecorteComMargemFracionariaNaoVazaMeioTom(t *testing.T) {
	const margem = 12.5
	const escala = 1.5

	e := scene.Elemento{
		Caminho: "vazado", Forma: scene.Retangulo,
		X: -40, Y: -40, L: 300, A: 220,
		Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}
	c := NewCanvas(200, 120, margem, margem, margem, margem, escala)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	// A faixa do Chrome à esquerda do Frame tem de ser TomChrome puro.
	limiteE := int(math.Floor(margem * escala))
	chrome := scene.TomChrome.Cinza()
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < limiteE; x++ {
			if got := tomEm(img, x, y); got != chrome {
				t.Fatalf("resíduo no Chrome em (%d,%d): %#x, quer %#x", x, y, got, chrome)
			}
		}
	}
}

// A máscara do Frame é o que segura as formas que o recorte da bounding box não
// consegue cortar sem deformar: o Retângulo arredondado, cuja caixa é folgada
// pelo raio, e o Círculo, que é traçado inteiro. Nenhum dos dois pode pintar no
// Chrome ao transbordar a borda.
func TestFormasCurvasNaoVazamParaOChromeAoTransbordar(t *testing.T) {
	const margem = 30

	casos := []struct {
		nome string
		e    scene.Elemento
	}{
		{"retangulo arredondado saindo pela esquerda", scene.Elemento{
			Forma: scene.Retangulo, X: -60, Y: 20, L: 120, A: 80,
			Arredondado: true, Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
		{"retangulo arredondado saindo pelo topo", scene.Elemento{
			Forma: scene.Retangulo, X: 20, Y: -40, L: 120, A: 80,
			Arredondado: true, Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
		{"circulo saindo pela esquerda", scene.Elemento{
			Forma: scene.Circulo, X: -40, Y: 20, L: 80, A: 80,
			Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
		{"circulo saindo pela direita", scene.Elemento{
			Forma: scene.Circulo, X: 160, Y: 20, L: 80, A: 80,
			Elevacao: 1, Tom: scene.TomDaElevacao(1)}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := NewCanvas(200, 120, margem, margem, margem, margem, 1)
			c.DesenhaElemento(caso.e)
			img := decodifica(t, codifica(t, c))

			chrome := scene.TomChrome.Cinza()
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					noFrame := x >= margem && x < margem+200 && y >= margem && y < margem+120
					if noFrame {
						continue
					}
					if got := tomEm(img, x, y); got != chrome {
						t.Fatalf("pixel do Chrome (%d,%d) = %#x, quer %#x: a forma vazou", x, y, got, chrome)
					}
				}
			}

			// E o Elemento continua visível dentro do Frame.
			if conta(img, caso.e.Tom) == 0 {
				t.Errorf("o Elemento não aparece dentro do Frame")
			}
		})
	}
}

// O recorte do Retângulo arredondado folga a caixa pelo raio justamente para
// que os cantos criados pelo corte caiam fora do Frame. Na borda por onde o
// Elemento entra, a aresta tem de sair RETA — se a folga sumisse, apareceria um
// canto arredondado espúrio dentro do Frame.
func TestArredondadoCortadoNaoGanhaCantoEspurio(t *testing.T) {
	const margem = 30
	e := scene.Elemento{
		Forma: scene.Retangulo, X: -30, Y: 20, L: 120, A: 80,
		Arredondado: true, Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}
	c := NewCanvas(200, 120, margem, margem, margem, margem, 1)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	// Coluna colada na borda esquerda do Frame, ao longo de toda a altura do
	// Elemento: tem de estar inteiramente pintada, sem canto comido.
	x := margem
	alvo := e.Tom.Cinza()
	for dy := 0; dy < 80; dy++ {
		y := margem + 20 + dy
		if got := tomEm(img, x, y); got != alvo {
			t.Fatalf("borda de entrada em (%d,%d) = %#x, quer %#x: o corte criou "+
				"um canto arredondado dentro do Frame", x, y, got, alvo)
		}
	}

	// Já o canto direito, que é do Elemento de verdade, continua arredondado.
	if got := tomEm(img, margem+90-1, margem+20); got == alvo {
		t.Errorf("o canto direito do Elemento deixou de ser arredondado")
	}
}

// B1 — nenhuma escala degenerada pode derrubar NewCanvas. +Inf chegava a
// produzir uma tela de 9223372036854775807 px de lado e estourava o gg.
func TestNewCanvasComEscalaDegeneradaNaoEntraEmPanico(t *testing.T) {
	casos := []struct {
		nome   string
		escala float64
	}{
		{"zero", 0},
		{"negativa", -1},
		{"NaN", math.NaN()},
		{"menos infinito", math.Inf(-1)},
		{"mais infinito", math.Inf(1)},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("NewCanvas entrou em pânico: %v", r)
				}
			}()
			c := NewCanvas(1280, 800, 30, 20, 30, 20, caso.escala)

			l, a := c.dc.Width(), c.dc.Height()
			if l < 1 || a < 1 {
				t.Fatalf("tela degenerada: %dx%d", l, a)
			}
			if l > LimiteDeArea/a {
				t.Errorf("tela %dx%d = %d px passa do teto %d", l, a, l*a, LimiteDeArea)
			}

			// A tela precisa ser utilizável de verdade, não só existir.
			c.Texto(1, 1, "nota", 8, scene.TomFrame)

			// Numa tela no teto de área, desenhar Elemento (que constrói a
			// máscara) e codificar custariam dezenas de segundos sem acrescentar
			// nada: o que importa aqui é não entrar em pânico e aceitar desenho.
			if l*a <= 4<<20 {
				c.DesenhaElemento(scene.Elemento{
					Forma: scene.Retangulo, X: 10, Y: 10, L: 100, A: 50,
					Elevacao: 1, Tom: scene.TomDaElevacao(1),
				})
				if err := c.CodificaWebP(io.Discard); err != nil {
					t.Errorf("CodificaWebP: %v", err)
				}
			}
			t.Logf("escala %v -> tela %dx%d", caso.escala, l, a)
		})
	}
}

// Q-A — um Retângulo arredondado que entra pouquíssimo no Frame pela esquerda
// não pode ser esticado para dentro pelo recorte: a aresta direita fica onde o
// Elemento realmente termina.
func TestArredondadoQueMalEntraNoFrameNaoEstica(t *testing.T) {
	const margem = 30
	// Entra 1 px: vai de -199 a 1 no espaço do Frame.
	e := scene.Elemento{
		Forma: scene.Retangulo, X: -199, Y: 20, L: 200, A: 80,
		Arredondado: true, Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}
	c := NewCanvas(200, 120, margem, margem, margem, margem, 1)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	alvo := e.Tom.Cinza()
	meio := margem + 20 + 40 // meio da altura do Elemento, longe dos cantos

	// A única coluna visível é a primeira do Frame.
	if got := tomEm(img, margem, meio); got != alvo {
		t.Errorf("coluna de entrada (%d,%d) = %#x, quer o Tom do Elemento %#x",
			margem, meio, got, alvo)
	}
	// E nada além dela: o Elemento termina em x = 1 do espaço do Frame.
	for dx := 1; dx <= 8; dx++ {
		x := margem + dx
		if got := tomEm(img, x, meio); got == alvo {
			t.Fatalf("o Elemento foi esticado para dentro do Frame até (%d,%d)", x, meio)
		}
	}
}

// Q-C — o polígono do Círculo gigante tem de ter a forma do Círculo, não só
// "não travar". Comparamos com o desenho exato do gg logo acima do limite em
// que o caminho gigante entra.
func TestPoligonoDoCirculoGiganteTemAFormaDoCirculo(t *testing.T) {
	const margem = 0
	novo := func() *Canvas { return NewCanvas(200, 120, margem, margem, margem, margem, 1) }

	limite := novo().limiteDeTracado()
	// Logo acima do limite: o caminho gigante entra, e o Círculo exato ainda é
	// barato de desenhar como referência.
	r := limite * 1.05
	lado := 2 * r

	casos := []struct {
		nome string
		x, y float64
	}{
		{"borda cruzando pela esquerda", -lado + 60, -r + 60},
		{"borda cruzando pelo topo", -r + 100, -lado + 40},
		{"cobrindo o Frame", -r + 100, -r + 60},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			e := scene.Elemento{
				Forma: scene.Circulo, X: caso.x, Y: caso.y, L: lado, A: lado,
				Elevacao: 1, Tom: scene.TomDaElevacao(1),
			}

			obtido := novo()
			obtido.DesenhaElemento(e)
			imgObtido := decodifica(t, codifica(t, obtido))

			esperado := novo()
			esperado.desenhaCirculoExato(e)
			imgEsperado := decodifica(t, codifica(t, esperado))

			difere, maiorDelta := 0, 0
			b := imgObtido.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					d := int(tomEm(imgObtido, x, y)) - int(tomEm(imgEsperado, x, y))
					if d < 0 {
						d = -d
					}
					if d > 0 {
						difere++
					}
					if d > maiorDelta {
						maiorDelta = d
					}
				}
			}

			// A divergência aceitável é a coluna de antialiasing da borda.
			if maiorDelta > 12 {
				t.Errorf("o polígono não segue o Círculo: maior delta %d (limite 12), "+
					"%d px diferem", maiorDelta, difere)
			}
			if difere > b.Dx()*b.Dy()/100 {
				t.Errorf("%d px diferem de %d: mais que 1%% da tela", difere, b.Dx()*b.Dy())
			}
			t.Logf("%d px diferem, maior delta %d", difere, maiorDelta)
		})
	}
}

// desenhaCirculoExato pinta o Círculo pelo caminho exato do gg, sem o polígono.
// Serve de referência para o teste acima.
func (c *Canvas) desenhaCirculoExato(e scene.Elemento) {
	x := (c.margemE + e.X) * c.escala
	y := (c.margemT + e.Y) * c.escala
	l, a := e.L*c.escala, e.A*c.escala

	_ = c.dc.SetMask(c.mascaraDoFrame())
	c.dc.SetColor(cor(e.Tom))
	c.dc.DrawCircle(x+l/2, y+a/2, math.Min(l, a)/2)
	c.dc.Fill()
	c.dc.ResetClip()
}

// Q-D — largura máxima não-positiva não pode truncar nem entrar em laço: cada
// palavra vai para a sua linha.
func TestQuebraTextoComLarguraNaoPositiva(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	const texto = "uma frase com cinco palavras"

	for _, larguraMax := range []float64{0, -10} {
		linhas := c.QuebraTexto(texto, 12, larguraMax)
		if got, quer := len(linhas), 5; got != quer {
			t.Errorf("larguraMax %v: %d linhas, quer %d (%q)", larguraMax, got, quer, linhas)
		}
		if got, quer := strings.Join(linhas, " "), texto; got != quer {
			t.Errorf("larguraMax %v: texto truncado\n got %q\nquer %q", larguraMax, got, quer)
		}
	}
}

// B1b — dimensoesDaTela garante o teto de área por conta própria, sem depender
// de a escala já ter sido reduzida. É a segunda tranca: foi confiar numa única
// guarda que deixou +Inf chegar até a alocação.
func TestDimensoesDaTelaGarantemOTetoSozinhas(t *testing.T) {
	casos := []struct {
		nome            string
		largura, altura float64
		escala          float64
	}{
		{"escala não reduzida", 1280, 800, 100},
		{"escala absurda", 1280, 800, 1e6},
		{"escala infinita", 1280, 800, math.Inf(1)},
		{"frame já enorme", 1e6, 1e6, 1},
		{"tira larga", 1e9, 1, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			// Sem passar por escalaQueCabeNoTeto de propósito.
			l, a := dimensoesDaTela(caso.largura, caso.altura, caso.escala)

			if l < 1 || a < 1 {
				t.Fatalf("dimensões degeneradas: %dx%d", l, a)
			}
			if l > LimiteDeArea/a {
				t.Errorf("%dx%d = %d px passa do teto %d", l, a, l*a, LimiteDeArea)
			}
			t.Logf("%v x %v @ %v -> %dx%d = %d px", caso.largura, caso.altura,
				caso.escala, l, a, l*a)
		})
	}
}

// TestFonteNaoAlocaSemTeto fixa o teto de tamanho da fonte. O truetype aloca a
// máscara de um glifo proporcional ao quadrado do tamanho, então uma escala
// grande pedia uma fonte de centenas de milhares de px e derrubava o processo
// por falta de memória — antes de o teto de área chegar a recusar a tela, e
// portanto sem mensagem nenhuma. Sem o teto, este teste não falha: ele mata o
// binário de teste inteiro, que é exatamente o defeito.
func TestFonteNaoAlocaSemTeto(t *testing.T) {
	// A régua das Notas é assim: uma tela mínima na escala do desenho final.
	c := NewCanvas(1, 1, 0, 0, 0, 0, 1e5)

	var antes, depois runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&antes)

	l, a := c.MedeTexto("Mg", 11)

	runtime.ReadMemStats(&depois)
	gasto := depois.TotalAlloc - antes.TotalAlloc

	if math.IsNaN(l) || math.IsInf(l, 0) || l <= 0 {
		t.Errorf("largura medida = %v, queria um número positivo e finito", l)
	}
	if math.IsNaN(a) || math.IsInf(a, 0) || a <= 0 {
		t.Errorf("altura medida = %v, queria um número positivo e finito", a)
	}
	// A folga é larga de propósito: o que está sob teste é a ordem de grandeza,
	// não o número exato do freetype. Sem teto nenhum, medir aqui pede dezenas
	// de GB e o processo morre antes de chegar a esta linha.
	if teto := uint64(256 << 20); gasto > teto {
		t.Errorf("medir custou %d MB, queria no máximo %d MB", gasto>>20, teto>>20)
	}
}
