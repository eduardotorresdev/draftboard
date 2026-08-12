package notes

import (
	"math"

	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// disporNaMargem empilha as Notas numa única coluna de Chrome à DIREITA do
// Frame e devolve, pelas margens do Plano, o Chrome que a tela precisa ter.
//
// Um lado só, e não os dois: com uma coluna a resolução de colisão é
// literalmente unidimensional — ordena por altura da âncora e empilha — o que
// mantém o algoritmo simples e, principalmente, o resultado estável entre
// edições. Repartir as Notas entre dois lados exigiria uma decisão global de
// atribuição, e qualquer Nota nova poderia migrar de lado e reembaralhar tudo.
// A direita, porque é onde a leitura ocidental termina: o olho varre o desenho
// e sai na anotação.
//
// A margem cresce o quanto for preciso: em largura, para caber a linha mais
// larga (uma palavra sozinha maior que a largura máxima transborda a quebra, e
// a faixa acompanha em vez de truncar); em altura, para cima e para baixo,
// quando a pilha ultrapassa as bordas do Frame.
func (p *Plano) disporNaMargem(regua *render.Canvas, f scene.Frame) {
	colunaX := float64(f.L) + respiro

	larguraDaColuna := 0.0
	for i := range p.notas {
		p.quebra(regua, &p.notas[i], larguraMaximaDoTexto)
		if l := p.notas[i].l; l > larguraDaColuna {
			larguraDaColuna = l
		}
	}

	// Empilhamento unidimensional: cada Nota quer ficar centrada na altura
	// da sua âncora e só desce o mínimo necessário para não encostar na
	// anterior. Como as Notas já vêm ordenadas por altura de âncora, uma
	// única passada resolve todas as colisões, sem busca global.
	cursor := math.Inf(-1)
	menorTopo, maiorFundo := math.Inf(1), math.Inf(-1)
	for i := range p.notas {
		n := &p.notas[i]
		topo := n.meioDoElemento - n.a/2
		if topo < cursor {
			topo = cursor
		}
		n.x, n.y = colunaX, topo
		n.ancoraX = n.direitaDoElemento
		n.chamadaX = colunaX - folgaDaChamada
		cursor = topo + n.a + espacoEntreNotas

		menorTopo = math.Min(menorTopo, topo)
		maiorFundo = math.Max(maiorFundo, topo+n.a)
	}

	p.margemD = math.Max(2*respiro+larguraDaColuna, larguraMinimaDaFaixa)
	p.margemT = math.Max(0, respiro-menorTopo)
	p.margemB = math.Max(0, maiorFundo+respiro-float64(f.A))
	p.margemE = 0
}
