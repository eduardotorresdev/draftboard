package controls

import (
	"fmt"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Ausente é o valor com que a decodificação marca um campo numérico que o YAML
// não declarou, para que Padroes consiga distinguir "não escrevi" de "escrevi
// zero" — a diferença entre `tabs` sem `active` e `tabs` com `active: 0`.
const Ausente = -1

// catalogo é o conjunto fechado de Controles. Acrescentar um Controle é
// acrescentar uma entrada aqui, e nada mais no repositório.
var catalogo = map[string]Definicao{
	"button": {
		Nome:   "button",
		Chaves: []string{"label"},
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, true)}
			return append(pecas, conteudoTextual(p.Rotulo, "rotulo",
				x+0.08*l, y+0.2*a, 0.84*l, 0.6*a,
				x+0.3*l, y+0.375*a, 0.4*l, 0.25*a, scene.AoCentro))
		},
		detalhe: detalheDoRotulo,
	},

	"input": {
		Nome:   "input",
		Chaves: []string{"label"},
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, true)}
			return append(pecas, conteudoTextual(p.Rotulo, "rotulo",
				x+0.05*l, y+0.2*a, 0.9*l, 0.6*a,
				x+0.05*l, y+0.375*a, 0.35*l, 0.25*a, scene.AEsquerda))
		},
		detalhe: detalheDoRotulo,
	},

	"tabs": {
		Nome:   "tabs",
		Chaves: []string{"items", "active"},
		padroes: func(p *Parametros) {
			if p.Quantos == Ausente {
				p.Quantos = 3
			}
			if p.Ativo == Ausente {
				p.Ativo = 1
			}
		},
		valida: func(p Parametros) string {
			if p.Quantos < 1 {
				return fmt.Sprintf("campo %q do Controle deve estar entre 1 e %d, encontrou %d",
					"items", LimiteDeItens, p.Quantos)
			}
			if p.Ativo > p.Quantos {
				return fmt.Sprintf("campo %q do Controle aponta o item %d, mas só existem %d itens",
					"active", p.Ativo, p.Quantos)
			}
			return ""
		},
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, false)}
			larg := l / float64(p.Quantos)
			for i := 0; i < p.Quantos; i++ {
				ix := x + float64(i)*larg
				item := fmt.Sprintf("item#%d", i)
				pecas = append(pecas, conteudoTextual(rotuloDoItem(p, i), item,
					ix+0.1*larg, y+0.22*a, 0.8*larg, 0.44*a,
					ix+0.25*larg, y+0.34*a, 0.5*larg, 0.2*a, scene.AoCentro))
				if p.Ativo == i+1 {
					// A aba ativa é marcada por sublinhado, e não por uma placa
					// atrás do rótulo: a placa seria mais uma Superfície, e o
					// rótulo em cima dela cairia num degrau de Elevação onde o
					// contraste contra o próprio fundo fica pior, não melhor.
					pecas = append(pecas, Peca{
						Segmento: item + "/ativo", Forma: scene.Retangulo, Arredondado: true,
						X: ix + 0.15*larg, Y: y + 0.78*a, L: 0.7 * larg, A: 0.1 * a,
					})
				}
			}
			return pecas
		},
		detalhe: func(p Parametros) string {
			itens := fmt.Sprintf("itens=%d", p.Quantos)
			if len(p.Itens) > 0 {
				itens = "itens=" + rotulos(p.Itens)
			}
			return fmt.Sprintf("%s ativo=%d", itens, p.Ativo)
		},
	},

	"slider": {
		Nome:   "slider",
		Chaves: []string{"value"},
		padroes: func(p *Parametros) {
			if p.Valor == Ausente {
				p.Valor = 50
			}
		},
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			// O polegar tem o diâmetro da altura do trilho e anda entre as duas
			// pontas sem vazar. O preenchido termina na borda direita do
			// polegar, de modo que o polegar fique geometricamente contido nele
			// — é a contenção que dá ao polegar o degrau de Elevação a mais, e
			// portanto o Tom mais escuro, sem ninguém declarar cor.
			diametro := a
			if l < diametro {
				diametro = l
			}
			centro := diametro/2 + (l-diametro)*p.Valor/100
			return []Peca{
				cabeca(x, y, l, a, true),
				{Segmento: "preenchido", Forma: scene.Retangulo, Arredondado: true,
					X: x, Y: y, L: centro + diametro/2, A: a},
				{Segmento: "polegar", Forma: scene.Circulo,
					X: x + centro - diametro/2, Y: y + (a-diametro)/2,
					L: diametro, A: diametro},
			}
		},
		detalhe: func(p Parametros) string { return "valor=" + numero(p.Valor) },
	},
}

// cabeca é a Peca que ocupa a box declarada. Toda Definicao devolve uma como
// primeira Peca: é a Superfície que sustenta o resto do Controle e é a única
// linha que o Controle imprime na árvore do inspect.
func cabeca(x, y, l, a float64, arredondado bool) Peca {
	return Peca{Forma: scene.Retangulo, Arredondado: arredondado, X: x, Y: y, L: l, A: a}
}

// conteudoTextual devolve o Rótulo quando há texto, e a barra cinza de texto
// placeholder quando não há. É o que mantém a promessa de que um Controle sem
// `label` continua sendo um wireframe legível, e não um buraco.
func conteudoTextual(texto, segmento string,
	tx, ty, tl, ta float64,
	bx, by, bl, ba float64, alinhamento scene.Alinhamento) Peca {
	if texto == "" {
		return Peca{Segmento: segmento, Forma: scene.Retangulo, Arredondado: true,
			X: bx, Y: by, L: bl, A: ba}
	}
	return Peca{Segmento: segmento, Forma: scene.Texto, X: tx, Y: ty, L: tl, A: ta,
		Rotulo: texto, Alinhamento: alinhamento}
}

func detalheDoRotulo(p Parametros) string {
	if p.Rotulo == "" {
		return ""
	}
	return fmt.Sprintf("rotulo=%q", p.Rotulo)
}
