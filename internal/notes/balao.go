package notes

import (
	"math"

	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// dispoe posiciona cada Nota perto da sua âncora, sobre o desenho. A tela tem
// as dimensões do Frame e não cresce, então todo o balão é preso dentro dele.
//
// Como o balão evita cobrir o Elemento anotado: ele nunca é posicionado dentro
// da bounding box do Elemento, sempre AO LADO dela, afastado de folgaDoBalao.
// A direita é a primeira escolha, porque é onde a leitura ocidental termina —
// o olho varre o desenho e sai na anotação.
//
// Anti-colisão. Dois balões nunca se cruzam, e quem cede é sempre o que chega
// depois: as Notas são atendidas na ordem de colhe, que é função só da
// geometria e do texto, então a ordem de declaração não decide nada. Quem
// chega escolhe entre a direita e a esquerda do seu Elemento aquele lado onde
// desce menos para livrar os balões já postos; empate fica com a direita, pelo
// mesmo motivo de sempre. É a mesma resolução unidimensional que a coluna de
// margem fazia — uma passada, sempre para baixo, sem busca global — só que
// agora entre duas colunas em vez de uma.
//
// Só desce, nunca sobe: uma direção única é o que mantém o resultado
// independente da ordem em que as Notas são atendidas dentro de um mesmo lado,
// e o que torna o laço de descida obviamente finito.
//
// Frame cheio demais é decidido, não evitado: quando nenhum dos dois lados
// abre espaço sem sair da tela, o balão volta para a posição de primeira
// escolha e a sobreposição é aceita. A alternativa seria crescer a tela, que é
// exatamente o que este plano de anotação não faz — e um Frame nessa situação
// tem Notas demais para o tamanho que declarou.
//
// Texto mais largo que o Frame continua transbordando e sendo cortado: aqui não
// há margem para crescer. É o que o teto de LimiteDaNota existe para evitar,
// diagnosticando o excesso antes de desenhar.
func (p *Plano) dispoe(regua *render.Canvas, f scene.Frame) {
	fl, fa := float64(f.L), float64(f.A)

	// Num Frame estreito a largura máxima fixa não caberia; o texto quebra
	// no que houver. Isso reduz o transbordo, mas não o elimina: uma palavra
	// indivisível mais larga que o Frame continua saindo da tela, porque
	// QuebraTexto nunca parte uma palavra e aqui não há margem para crescer.
	larguraMax := math.Min(larguraMaximaDoTexto, math.Max(fl-4*respiro, corpoDaFonte))

	postos := make([]caixa, 0, len(p.notas))
	for i := range p.notas {
		n := &p.notas[i]
		p.quebra(regua, n, larguraMax)

		// O y desejado centra o balão na âncora, já preso à tela: é dele
		// que a descida parte, nos dois lados.
		desejado := preso(n.meioDoElemento-n.a/2, respiro, math.Max(respiro, fa-respiro-n.a))

		escolhido, achou := ladoQueCede(*n, desejado, fl, fa, postos)
		if !achou {
			escolhido = primeiraEscolha(*n, fl)
			escolhido.y = desejado
		}
		n.x, n.y = escolhido.x, escolhido.y
		n.ancoraX, n.chamadaX = escolhido.ancoraX, escolhido.chamadaX
		postos = append(postos, n.balao())
	}
}

// posicao é um lugar candidato para o bloco de texto de uma Nota: onde ele
// começa, de onde sai a linha de chamada e onde ela termina.
type posicao struct {
	x, y              float64
	ancoraX, chamadaX float64
}

// ladoQueCede escolhe entre a direita e a esquerda do Elemento o lado onde o
// balão desce menos para livrar os já postos, e devolve falso quando nenhum dos
// dois cabe inteiro na tela depois de descer.
func ladoQueCede(n nota, desejado, fl, fa float64, postos []caixa) (posicao, bool) {
	var melhor posicao
	achou := false
	// A direita é avaliada primeiro para que o empate caia nela sem precisar
	// de regra própria.
	for _, cand := range []posicao{aDireita(n), aEsquerda(n)} {
		if cand.x < respiro || cand.x+n.l+respiro > fl {
			continue
		}
		cand.y = desceAteLivrar(cand.x, desejado, n.l, n.a, postos)
		if cand.y+n.a+respiro > fa {
			continue
		}
		if !achou || cand.y < melhor.y {
			melhor, achou = cand, true
		}
	}
	return melhor, achou
}

// desceAteLivrar baixa o bloco de texto até que o seu balão não cruze nenhum
// dos já postos.
//
// O laço termina sempre: cada passo leva o topo do balão para a base de um dos
// postos, e como ele só sobe de valor, cada posto empurra no máximo uma vez.
func desceAteLivrar(x, y, l, a float64, postos []caixa) float64 {
	b := caixa{x - respiro, y - respiro, x + l + respiro, y + a + respiro}
	for mexeu := true; mexeu; {
		mexeu = false
		for _, q := range postos {
			if !b.cruza(q) {
				continue
			}
			altura := b.y1 - b.y0
			b.y0 = q.y1 + espacoEntreBaloes
			b.y1 = b.y0 + altura
			mexeu = true
		}
	}
	return b.y0 + respiro
}

// aDireita e aEsquerda são as duas posições possíveis do bloco de texto,
// coladas na borda correspondente do Elemento e afastadas dela por folgaDoBalao.
func aDireita(n nota) posicao {
	x := n.direitaDoElemento + folgaDoBalao + respiro
	return posicao{x: x, ancoraX: n.direitaDoElemento, chamadaX: x - folgaDaChamada}
}

func aEsquerda(n nota) posicao {
	x := n.esquerdaDoElemento - folgaDoBalao - respiro - n.l
	return posicao{x: x, ancoraX: n.esquerdaDoElemento, chamadaX: x + n.l + folgaDaChamada}
}

// primeiraEscolha é a posição que a Nota ocuparia se estivesse sozinha no
// Frame: direita, esquerda, ou — quando o Elemento é largo demais ou o Frame
// estreito demais para os dois — presa à borda, o mais perto possível de caber.
func primeiraEscolha(n nota, fl float64) posicao {
	if d := aDireita(n); d.x+n.l+respiro <= fl {
		return d
	}
	if e := aEsquerda(n); e.x >= respiro {
		return e
	}
	d := aDireita(n)
	d.x = preso(d.x, respiro, math.Max(respiro, fl-respiro-n.l))
	d.chamadaX = d.x - folgaDaChamada
	return d
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
