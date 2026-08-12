package render

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

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// dirGolden guarda fixtures e goldens desta funcionalidade.
const dirGolden = "../../testdata/f3"

// atualizar regrava os goldens em vez de compará-los.
var atualizar = flag.Bool("update", false, "regrava os goldens de testdata/f3")

// frameExemplo monta à mão um Frame já resolvido: duas Camadas, um Retângulo
// arredondado, um Círculo e um Retângulo sobreposto na Camada de cima.
func frameExemplo() scene.Frame {
	return scene.Frame{
		Nome: "home",
		L:    200,
		A:    120,
		Camadas: []scene.Camada{
			{Nome: "conteudo", Elementos: []scene.Elemento{
				{
					Caminho: "header", Forma: scene.Retangulo,
					X: 10, Y: 10, L: 180, A: 30,
					Arredondado: true,
					Elevacao:    1, Tom: scene.TomDaElevacao(1),
				},
				{
					Caminho: "avatar", Forma: scene.Circulo,
					X: 20, Y: 60, L: 40, A: 40,
					Elevacao: 1, Tom: scene.TomDaElevacao(1),
				},
			}},
			{Nome: "modal", Elementos: []scene.Elemento{
				{
					Caminho: "modal", Forma: scene.Retangulo,
					X: 90, Y: 55, L: 90, A: 50,
					Elevacao: 2, Tom: scene.TomDaElevacao(2),
				},
			}},
		},
	}
}

// codifica devolve os bytes WebP de um Canvas.
func codifica(t *testing.T, c *Canvas) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := c.CodificaWebP(&buf); err != nil {
		t.Fatalf("CodificaWebP: %v", err)
	}
	return buf.Bytes()
}

// decodifica lê de volta os bytes WebP produzidos pelo Canvas.
func decodifica(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := webp.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("WebP inválido: %v", err)
	}
	return img
}

// tomEm devolve o valor de cinza do pixel. A imagem é escala de cinza, então
// qualquer canal serve.
func tomEm(img image.Image, x, y int) uint8 {
	r, _, _, _ := img.At(x, y).RGBA()
	return uint8(r >> 8)
}

// conta quantos pixels da imagem têm exatamente o cinza do Tom dado.
func conta(img image.Image, t scene.Tom) int {
	alvo := t.Cinza()
	b := img.Bounds()
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if tomEm(img, x, y) == alvo {
				n++
			}
		}
	}
	return n
}

// comparaGolden compara bytes com o golden de nome dado, regravando-o sob -update.
func comparaGolden(t *testing.T, nome string, got []byte) {
	t.Helper()
	caminho := filepath.Join(dirGolden, nome)
	if *atualizar {
		if err := os.MkdirAll(dirGolden, 0o755); err != nil {
			t.Fatalf("criando %s: %v", dirGolden, err)
		}
		if err := os.WriteFile(caminho, got, 0o644); err != nil {
			t.Fatalf("gravando golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo golden (rode com -update para gerar): %v", err)
	}
	if !bytes.Equal(got, want) {
		gerado := filepath.Join(t.TempDir(), nome)
		if err := os.WriteFile(gerado, got, 0o644); err != nil {
			t.Fatalf("gravando o gerado para diagnóstico: %v", err)
		}
		t.Errorf("bytes diferem do golden %s: %d bytes gerados, %d no golden\n"+
			"gerado em %s\ngolden  em %s\n"+
			"compare com: open %s %s\nse a mudança for esperada, rode: go test ./internal/render -update",
			nome, len(got), len(want), gerado, caminho, gerado, caminho)
	}
}

// A renderização precisa ser determinística: o diff em Git só é confiável se
// rodar duas vezes o mesmo Frame produzir exatamente os mesmos bytes.
func TestDeterminismoBytesIdenticos(t *testing.T) {
	f := frameExemplo()

	primeira := codifica(t, DesenhaFrame(f, 1, 0, 0, 0, 0, -1))
	segunda := codifica(t, DesenhaFrame(f, 1, 0, 0, 0, 0, -1))

	if len(primeira) != len(segunda) {
		t.Fatalf("tamanhos diferentes: %d e %d", len(primeira), len(segunda))
	}
	for i := range primeira {
		if primeira[i] != segunda[i] {
			t.Fatalf("byte %d difere entre as duas renderizações: %#x e %#x",
				i, primeira[i], segunda[i])
		}
	}
}

// O mesmo Frame com Chrome também precisa bater byte a byte com o golden
// comitado, para que uma mudança de renderização apareça na revisão.
func TestGoldenFrameComChrome(t *testing.T) {
	c := DesenhaFrame(frameExemplo(), 1, 20, 30, 20, 30, -1)
	comparaGolden(t, "home-chrome.webp", codifica(t, c))
}

// O arquivo gerado precisa ser um WebP legível e ter exatamente as dimensões
// pedidas, incluindo o Chrome.
func TestWebPValidoEDimensoes(t *testing.T) {
	c := NewCanvas(200, 120, 20, 30, 40, 50, 1)
	img := decodifica(t, codifica(t, c))

	// tela = (margemE + l + margemD) x (margemT + a + margemB)
	querL, querA := 50+200+30, 20+120+40
	if b := img.Bounds(); b.Dx() != querL || b.Dy() != querA {
		t.Errorf("Bounds() = %dx%d, quer %dx%d", b.Dx(), b.Dy(), querL, querA)
	}
}

// O Chrome usa o Tom reservado e a área do Frame o Tom de fundo fixo.
func TestChromeEFrameUsamSeusTons(t *testing.T) {
	c := NewCanvas(100, 60, 10, 10, 10, 10, 1)
	img := decodifica(t, codifica(t, c))

	if got, quer := tomEm(img, 2, 2), scene.TomChrome.Cinza(); got != quer {
		t.Errorf("pixel do Chrome = %#x, quer %#x", got, quer)
	}
	if got, quer := tomEm(img, 50, 30), scene.TomFrame.Cinza(); got != quer {
		t.Errorf("pixel do Frame = %#x, quer %#x", got, quer)
	}
}

// OrigemDoFrame é o deslocamento do Frame dentro da tela: margem esquerda e
// margem superior.
func TestOrigemDoFrame(t *testing.T) {
	c := NewCanvas(100, 60, 11, 22, 33, 44, 2)
	x, y := c.OrigemDoFrame()
	if x != 44 || y != 11 {
		t.Errorf("OrigemDoFrame() = %v,%v, quer 44,11", x, y)
	}
}

// O fator de escala multiplica as dimensões finais, e o produto é arredondado
// para o inteiro mais próximo — inclusive em fatores não-inteiros.
func TestEscalaProduzDimensoesProporcionais(t *testing.T) {
	casos := []struct {
		escala     float64
		querL      int
		querA      int
		nomeDoCaso string
	}{
		{1, 200, 120, "identidade"},
		{2, 400, 240, "inteiro"},
		{1.5, 300, 180, "meio"},
		{2.5, 500, 300, "dois e meio"},
		{0.5, 100, 60, "reducao"},
		{1.337, 267, 160, "arbitrario"},          // 267.4 -> 267 ; 160.44 -> 160
		{1.333, 267, 160, "arredonda para cima"}, // 266.6 -> 267 ; 159.96 -> 160
	}
	for _, caso := range casos {
		t.Run(caso.nomeDoCaso, func(t *testing.T) {
			c := NewCanvas(200, 120, 0, 0, 0, 0, caso.escala)
			img := decodifica(t, codifica(t, c))
			if b := img.Bounds(); b.Dx() != caso.querL || b.Dy() != caso.querA {
				t.Errorf("escala %v: Bounds() = %dx%d, quer %dx%d",
					caso.escala, b.Dx(), b.Dy(), caso.querL, caso.querA)
			}
		})
	}
}

// A escala também vale para o Chrome: margens fracionárias entram no produto.
func TestEscalaIncluiOChrome(t *testing.T) {
	c := NewCanvas(100, 50, 10, 10, 10, 10, 1.5)
	img := decodifica(t, codifica(t, c))
	// (10 + 100 + 10) * 1.5 = 180 ; (10 + 50 + 10) * 1.5 = 105
	if b := img.Bounds(); b.Dx() != 180 || b.Dy() != 105 {
		t.Errorf("Bounds() = %dx%d, quer 180x105", b.Dx(), b.Dy())
	}
}

// Com Arredondado ligado o canto do Retângulo fica vazado; desligado, o canto
// é pintado até o extremo.
func TestCantoArredondadoLigadoEDesligado(t *testing.T) {
	base := scene.Elemento{
		Caminho: "caixa", Forma: scene.Retangulo,
		X: 20, Y: 20, L: 100, A: 60,
		Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}

	reto := base
	cReto := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	cReto.DesenhaElemento(reto)
	imgReto := decodifica(t, codifica(t, cReto))

	redondo := base
	redondo.Arredondado = true
	cRedondo := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	cRedondo.DesenhaElemento(redondo)
	imgRedondo := decodifica(t, codifica(t, cRedondo))

	// O pixel do canto superior esquerdo do Elemento.
	if got, quer := tomEm(imgReto, 20, 20), base.Tom.Cinza(); got != quer {
		t.Errorf("canto reto = %#x, quer o Tom do Elemento %#x", got, quer)
	}
	if got, naoQuer := tomEm(imgRedondo, 20, 20), base.Tom.Cinza(); got == naoQuer {
		t.Errorf("canto arredondado = %#x, não deveria ser o Tom do Elemento", got)
	}
	if got, quer := tomEm(imgRedondo, 20, 20), scene.TomFrame.Cinza(); got != quer {
		t.Errorf("canto arredondado = %#x, quer o Tom do Frame %#x", got, quer)
	}

	// O centro é sólido nos dois casos: só o canto muda.
	if got, quer := tomEm(imgRedondo, 70, 50), base.Tom.Cinza(); got != quer {
		t.Errorf("centro do Retângulo arredondado = %#x, quer %#x", got, quer)
	}
}

// O raio é constante em toda a tela e limitado a um quarto do menor lado, para
// que um Retângulo pequeno não vire pílula.
func TestRaioConstanteELimitadoNoElementoPequeno(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)

	grande := scene.Elemento{Forma: scene.Retangulo, L: 100, A: 60, Arredondado: true}
	if got := c.raio(grande); got != raioBase {
		t.Errorf("raio do Elemento grande = %v, quer o raio base %v", got, raioBase)
	}

	// Menor lado 16 => limite 16 * 0.25 = 4, abaixo do raio base.
	pequeno := scene.Elemento{Forma: scene.Retangulo, L: 40, A: 16, Arredondado: true}
	if got, quer := c.raio(pequeno), 4.0; got != quer {
		t.Errorf("raio do Elemento pequeno = %v, quer %v", got, quer)
	}

	// Nunca chega à metade do menor lado, que é onde a pílula começa.
	if got, pilula := c.raio(pequeno), 16.0/2; got >= pilula {
		t.Errorf("raio %v chegou ao raio de pílula %v", got, pilula)
	}

	// O raio escala junto com o fator de escala.
	c2 := NewCanvas(200, 120, 0, 0, 0, 0, 2)
	if got, quer := c2.raio(grande), raioBase*2; got != quer {
		t.Errorf("raio na escala 2 = %v, quer %v", got, quer)
	}
}

// Prova em pixels que o Retângulo pequeno é arredondado mas não vira pílula.
func TestElementoPequenoNaoViraPilula(t *testing.T) {
	e := scene.Elemento{
		Caminho: "chip", Forma: scene.Retangulo,
		X: 10, Y: 10, L: 16, A: 16,
		Arredondado: true,
		Elevacao:    1, Tom: scene.TomDaElevacao(1),
	}
	c := NewCanvas(60, 60, 0, 0, 0, 0, 1)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	// Com raio 4 o pixel do canto exato fica de fora: o canto é arredondado.
	if got, quer := tomEm(img, 10, 10), scene.TomFrame.Cinza(); got != quer {
		t.Errorf("canto (10,10) = %#x, quer o Tom do Frame %#x", got, quer)
	}
	// Com raio 4 o pixel (12,12) já está inteiramente dentro; com raio 8 — a
	// pílula de um Elemento 16x16 — ele ficaria na borda serrilhada.
	if got, quer := tomEm(img, 12, 12), e.Tom.Cinza(); got != quer {
		t.Errorf("pixel (12,12) = %#x, quer o Tom do Elemento %#x; o raio passou do limite", got, quer)
	}
}

// Elemento que ultrapassa a borda do Frame é cortado e nunca invade o Chrome.
func TestRecorteNaBordaNaoInvadeOChrome(t *testing.T) {
	// O Retângulo começa dentro do Frame e transborda os quatro lados.
	e := scene.Elemento{
		Caminho: "vazado", Forma: scene.Retangulo,
		X: -30, Y: -30, L: 260, A: 180,
		Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}
	const margem = 20
	c := NewCanvas(200, 120, margem, margem, margem, margem, 1)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	// Todo o Chrome continua com o Tom reservado.
	b := img.Bounds()
	chrome := scene.TomChrome.Cinza()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			noFrame := x >= margem && x < margem+200 && y >= margem && y < margem+120
			if noFrame {
				continue
			}
			if got := tomEm(img, x, y); got != chrome {
				t.Fatalf("pixel do Chrome (%d,%d) = %#x, quer %#x: o Elemento vazou", x, y, got, chrome)
			}
		}
	}

	// E o Frame inteiro foi coberto pelo Elemento.
	if got, quer := tomEm(img, margem, margem), e.Tom.Cinza(); got != quer {
		t.Errorf("canto do Frame = %#x, quer o Tom do Elemento %#x", got, quer)
	}
	if got, quer := tomEm(img, margem+199, margem+119), e.Tom.Cinza(); got != quer {
		t.Errorf("canto oposto do Frame = %#x, quer o Tom do Elemento %#x", got, quer)
	}
}

// Círculo continua redondo num Frame não-quadrado: a largura e a altura
// pintadas são iguais.
func TestCirculoRedondoEmFrameNaoQuadrado(t *testing.T) {
	e := scene.Elemento{
		Caminho: "avatar", Forma: scene.Circulo,
		X: 50, Y: 20, L: 40, A: 40,
		Elevacao: 1, Tom: scene.TomDaElevacao(1),
	}
	// Frame bem mais largo que alto.
	c := NewCanvas(400, 100, 0, 0, 0, 0, 1)
	c.DesenhaElemento(e)
	img := decodifica(t, codifica(t, c))

	alvo := e.Tom.Cinza()
	// Largura pintada na linha central do Círculo.
	cy := 20 + 20
	larguraPintada := 0
	for x := 0; x < 400; x++ {
		if tomEm(img, x, cy) == alvo {
			larguraPintada++
		}
	}
	// Altura pintada na coluna central do Círculo.
	cx := 50 + 20
	alturaPintada := 0
	for y := 0; y < 100; y++ {
		if tomEm(img, cx, y) == alvo {
			alturaPintada++
		}
	}
	if larguraPintada != alturaPintada {
		t.Errorf("Círculo virou elipse: largura %d, altura %d", larguraPintada, alturaPintada)
	}
	if larguraPintada < 36 || larguraPintada > 40 {
		t.Errorf("diâmetro pintado = %d, esperado perto de 40", larguraPintada)
	}
	// Fora do Círculo, dentro da bounding box, o fundo do Frame aparece.
	if got, quer := tomEm(img, 50, 20), scene.TomFrame.Cinza(); got != quer {
		t.Errorf("canto da bounding box = %#x, quer o Tom do Frame %#x", got, quer)
	}
}

// O export por Camada é cumulativo: ateCamada crescente inclui progressivamente
// mais Elementos, e o resultado final é igual a pedir todas.
func TestExportPorCamadaCumulativo(t *testing.T) {
	f := frameExemplo()

	pintados := func(ate int) int {
		c := DesenhaFrame(f, 1, 0, 0, 0, 0, ate)
		img := decodifica(t, codifica(t, c))
		// Tudo que não é o fundo do Frame foi pintado por algum Elemento.
		b := img.Bounds()
		fundo := scene.TomFrame.Cinza()
		n := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if tomEm(img, x, y) != fundo {
					n++
				}
			}
		}
		return n
	}

	so0 := pintados(0)
	ate1 := pintados(1)
	todas := pintados(-1)

	if so0 <= 0 {
		t.Fatalf("Camada 0 não pintou nada")
	}
	if ate1 <= so0 {
		t.Errorf("Camada 1 não acrescentou Elementos: %d depois de %d", ate1, so0)
	}
	if todas != ate1 {
		t.Errorf("ateCamada < 0 = %d pixels, quer o mesmo que a última Camada %d", todas, ate1)
	}

	// A Camada de cima permanece por cima: o Tom do modal aparece na imagem
	// cumulativa.
	c := DesenhaFrame(f, 1, 0, 0, 0, 0, -1)
	img := decodifica(t, codifica(t, c))
	if conta(img, scene.TomDaElevacao(2)) == 0 {
		t.Errorf("o Elemento da Camada de cima não aparece no render completo")
	}
}

// ateCamada além do número de Camadas não estoura e vale como todas.
func TestAteCamadaAlemDoFimValeTodas(t *testing.T) {
	f := frameExemplo()
	demais := codifica(t, DesenhaFrame(f, 1, 0, 0, 0, 0, 99))
	todas := codifica(t, DesenhaFrame(f, 1, 0, 0, 0, 0, -1))
	if !bytes.Equal(demais, todas) {
		t.Errorf("ateCamada = 99 não bateu com ateCamada = -1")
	}
}

// Elemento de área zero não pinta nada.
func TestElementoDeAreaZeroNaoPinta(t *testing.T) {
	c := NewCanvas(50, 50, 0, 0, 0, 0, 1)
	antes := codifica(t, c)

	c2 := NewCanvas(50, 50, 0, 0, 0, 0, 1)
	c2.DesenhaElemento(scene.Elemento{Forma: scene.Retangulo, X: 10, Y: 10, L: 0, A: 20, Tom: scene.TomDaElevacao(1)})
	depois := codifica(t, c2)

	if !bytes.Equal(antes, depois) {
		t.Errorf("Elemento de área zero mudou a imagem")
	}
}

// As primitivas de anotação alcançam o Chrome: as coordenadas são da tela
// inteira, não do Frame.
func TestPrimitivasDeAnotacaoAlcancamOChrome(t *testing.T) {
	c := NewCanvas(100, 60, 30, 30, 30, 30, 1)
	c.Retangulo(2, 2, 20, 20, scene.TomFrame)
	c.Linha(0, 50, 160, 50, 2, scene.TomFrame)
	img := decodifica(t, codifica(t, c))

	if got, quer := tomEm(img, 10, 10), scene.TomFrame.Cinza(); got != quer {
		t.Errorf("Retangulo no Chrome = %#x, quer %#x", got, quer)
	}
	if got, quer := tomEm(img, 5, 50), scene.TomFrame.Cinza(); got != quer {
		t.Errorf("Linha no Chrome = %#x, quer %#x", got, quer)
	}
}

// QuebraTexto quebra em várias linhas, quebra só entre palavras e nunca perde
// caracteres.
func TestQuebraTextoEmVariasLinhasSemTruncar(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	const texto = "Cabeçalho fixo com ações de navegação e o avatar do usuário logado"

	linhas := c.QuebraTexto(texto, 12, 80)
	if len(linhas) < 3 {
		t.Fatalf("QuebraTexto devolveu %d linha(s), esperado várias: %q", len(linhas), linhas)
	}

	// Nenhuma palavra se perdeu nem foi cortada.
	if got, quer := strings.Join(linhas, " "), strings.Join(strings.Fields(texto), " "); got != quer {
		t.Errorf("QuebraTexto truncou ou reordenou:\n got %q\nquer %q", got, quer)
	}

	// Cada linha cabe na largura pedida, salvo linha de uma palavra só.
	for _, linha := range linhas {
		l, _ := c.MedeTexto(linha, 12)
		if l > 80 && len(strings.Fields(linha)) > 1 {
			t.Errorf("linha %q mede %v, passa da largura máxima 80", linha, l)
		}
	}
}

// Uma palavra maior que a largura máxima fica inteira na sua linha em vez de
// ser truncada.
func TestQuebraTextoNaoTruncaPalavraLonga(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	const palavra = "responsabilidadessobrepostas"

	linhas := c.QuebraTexto("ok "+palavra+" fim", 12, 20)
	achou := false
	for _, linha := range linhas {
		if linha == palavra {
			achou = true
		}
	}
	if !achou {
		t.Errorf("a palavra longa foi truncada ou quebrada: %q", linhas)
	}
}

// QuebraTexto preserva as quebras de linha explícitas do texto.
func TestQuebraTextoPreservaQuebrasExplicitas(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	linhas := c.QuebraTexto("primeira\nsegunda", 12, 500)
	if len(linhas) != 2 || linhas[0] != "primeira" || linhas[1] != "segunda" {
		t.Errorf("QuebraTexto = %q, quer [primeira segunda]", linhas)
	}
}

// Texto vazio não produz linha nenhuma.
func TestQuebraTextoVazio(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)
	if linhas := c.QuebraTexto("", 12, 100); len(linhas) != 0 {
		t.Errorf("QuebraTexto(\"\") = %q, quer vazio", linhas)
	}
}

// MedeTexto cresce com o texto e com o tamanho, e devolve px do espaço do
// Frame — independentes da escala.
func TestMedeTexto(t *testing.T) {
	c := NewCanvas(200, 120, 0, 0, 0, 0, 1)

	lCurto, aCurto := c.MedeTexto("ab", 12)
	lLongo, aLongo := c.MedeTexto("abcdefgh", 12)
	if lLongo <= lCurto {
		t.Errorf("texto mais longo mediu %v, não é maior que %v", lLongo, lCurto)
	}
	if aCurto != aLongo {
		t.Errorf("alturas de linha diferentes no mesmo tamanho: %v e %v", aCurto, aLongo)
	}
	if aCurto <= 0 {
		t.Errorf("altura de linha = %v, quer positiva", aCurto)
	}

	_, aGrande := c.MedeTexto("ab", 24)
	if aGrande <= aCurto {
		t.Errorf("altura no tamanho 24 = %v, não é maior que no 12 = %v", aGrande, aCurto)
	}

	// Na escala 2 o texto ocupa o mesmo espaço no espaço do Frame.
	c2 := NewCanvas(200, 120, 0, 0, 0, 0, 2)
	l2, a2 := c2.MedeTexto("abcdefgh", 12)
	if diferencaRelativa(l2, lLongo) > 0.05 {
		t.Errorf("largura na escala 2 = %v, quer perto de %v", l2, lLongo)
	}
	if diferencaRelativa(a2, aLongo) > 0.05 {
		t.Errorf("altura na escala 2 = %v, quer perto de %v", a2, aLongo)
	}
}

// Texto trata y como o TOPO da linha, não como a linha de base. A âncora é
// verificada de forma exata: "H" se apoia na linha de base, então a última
// linha de pixels com tinta é a imediatamente acima dela, e a linha de base
// fica em y + subida. Um deslocamento de um único pixel quebra este teste — é
// o que garante que F4 possa empilhar Notas por altura.
func TestTextoTrataYComoTopoDaLinha(t *testing.T) {
	const topo = 40

	for _, tamanho := range []float64{9, 12, 16, 24, 48} {
		c := NewCanvas(300, 140, 0, 0, 0, 0, 1)
		c.Texto(10, topo, "H", tamanho, scene.TomChrome)
		img := decodifica(t, codifica(t, c))

		fundo := scene.TomFrame.Cinza()
		primeira, ultima := -1, -1
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if tomEm(img, x, y) != fundo {
					if primeira < 0 {
						primeira = y
					}
					ultima = y
					break
				}
			}
		}
		if primeira < 0 {
			t.Fatalf("tamanho %v: Texto não pintou nada", tamanho)
		}

		// Nada pode ser pintado acima do topo declarado.
		if primeira < topo {
			t.Errorf("tamanho %v: Texto pintou na linha %d, acima do topo %d",
				tamanho, primeira, topo)
		}

		// E a linha de base tem de cair exatamente em topo + subida.
		subida, altura := metricas(c.face(tamanho))
		querUltima := int(math.Round(topo+subida)) - 1
		if ultima != querUltima {
			t.Errorf("tamanho %v: última linha com tinta = %d, quer %d; "+
				"a linha de base não está em topo + subida (%.2f)",
				tamanho, ultima, querUltima, topo+subida)
		}

		// E a tinta toda cabe na caixa de linha devolvida por MedeTexto.
		if float64(ultima) > topo+altura {
			t.Errorf("tamanho %v: tinta até %d passa da caixa de linha que "+
				"termina em %.2f", tamanho, ultima, topo+altura)
		}
	}
}

// Texto vazio não pinta nada.
func TestTextoVazioNaoPinta(t *testing.T) {
	c := NewCanvas(100, 50, 0, 0, 0, 0, 1)
	antes := codifica(t, c)

	c2 := NewCanvas(100, 50, 0, 0, 0, 0, 1)
	c2.Texto(10, 10, "", 12, scene.TomChrome)
	if !bytes.Equal(antes, codifica(t, c2)) {
		t.Errorf("Texto vazio mudou a imagem")
	}
}

// CodificaWebP propaga erro de escrita em vez de relatar sucesso.
func TestCodificaWebPPropagaErroDeEscrita(t *testing.T) {
	c := NewCanvas(20, 20, 0, 0, 0, 0, 1)
	if err := c.CodificaWebP(escritorQuebrado{}); err == nil {
		t.Errorf("CodificaWebP não relatou o erro de escrita")
	}
}

type escritorQuebrado struct{}

func (escritorQuebrado) Write([]byte) (int, error) {
	return 0, os.ErrClosed
}

func diferencaRelativa(a, b float64) float64 {
	if b == 0 {
		return a
	}
	d := (a - b) / b
	if d < 0 {
		return -d
	}
	return d
}
