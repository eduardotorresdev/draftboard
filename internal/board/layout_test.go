package board

import (
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// documento monta um Documento resolvido só com o que o layout enxerga: nome,
// dimensões e os Destinos declarados.
func documento(frames ...scene.Frame) *scene.Documento {
	return &scene.Documento{Nome: "t", Frames: frames}
}

func frame(nome string, l, a int, destinos ...string) scene.Frame {
	c := scene.Camada{Nome: "base"}
	for i, d := range destinos {
		c.Elementos = append(c.Elementos, scene.Elemento{
			Caminho: "e" + string(rune('0'+i)),
			X:       0, Y: float64(i * 10), L: 10, A: 10,
			Destino: d,
		})
	}
	return scene.Frame{Nome: nome, L: l, A: a, Camadas: []scene.Camada{c}}
}

// TestTelaDeEntradaFicaNaPrimeiraColuna: a coluna de um Frame é a distância
// dele até a entrada, e quem não recebe Ligação é entrada.
func TestTelaDeEntradaFicaNaPrimeiraColuna(t *testing.T) {
	d := documento(
		frame("login", 100, 100, "dashboard"),
		frame("dashboard", 100, 100, "detalhe"),
		frame("detalhe", 100, 100),
	)
	posicoes, ligacoes := dispoe(d)
	if len(ligacoes) != 2 {
		t.Fatalf("colheu %d Ligações, esperava 2", len(ligacoes))
	}
	if !(posicoes[0].X < posicoes[1].X && posicoes[1].X < posicoes[2].X) {
		t.Errorf("a cadeia não cresce para a direita: %v", posicoes)
	}
}

// TestLigacaoDeVoltaNaoEmpurraAEntrada protege a decisão de medir a distância
// mais curta: quase todo fluxo tem um botão que volta ao início, e medir pelo
// caminho mais longo jogaria a tela de entrada para o fim da Prancheta.
func TestLigacaoDeVoltaNaoEmpurraAEntrada(t *testing.T) {
	d := documento(
		frame("login", 100, 100, "dashboard"),
		frame("dashboard", 100, 100, "login"),
	)
	posicoes, _ := dispoe(d)
	if posicoes[0].X >= posicoes[1].X {
		t.Errorf("a tela de entrada não ficou à esquerda: %v", posicoes)
	}
}

// TestFluxoTodoEmCicloAindaTemEntrada: sem nenhum Frame livre de Ligação de
// entrada, a primeira tela declarada é a entrada — foi por ela que quem
// escreveu o Documento começou.
func TestFluxoTodoEmCicloAindaTemEntrada(t *testing.T) {
	d := documento(
		frame("a", 100, 100, "b"),
		frame("b", 100, 100, "c"),
		frame("c", 100, 100, "a"),
	)
	posicoes, _ := dispoe(d)
	if !(posicoes[0].X < posicoes[1].X && posicoes[1].X < posicoes[2].X) {
		t.Errorf("o ciclo não foi desenrolado a partir do primeiro Frame: %v", posicoes)
	}
}

// TestFrameDesligadoDoGrafoAparece: um trecho sem Ligação nenhuma com o resto
// não pode sumir da Prancheta.
func TestFrameDesligadoDoGrafoAparece(t *testing.T) {
	d := documento(
		frame("a", 100, 100, "b"),
		frame("b", 100, 100),
		frame("solto", 100, 100),
	)
	posicoes, _ := dispoe(d)
	if len(posicoes) != 3 {
		t.Fatalf("a Prancheta tem %d posições, esperava 3", len(posicoes))
	}
	conferaSemSobreposicao(t, d, posicoes)
}

// TestDocumentoSemLigacaoViraGrade: sem grafo não há coluna a derivar, e os
// Frames ainda têm de caber lado a lado sem se sobrepor.
func TestDocumentoSemLigacaoViraGrade(t *testing.T) {
	d := documento(
		frame("um", 320, 240),
		frame("dois", 320, 240),
		frame("tres", 320, 240),
		frame("quatro", 320, 240),
	)
	posicoes, ligacoes := dispoe(d)
	if len(ligacoes) != 0 {
		t.Fatalf("Documento sem `to` colheu %d Ligações", len(ligacoes))
	}
	conferaSemSobreposicao(t, d, posicoes)
	if posicoes[0].Y == posicoes[2].Y {
		t.Error("quatro Frames sem Ligação ficaram numa linha só, não numa grade")
	}
}

// TestAutoLigacaoNaoEmpurraOFrame: um Frame que aponta para si mesmo continua
// sendo tela de entrada.
func TestAutoLigacaoNaoEmpurraOFrame(t *testing.T) {
	d := documento(frame("a", 100, 100, "a"))
	posicoes, ligacoes := dispoe(d)
	if len(ligacoes) != 1 || ligacoes[0].de != ligacoes[0].para {
		t.Fatalf("a auto-Ligação não foi colhida: %v", ligacoes)
	}
	if posicoes[0].X != margemDaPrancheta {
		t.Errorf("a auto-Ligação moveu o Frame para %v", posicoes[0])
	}
}

// TestDocumentoVazioNaoEntraEmPanico: nenhum Frame é caso degenerado, não é
// erro do layout.
func TestDocumentoVazioNaoEntraEmPanico(t *testing.T) {
	posicoes, ligacoes := dispoe(documento())
	if len(posicoes) != 0 || len(ligacoes) != 0 {
		t.Errorf("Documento sem Frame produziu %v e %v", posicoes, ligacoes)
	}
}

func conferaSemSobreposicao(t *testing.T, d *scene.Documento, posicoes []posicao) {
	t.Helper()
	for i := range posicoes {
		for j := i + 1; j < len(posicoes); j++ {
			ax, ay := posicoes[i].X, posicoes[i].Y
			bx, by := posicoes[j].X, posicoes[j].Y
			al, aa := float64(d.Frames[i].L), float64(d.Frames[i].A)
			bl, ba := float64(d.Frames[j].L), float64(d.Frames[j].A)
			if ax < bx+bl && bx < ax+al && ay < by+ba && by < ay+aa {
				t.Errorf("os Frames %d e %d se sobrepõem: %v e %v", i, j, posicoes[i], posicoes[j])
			}
		}
	}
}
