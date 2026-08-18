package board

import (
	"math"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

const (
	// intervaloH e intervaloV são o respiro entre Frames vizinhos na
	// Prancheta, em px do espaço do Frame. São largos de propósito: é neles
	// que as Ligações têm espaço para curvar.
	intervaloH = 240.0
	intervaloV = 160.0
	// alturaDoTitulo é a faixa reservada acima de cada Frame para o nome e as
	// dimensões. Faz parte da Prancheta, nunca do Frame.
	alturaDoTitulo = 40.0
	// margemDaPrancheta é a folga entre o conteúdo e a borda do mundo.
	margemDaPrancheta = 120.0
)

// posicao é o canto superior esquerdo de um Frame na Prancheta.
type posicao struct {
	X, Y float64
}

// ligacao é uma Ligação já resolvida em índices de Frame, com a geometria do
// Elemento gatilho relativa ao Frame de origem.
type ligacao struct {
	de, para   int
	caminho    string
	nome       string
	x, y, l, a float64
}

// dispoe calcula a posição de cada Frame e colhe as Ligações do Documento.
//
// A disposição é derivada do grafo de Ligações e nunca declarada: a coluna de um
// Frame é a distância dele até uma tela de entrada, medida pelo caminho mais
// longo. Um Documento sem Ligação nenhuma não tem grafo, e cai numa grade.
func dispoe(d *scene.Documento) ([]posicao, []ligacao) {
	n := len(d.Frames)
	posicoes := make([]posicao, n)
	if n == 0 {
		return posicoes, nil
	}

	indiceDoFrame := make(map[string]int, n)
	for i, f := range d.Frames {
		// Nomes repetidos são resolvidos pelo primeiro: a Ligação cita um
		// nome, e o primeiro é o único desempate estável.
		if _, ok := indiceDoFrame[f.Nome]; !ok {
			indiceDoFrame[f.Nome] = i
		}
	}

	var ligacoes []ligacao
	for i, f := range d.Frames {
		for _, c := range f.Camadas {
			for _, e := range c.Elementos {
				if e.Destino == "" {
					continue
				}
				alvo, ok := indiceDoFrame[e.Destino]
				if !ok {
					// A resolução já recusou destino desconhecido. Chegar aqui
					// significa Documento montado à mão: ignora em vez de
					// desenhar uma seta para lugar nenhum.
					continue
				}
				ligacoes = append(ligacoes, ligacao{
					de: i, para: alvo, caminho: e.Caminho, nome: e.Destino,
					x: e.X, y: e.Y, l: e.L, a: e.A,
				})
			}
		}
	}

	colunas := colunasDoGrafo(n, ligacoes)
	posiciona(d, colunas, posicoes)
	return posicoes, ligacoes
}

// colunasDoGrafo devolve a coluna de cada Frame. Sem Ligação, é uma grade
// quadrada na ordem de declaração; com Ligação, é a distância até a tela de
// entrada mais próxima, medida em largura.
//
// A distância é a menor, não a maior. Quase todo fluxo real tem Ligação de
// volta — um botão "Sair" que devolve ao login — e medir pelo caminho mais
// longo faria essa aresta empurrar a tela de entrada para o fim da Prancheta.
// A busca em largura ignora a aresta de volta por construção.
func colunasDoGrafo(n int, ligacoes []ligacao) []int {
	coluna := make([]int, n)
	if len(ligacoes) == 0 {
		largura := int(math.Ceil(math.Sqrt(float64(n))))
		for i := range coluna {
			coluna[i] = i % largura
		}
		return coluna
	}

	saida := make([][]int, n)
	temEntrada := make([]bool, n)
	for _, lg := range ligacoes {
		if lg.de == lg.para {
			// A auto-Ligação não é entrada de ninguém: um Frame que só aponta
			// para si mesmo continua sendo tela de entrada.
			continue
		}
		saida[lg.de] = append(saida[lg.de], lg.para)
		temEntrada[lg.para] = true
	}

	visto := make([]bool, n)
	fila := make([]int, 0, n)
	visita := func(i, c int) {
		if visto[i] {
			return
		}
		visto[i] = true
		coluna[i] = c
		fila = append(fila, i)
	}

	// Primeiro as telas de entrada, na ordem de declaração. Um fluxo todo em
	// ciclo não tem nenhuma, e aí a primeira tela declarada é a entrada — quem
	// escreveu o Documento começou por ela.
	for i := 0; i < n; i++ {
		if !temEntrada[i] {
			visita(i, 0)
		}
	}
	if len(fila) == 0 {
		visita(0, 0)
	}
	for p := 0; p < len(fila); p++ {
		atual := fila[p]
		for _, prox := range saida[atual] {
			visita(prox, coluna[atual]+1)
		}
		// Um trecho desligado do resto só aparece quando a fila seca: ele
		// recomeça na coluna 0, como um segundo fluxo.
		if p == len(fila)-1 {
			for i := 0; i < n; i++ {
				if !visto[i] {
					visita(i, 0)
					break
				}
			}
		}
	}
	return coluna
}

// posiciona converte colunas em coordenadas: cada coluna é larga o bastante
// para o seu Frame mais largo, e as colunas são centradas na vertical entre si.
func posiciona(d *scene.Documento, coluna []int, posicoes []posicao) {
	total := 0
	for _, c := range coluna {
		if c+1 > total {
			total = c + 1
		}
	}

	largura := make([]float64, total)
	altura := make([]float64, total)
	for i, f := range d.Frames {
		c := coluna[i]
		if l := float64(f.L); l > largura[c] {
			largura[c] = l
		}
		if altura[c] > 0 {
			altura[c] += intervaloV
		}
		altura[c] += float64(f.A) + alturaDoTitulo
	}

	x := make([]float64, total)
	corrente := margemDaPrancheta
	for c := 0; c < total; c++ {
		x[c] = corrente
		corrente += largura[c] + intervaloH
	}

	maisAlta := 0.0
	for _, a := range altura {
		if a > maisAlta {
			maisAlta = a
		}
	}

	y := make([]float64, total)
	for c := 0; c < total; c++ {
		y[c] = margemDaPrancheta + (maisAlta-altura[c])/2
	}
	for i, f := range d.Frames {
		c := coluna[i]
		// O Frame é centrado na largura da coluna: uma tela estreita ao lado de
		// uma larga não fica encostada na esquerda.
		posicoes[i] = posicao{
			X: x[c] + (largura[c]-float64(f.L))/2,
			Y: y[c] + alturaDoTitulo,
		}
		y[c] += float64(f.A) + alturaDoTitulo + intervaloV
	}
}

// mundo devolve as dimensões da Prancheta inteira, já com a margem.
func mundo(d *scene.Documento, posicoes []posicao) (l, a float64) {
	for i, f := range d.Frames {
		if d := posicoes[i].X + float64(f.L); d > l {
			l = d
		}
		if b := posicoes[i].Y + float64(f.A); b > a {
			a = b
		}
	}
	return l + margemDaPrancheta, a + margemDaPrancheta
}
