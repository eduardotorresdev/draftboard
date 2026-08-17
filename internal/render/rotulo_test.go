package render

import (
	"image"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Geometria do Frame de teste, em px do espaço do Frame.
const (
	botaoX, botaoY, botaoL, botaoA     = 20, 10, 360, 100
	rotuloX, rotuloY, rotuloL, rotuloA = 40, 40, 320, 40
	frameL, frameA                     = 400, 120
)

// frameComRotulo monta um Frame de um Retângulo claro com um Rótulo escuro
// dentro, que é a forma como um Controle chega ao rasterizador depois da
// resolução: a peça de Forma Texto traz a área que lhe coube, nunca a caixa das
// glifas.
func frameComRotulo(texto string, alinhamento scene.Alinhamento) scene.Frame {
	return scene.Frame{
		Nome: "rotulo", L: frameL, A: frameA,
		Camadas: []scene.Camada{{Nome: "c", Elementos: []scene.Elemento{
			{
				Caminho: "botao", Forma: scene.Retangulo,
				X: botaoX, Y: botaoY, L: botaoL, A: botaoA,
				Elevacao: 1, Tom: scene.TomDaElevacao(1),
			},
			{
				Caminho: "botao/rotulo", Forma: scene.Texto,
				X: rotuloX, Y: rotuloY, L: rotuloL, A: rotuloA,
				Rotulo: texto, Alinhamento: alinhamento,
				Elevacao: 2, Tom: scene.TomDaElevacao(2),
			},
		}}},
	}
}

// desenhaSemChrome rasteriza o Frame sem margens, para que todo pixel da tela
// seja Frame e o teste possa afirmar sobre a ausência de tinta fora da área.
func desenhaSemChrome(t *testing.T, f scene.Frame) image.Image {
	t.Helper()
	return decodifica(t, codifica(t, DesenhaFrame(f, 1, 0, 0, 0, 0, -1)))
}

// tomDaSuperficie é o cinza do Retângulo que sustenta o Rótulo. Todo pixel mais
// escuro que ele, dentro do Retângulo, é tinta do texto — a asserção é sobre
// "há tinta", e não sobre o Tom exato, porque a borda do glifo é uma mistura
// entre o Tom do texto e o da Superfície.
func tomDaSuperficie() uint8 { return scene.TomDaElevacao(1).Cinza() }

// TestRotuloPintaDentroDaSuaArea prova que o Rótulo vira tinta na imagem.
func TestRotuloPintaDentroDaSuaArea(t *testing.T) {
	img := desenhaSemChrome(t, frameComRotulo("Salvar", scene.AoCentro))

	if n := tintaEm(img, rotuloX, rotuloY, rotuloL, rotuloA); n == 0 {
		t.Fatal("nenhuma tinta dentro da área do Rótulo: o texto não foi desenhado")
	}
}

// TestRotuloVazioNaoPintaNada protege o caso do Controle sem label: a peça
// existe na cena, mas não pode deixar tinta nenhuma.
func TestRotuloVazioNaoPintaNada(t *testing.T) {
	img := desenhaSemChrome(t, frameComRotulo("", scene.AoCentro))

	if n := tintaEm(img, botaoX, botaoY, botaoL, botaoA); n != 0 {
		t.Errorf("pixels de tinta = %d, quer 0 num Rótulo vazio", n)
	}
}

// TestRotuloLongoNaoVazaDaArea é a regra de recorte: um texto mais largo que a
// área que lhe coube é cortado nela. Sem isso o Rótulo de um Controle estreito
// atravessaria a Superfície vizinha e o wireframe mentiria sobre o espaço que
// o conteúdo ocupa.
func TestRotuloLongoNaoVazaDaArea(t *testing.T) {
	longo := "Um rotulo absurdamente comprido que nao cabe de jeito nenhum aqui dentro"
	img := desenhaSemChrome(t, frameComRotulo(longo, scene.AEsquerda))

	if n := tintaEm(img, rotuloX, rotuloY, rotuloL, rotuloA); n == 0 {
		t.Fatal("o Rótulo longo não pintou nada dentro da própria área")
	}

	casos := []struct {
		nome       string
		x, y, l, a int
	}{
		{"à direita da área", rotuloX + rotuloL, botaoY, botaoX + botaoL - (rotuloX + rotuloL), botaoA},
		{"à esquerda da área", botaoX, botaoY, rotuloX - botaoX, botaoA},
		{"acima da área", botaoX, botaoY, botaoL, rotuloY - botaoY},
		{"abaixo da área", botaoX, rotuloY + rotuloA, botaoL, botaoY + botaoA - (rotuloY + rotuloA)},
	}
	for _, caso := range casos {
		if n := tintaEm(img, caso.x, caso.y, caso.l, caso.a); n != 0 {
			t.Errorf("tinta %s = %d, quer 0", caso.nome, n)
		}
	}
}

// TestRotuloNaoInvadeOChrome fecha a mesma regra pela outra borda: um Rótulo
// que começa dentro do Frame e se estende para fora dele é cortado na borda, e
// o Chrome fica intocado.
func TestRotuloNaoInvadeOChrome(t *testing.T) {
	f := frameComRotulo("Salvar em algum lugar bem longe", scene.AEsquerda)
	f.Camadas[0].Elementos[1].X = frameL - 40
	f.Camadas[0].Elementos[1].L = 300

	const margemE, margemD, margemT, margemB = 10, 40, 10, 10
	img := decodifica(t, codifica(t, DesenhaFrame(f, 1, margemT, margemD, margemB, margemE, -1)))

	chrome := scene.TomChrome.Cinza()
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := margemE + frameL; x < b.Max.X; x++ {
			if tomEm(img, x, y) != chrome {
				t.Fatalf("pixel %d,%d do Chrome = %#x, quer %#x: o Rótulo vazou do Frame", x, y, tomEm(img, x, y), chrome)
			}
		}
	}
}

// TestRotuloAlinhaConformeOControle prova que o alinhamento vem da cena: o
// mesmo texto na mesma área começa mais à esquerda quando alinhado à esquerda
// do que quando centralizado.
func TestRotuloAlinhaConformeOControle(t *testing.T) {
	esquerda := primeiraColunaComTinta(desenhaSemChrome(t, frameComRotulo("Ok", scene.AEsquerda)))
	centro := primeiraColunaComTinta(desenhaSemChrome(t, frameComRotulo("Ok", scene.AoCentro)))

	if esquerda < 0 || centro < 0 {
		t.Fatalf("Rótulo não pintou: esquerda=%d centro=%d", esquerda, centro)
	}
	if esquerda >= centro {
		t.Errorf("primeira coluna com tinta: esquerda=%d, centro=%d; queria esquerda menor", esquerda, centro)
	}
}

// tintaEm conta os pixels mais escuros que a Superfície dentro do retângulo.
func tintaEm(img image.Image, x, y, l, a int) int {
	b := img.Bounds()
	fundo := tomDaSuperficie()
	n := 0
	for py := y; py < y+a; py++ {
		for px := x; px < x+l; px++ {
			if !(image.Point{X: px, Y: py}).In(b) {
				continue
			}
			if tomEm(img, px, py) < fundo {
				n++
			}
		}
	}
	return n
}

// primeiraColunaComTinta devolve a menor coluna com tinta de Rótulo, ou -1
// quando não há nenhuma.
func primeiraColunaComTinta(img image.Image) int {
	b := img.Bounds()
	fundo := tomDaSuperficie()
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if tomEm(img, x, y) < fundo {
				return x
			}
		}
	}
	return -1
}
