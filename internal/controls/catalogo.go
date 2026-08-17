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
		Nome:    "tabs",
		Chaves:  []string{"items", "active"},
		padroes: padraoDeLista,
		valida:  validaLista,
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
		detalhe: detalheDaLista,
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

// --- Fatia 2 do catálogo ------------------------------------------------
//
// Nenhum Controle daqui inventa campo: todos cabem em `label`, `items`,
// `active` e `value`, que a decodificação já conhece. Foi o teste da
// abstração — precisar de uma chave nova aqui seria sinal de que a fatia 1
// desenhou a fronteira no lugar errado.

func init() {
	acrescenta("checkbox", Definicao{
		Chaves: []string{"label", "active"},
		padroes: func(p *Parametros) {
			if p.Ativo == Ausente {
				p.Ativo = 0
			}
		},
		valida:  validaLiga,
		detalhe: detalheDoRotuloEAtivo,
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, false)}
			lado := menor(0.55*a, 0.55*l)
			mx, my := x+0.25*lado, y+(a-lado)/2
			pecas = append(pecas, Peca{Segmento: "marca", Forma: scene.Retangulo,
				Arredondado: true, X: mx, Y: my, L: lado, A: lado})
			if p.Ativo == 1 {
				// O tique fica geometricamente contido na marca, e é a
				// contenção — não uma cor declarada — que lhe dá o degrau de
				// Elevação a mais que o faz aparecer.
				pecas = append(pecas, Peca{Segmento: "marca/tique", Forma: scene.Retangulo,
					Arredondado: true, X: mx + 0.25*lado, Y: my + 0.25*lado,
					L: 0.5 * lado, A: 0.5 * lado})
			}
			tx := mx + lado + 0.4*lado
			sobra := x + l - tx - 0.05*l
			return append(pecas, conteudoTextual(p.Rotulo, "rotulo",
				tx, y+0.2*a, sobra, 0.6*a,
				tx, y+0.375*a, 0.4*sobra, 0.25*a, scene.AEsquerda))
		},
	})

	acrescenta("radio", Definicao{
		Chaves:  []string{"items", "active"},
		padroes: padraoDeLista,
		valida:  validaLista,
		detalhe: detalheDaLista,
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, false)}
			alt := a / float64(p.Quantos)
			d := menor(0.5*alt, 0.5*l)
			for i := 0; i < p.Quantos; i++ {
				iy := y + float64(i)*alt
				item := fmt.Sprintf("item#%d", i)
				pecas = append(pecas, Peca{Segmento: item, Forma: scene.Circulo,
					X: x + 0.5*d, Y: iy + (alt-d)/2, L: d, A: d})
				if p.Ativo == i+1 {
					pecas = append(pecas, Peca{Segmento: item + "/ativo", Forma: scene.Circulo,
						X: x + 0.5*d + 0.25*d, Y: iy + (alt-d)/2 + 0.25*d,
						L: 0.5 * d, A: 0.5 * d})
				}
				tx := x + 2*d
				sobra := x + l - tx - 0.05*l
				pecas = append(pecas, conteudoTextual(rotuloDoItem(p, i), item+"/rotulo",
					tx, iy+0.2*alt, sobra, 0.6*alt,
					tx, iy+0.375*alt, 0.4*sobra, 0.25*alt, scene.AEsquerda))
			}
			return pecas
		},
	})

	acrescenta("toggle", Definicao{
		Chaves: []string{"active"},
		padroes: func(p *Parametros) {
			if p.Ativo == Ausente {
				p.Ativo = 0
			}
		},
		valida:  validaLiga,
		detalhe: func(p Parametros) string { return fmt.Sprintf("ativo=%d", p.Ativo) },
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			// Ligado e desligado se distinguem só pela posição do botão, como
			// num interruptor de verdade. Não há segundo Tom a declarar: o
			// botão é escuro porque está contido no trilho.
			d := menor(a, l)
			bx := x
			if p.Ativo == 1 {
				bx = x + l - d
			}
			return []Peca{
				cabeca(x, y, l, a, true),
				{Segmento: "botao", Forma: scene.Circulo,
					X: bx, Y: y + (a-d)/2, L: d, A: d},
			}
		},
	})

	acrescenta("accordion", Definicao{
		Chaves:  []string{"items", "active"},
		padroes: padraoDeLista,
		valida:  validaLista,
		detalhe: detalheDaLista,
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			// A seção aberta vale três faixas: uma de cabeçalho e duas de
			// corpo. Sem nenhuma aberta, as faixas dividem a altura por igual.
			faixas := float64(p.Quantos)
			if p.Ativo > 0 {
				faixas += 2
			}
			h := a / faixas
			pecas := []Peca{cabeca(x, y, l, a, false)}
			iy := y
			for i := 0; i < p.Quantos; i++ {
				item := fmt.Sprintf("item#%d", i)
				// O cabeçalho não é uma placa: pela mesma razão da aba ativa do
				// `tabs`, uma placa atrás do rótulo o empurra um degrau de
				// Elevação para baixo e piora o contraste em vez de melhorar.
				// A faixa é separada por um filete, que custa uma peça só.
				pecas = append(pecas, conteudoTextual(rotuloDoItem(p, i), item,
					x+0.04*l, iy+0.2*h, 0.7*l, 0.6*h,
					x+0.04*l, iy+0.4*h, 0.3*l, 0.2*h, scene.AEsquerda))
				pecas = append(pecas, Peca{Segmento: item + "/seta", Forma: scene.Retangulo,
					Arredondado: true, X: x + 0.9*l, Y: iy + 0.44*h, L: 0.06 * l, A: 0.12 * h})
				pecas = append(pecas, Peca{Segmento: item + "/filete", Forma: scene.Retangulo,
					X: x + 0.04*l, Y: iy + 0.97*h, L: 0.92 * l, A: 0.03 * h})
				iy += h
				if p.Ativo != i+1 {
					continue
				}
				corpo := item + "/corpo"
				pecas = append(pecas, Peca{Segmento: corpo, Forma: scene.Retangulo,
					Arredondado: true, X: x + 0.04*l, Y: iy + 0.05*h, L: 0.92 * l, A: 1.9 * h})
				for j, largura := range []float64{0.84, 0.6} {
					pecas = append(pecas, Peca{Segmento: fmt.Sprintf("%s/linha#%d", corpo, j),
						Forma: scene.Retangulo, Arredondado: true,
						X: x + 0.08*l, Y: iy + (0.45+float64(j)*0.7)*h,
						L: largura * l, A: 0.25 * h})
				}
				iy += 2 * h
			}
			return pecas
		},
	})

	acrescenta("dropdown", Definicao{
		Chaves:  []string{"label"},
		detalhe: detalheDoRotulo,
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, true)}
			pecas = append(pecas, conteudoTextual(p.Rotulo, "rotulo",
				x+0.05*l, y+0.2*a, 0.78*l, 0.6*a,
				x+0.05*l, y+0.375*a, 0.35*l, 0.25*a, scene.AEsquerda))
			return append(pecas, Peca{Segmento: "seta", Forma: scene.Retangulo,
				Arredondado: true, X: x + 0.87*l, Y: y + 0.42*a, L: 0.08 * l, A: 0.16 * a})
		},
	})

	acrescenta("avatar", Definicao{
		Chaves:  []string{"label"},
		detalhe: detalheDoRotulo,
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			// A cabeça ocupa a box declarada, como manda o contrato, mas sai
			// redonda: o rasterizador inscreve o Círculo no menor lado.
			pecas := []Peca{{Forma: scene.Circulo, X: x, Y: y, L: l, A: a}}
			if p.Rotulo == "" {
				// Sem iniciais, o avatar é o disco e nada mais. A barra cinza
				// de texto placeholder dos outros Controles ficaria atravessada
				// no meio do disco e leria como outra coisa.
				return pecas
			}
			return append(pecas, Peca{Segmento: "rotulo", Forma: scene.Texto,
				X: x + 0.2*l, Y: y + 0.3*a, L: 0.6 * l, A: 0.4 * a,
				Rotulo: p.Rotulo, Alinhamento: scene.AoCentro})
		},
	})

	acrescenta("badge", Definicao{
		Chaves:  []string{"label"},
		detalhe: detalheDoRotulo,
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, true)}
			return append(pecas, conteudoTextual(p.Rotulo, "rotulo",
				x+0.1*l, y+0.2*a, 0.8*l, 0.6*a,
				x+0.25*l, y+0.4*a, 0.5*l, 0.2*a, scene.AoCentro))
		},
	})

	acrescenta("progress", Definicao{
		Chaves: []string{"value"},
		padroes: func(p *Parametros) {
			if p.Valor == Ausente {
				p.Valor = 50
			}
		},
		detalhe: func(p Parametros) string { return "valor=" + numero(p.Valor) },
		layout: func(p Parametros, x, y, l, a float64) []Peca {
			pecas := []Peca{cabeca(x, y, l, a, true)}
			if p.Valor <= 0 {
				// Peça de largura zero viraria aviso de "Elemento de área
				// zero". Progresso em 0% é o trilho vazio, e é só isso.
				return pecas
			}
			return append(pecas, Peca{Segmento: "preenchido", Forma: scene.Retangulo,
				Arredondado: true, X: x, Y: y, L: l * p.Valor / 100, A: a})
		},
	})
}

// acrescenta registra um Controle no catálogo. Existe para que a entrada não
// precise repetir o próprio nome, e para que um nome duplicado exploda na
// carga do pacote em vez de sobrescrever a entrada anterior em silêncio.
func acrescenta(nome string, d Definicao) {
	if _, repetido := catalogo[nome]; repetido {
		panic("controls: Controle declarado duas vezes no catálogo: " + nome)
	}
	d.Nome = nome
	catalogo[nome] = d
}

// padraoDeLista é o padrão dos Controles de lista: três itens, o primeiro ativo.
func padraoDeLista(p *Parametros) {
	if p.Quantos == Ausente {
		p.Quantos = 3
	}
	if p.Ativo == Ausente {
		p.Ativo = 1
	}
}

// validaLista recusa lista vazia e item ativo que não existe.
func validaLista(p Parametros) string {
	if p.Quantos < 1 {
		return fmt.Sprintf("campo %q do Controle deve estar entre 1 e %d, encontrou %d",
			"items", LimiteDeItens, p.Quantos)
	}
	if p.Ativo > p.Quantos {
		return fmt.Sprintf("campo %q do Controle aponta o item %d, mas só existem %d itens",
			"active", p.Ativo, p.Quantos)
	}
	return ""
}

// validaLiga é a validação dos Controles de dois estados: `active` só pode
// dizer ligado ou desligado, e um 2 aqui é quase sempre engano de quem achou
// que era uma lista.
func validaLiga(p Parametros) string {
	if p.Ativo > 1 {
		return fmt.Sprintf("campo %q do Controle só aceita 0 ou 1, encontrou %d", "active", p.Ativo)
	}
	return ""
}

// menor devolve o menor de dois lados. Serve para manter Círculo redondo e
// marca quadrada em caixa de qualquer proporção.
func menor(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func detalheDoRotuloEAtivo(p Parametros) string {
	ativo := fmt.Sprintf("ativo=%d", p.Ativo)
	if r := detalheDoRotulo(p); r != "" {
		return r + " " + ativo
	}
	return ativo
}

// detalheDaLista formata itens e item ativo, que é o Detalhe comum a tabs,
// radio e accordion.
func detalheDaLista(p Parametros) string {
	itens := fmt.Sprintf("itens=%d", p.Quantos)
	if len(p.Itens) > 0 {
		itens = "itens=" + rotulos(p.Itens)
	}
	return fmt.Sprintf("%s ativo=%d", itens, p.Ativo)
}
