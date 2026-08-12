package notes

import (
	"math"

	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// disporFlutuante posiciona cada Nota perto da sua âncora, sobre o desenho. A
// tela mantém as dimensões do Frame, então as margens permanecem zeradas e todo
// o balão é preso dentro do Frame.
//
// Como o balão evita cobrir o Elemento anotado: ele nunca é posicionado dentro
// da bounding box do Elemento, sempre AO LADO dela, afastado de folgaFlutuante.
// A direita é a primeira escolha, pelo mesmo motivo do modo Margem; se o balão
// não couber à direita, ele vai para a esquerda do Elemento; se não couber de
// nenhum dos dois lados — Elemento largo demais, ou Frame estreito demais — o
// balão é preso à borda do Frame, o mais perto possível de caber.
//
// Duas consequências aceitas, porque este modo não tem margem para crescer:
//
//   - Notas com âncoras vizinhas podem se sobrepor.
//   - Texto mais largo que o Frame transborda a tela e é cortado. A garantia
//     de que a Nota cabe inteira é do modo Margem, que é o padrão e onde o
//     Chrome cresce o quanto for preciso; aqui a tela tem o tamanho do Frame e
//     ponto final.
//
// Quem precisa de qualquer uma das duas garantias usa o modo Margem.
func (p *Plano) disporFlutuante(regua *render.Canvas, f scene.Frame) {
	fl, fa := float64(f.L), float64(f.A)

	// Num Frame estreito a largura máxima fixa não caberia; o texto quebra
	// no que houver. Isso reduz o transbordo, mas não o elimina: uma palavra
	// indivisível mais larga que o Frame continua saindo da tela, porque
	// QuebraTexto nunca parte uma palavra e aqui não há margem para crescer.
	larguraMax := math.Min(larguraMaximaDoTexto, math.Max(fl-4*respiro, corpoDaFonte))

	for i := range p.notas {
		n := &p.notas[i]
		p.quebra(regua, n, larguraMax)

		direita := n.direitaDoElemento + folgaFlutuante + respiro
		esquerda := n.esquerdaDoElemento - folgaFlutuante - respiro - n.l
		switch {
		case direita+n.l+respiro <= fl:
			n.x = direita
			n.ancoraX = n.direitaDoElemento
			n.chamadaX = n.x - folgaDaChamada
		case esquerda >= respiro:
			n.x = esquerda
			n.ancoraX = n.esquerdaDoElemento
			n.chamadaX = n.x + n.l + folgaDaChamada
		default:
			n.x = preso(direita, respiro, math.Max(respiro, fl-respiro-n.l))
			n.ancoraX = n.direitaDoElemento
			n.chamadaX = n.x - folgaDaChamada
		}
		n.y = preso(n.meioDoElemento-n.a/2, respiro, math.Max(respiro, fa-respiro-n.a))
	}

	p.margemT, p.margemD, p.margemB, p.margemE = 0, 0, 0, 0
}

// preso limita v ao intervalo [min, max].
func preso(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
