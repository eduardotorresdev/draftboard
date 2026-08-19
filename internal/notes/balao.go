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
// margem fazia — uma passada, sem busca global — só que agora entre duas
// colunas em vez de uma.
//
// Desce primeiro, sobe só por exceção: descer é a regra, porque uma direção
// única é o que mantém o resultado independente da ordem em que as Notas são
// atendidas dentro de um mesmo lado. Mas há um Frame em que descer não é
// opção: a barra inferior, onde todas as Notas nascem coladas na base e a tela
// vazia está toda ACIMA delas. Aí o espelho — subir até livrar — é tentado
// antes de qualquer sobreposição, no ramo de último recurso e só nele.
//
// Frame cheio demais é decidido, não evitado: quando nem descer nem subir abre
// espaço sem sair da tela, o balão volta para a altura desejada e a
// sobreposição é aceita. A alternativa seria crescer a tela, que é exatamente
// o que este plano de anotação não faz — e um Frame nessa situação tem Notas
// demais para o tamanho que declarou. O que mudou é que esse recurso passou a
// ser de fato o último: antes ele era tomado às cegas, sem consultar os balões
// já postos, e aceitava sobreposição com a tela ainda vazia.
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

		// A âncora é presa à tela antes de qualquer conta. O diagnóstico
		// deixa passar como AVISO o Elemento que cai fora do Frame, e um
		// Elemento assim pode declarar dimensão absurda: sem este limite
		// a linha de chamada sairia daí para o rasterizador, que varre
		// célula a célula e nunca volta.
		n.esquerdaDoElemento = preso(n.esquerdaDoElemento, 0, fl)
		n.direitaDoElemento = preso(n.direitaDoElemento, 0, fl)
		n.meioDoElemento = preso(n.meioDoElemento, 0, fa)

		faixaX, faixaY := naTela(n.l, fl), naTela(n.a, fa)

		// O y desejado centra o balão na âncora, já preso à tela: é dele
		// que a descida parte, nos dois lados.
		desejado := faixaY.preso(n.meioDoElemento - n.a/2)

		escolhido, achou := ladoQueCede(*n, desejado, faixaX, faixaY, postos)
		if !achou {
			escolhido = ultimoRecurso(*n, desejado, faixaX, faixaY, postos)
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

// faixa é o intervalo em que uma coordenada do bloco de texto pode cair sem
// que o balão saia da tela. min nunca é maior que max, então preso e cabe
// respondem sempre.
type faixa struct{ min, max float64 }

func (f faixa) preso(v float64) float64 { return preso(v, f.min, f.max) }

func (f faixa) cabe(v float64) bool { return v >= f.min && v <= f.max }

// naTela devolve a faixa de uma coordenada do BLOCO DE TEXTO a partir do
// tamanho do texto nesse eixo e do tamanho da tela.
//
// O limite é do BALÃO, não do bloco de texto: o balão é texto mais respiro dos
// quatro lados, e prender o bloco fazia o balão encostar em 0 e na borda
// oposta — folga zero contra a moldura, nos quatro lados.
//
// São três regimes, do desejável ao inevitável:
//
//   - com espaço de sobra, o balão ainda guarda um respiro contra a borda da
//     tela, o mesmo respiro que separa o texto do balão;
//   - num Frame apertado demais para essa folga, o balão vai até encostar na
//     borda;
//   - e quando nem o balão cabe — tela menor que o texto mais dois respiros —
//     ele é preso ao canto de origem e o texto transborda, cortado. Abaixo de
//     2*respiro + alturaDeLinha, cerca de 30 px, não existe posição que caiba:
//     a decisão é encostar no canto, e não flutuar em lugar nenhum.
func naTela(texto, tela float64) faixa {
	if max := tela - 2*respiro - texto; max >= 2*respiro {
		return faixa{2 * respiro, max}
	}
	if max := tela - respiro - texto; max >= respiro {
		return faixa{respiro, max}
	}
	return faixa{respiro, respiro}
}

// ladoQueCede escolhe entre a direita e a esquerda do Elemento o lado onde o
// balão desce menos para livrar os já postos, e devolve falso quando nenhum dos
// dois cabe inteiro na tela depois de descer.
func ladoQueCede(n nota, desejado float64, faixaX, faixaY faixa, postos []caixa) (posicao, bool) {
	var melhor posicao
	achou := false
	// A direita é avaliada primeiro para que o empate caia nela sem precisar
	// de regra própria.
	for _, cand := range []posicao{aDireita(n), aEsquerda(n)} {
		if !faixaX.cabe(cand.x) {
			continue
		}
		cand.y = desceAteLivrar(cand.x, desejado, n.l, n.a, postos)
		if !faixaY.cabe(cand.y) {
			continue
		}
		if !achou || cand.y < melhor.y {
			melhor, achou = cand, true
		}
	}
	return melhor, achou
}

// ultimoRecurso posiciona o balão quando nenhum dos dois lados abre espaço
// descendo dentro da tela — Frame estreito, onde os dois lados são descartados
// pela largura, ou âncora perto da base, onde não há para onde descer.
//
// Ele não desiste de imediato: no lado de primeira escolha, tenta a descida
// contra os balões já postos, depois o espelho, e só então aceita a
// sobreposição na altura desejada. É a diferença entre um Frame que realmente
// não cabe e um que só não cabia para baixo.
func ultimoRecurso(n nota, desejado float64, faixaX, faixaY faixa, postos []caixa) posicao {
	escolhido := primeiraEscolha(n, faixaX)
	if y := desceAteLivrar(escolhido.x, desejado, n.l, n.a, postos); faixaY.cabe(y) {
		escolhido.y = y
		return escolhido
	}
	if y := sobeAteLivrar(escolhido.x, desejado, n.l, n.a, postos); faixaY.cabe(y) {
		escolhido.y = y
		return escolhido
	}
	escolhido.y = desejado
	return escolhido
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

// sobeAteLivrar é o espelho de desceAteLivrar: levanta o bloco de texto até
// que o seu balão não cruze nenhum dos já postos.
//
// Termina pelo mesmo argumento, invertido: cada passo leva a BASE do balão
// para o topo de um dos postos, esse valor só desce, e os valores possíveis
// vêm do conjunto finito {q.y0 - espacoEntreBaloes : q ∈ postos} — logo, no
// máximo um passo por posto.
func sobeAteLivrar(x, y, l, a float64, postos []caixa) float64 {
	b := caixa{x - respiro, y - respiro, x + l + respiro, y + a + respiro}
	for mexeu := true; mexeu; {
		mexeu = false
		for _, q := range postos {
			if !b.cruza(q) {
				continue
			}
			altura := b.y1 - b.y0
			b.y1 = q.y0 - espacoEntreBaloes
			b.y0 = b.y1 - altura
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
func primeiraEscolha(n nota, faixaX faixa) posicao {
	if d := aDireita(n); faixaX.cabe(d.x) {
		return d
	}
	if e := aEsquerda(n); faixaX.cabe(e.x) {
		return e
	}
	d := aDireita(n)
	d.x = faixaX.preso(d.x)
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
