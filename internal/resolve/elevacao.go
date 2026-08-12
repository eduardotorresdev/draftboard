package resolve

import "github.com/eduardotorresdev/draftboard/internal/scene"

// atribuiElevacao percorre os Elementos já achatados de um Frame na ordem de
// pintura — Camadas na ordem declarada, Elementos na ordem declarada dentro da
// Camada — e preenche a Elevação e o Tom de cada um.
//
// A base de uma Camada é o degrau que ela acrescenta sobre tudo que está
// abaixo dela. A Superfície de um Elemento é o último Elemento já pintado cuja
// bounding box o contém; quando não há nenhum, o Elemento se apoia direto na
// base da sua Camada.
func atribuiElevacao(camadas []scene.Camada) {
	// pintados guarda, na ordem de pintura, todos os Elementos já
	// resolvidos do Frame, de todas as Camadas.
	var pintados []*scene.Elemento
	// base começa em 0: a base da primeira Camada é o próprio Frame.
	base := 0
	for i := range camadas {
		if i > 0 {
			base = max(base, maiorElevacao(camadas[i-1])) + 1
		}
		for j := range camadas[i].Elementos {
			e := &camadas[i].Elementos[j]
			elevacaoDaSuperficie := base
			for k := len(pintados) - 1; k >= 0; k-- {
				if contemGeometricamente(pintados[k], e) {
					elevacaoDaSuperficie = pintados[k].Elevacao
					break
				}
			}
			e.Elevacao = max(elevacaoDaSuperficie, base) + 1
			e.Tom = scene.TomDaElevacao(e.Elevacao)
			pintados = append(pintados, e)
		}
	}
}

// maiorElevacao devolve a maior Elevação da Camada, ou 0 quando ela está
// vazia.
func maiorElevacao(c scene.Camada) int {
	maior := 0
	for _, e := range c.Elementos {
		maior = max(maior, e.Elevacao)
	}
	return maior
}

// contemGeometricamente diz se a bounding box declarada de superficie contém a
// de elemento. A contenção é inclusiva: bordas coincidentes contam.
func contemGeometricamente(superficie, elemento *scene.Elemento) bool {
	return superficie.X <= elemento.X &&
		superficie.Y <= elemento.Y &&
		superficie.X+superficie.L >= elemento.X+elemento.L &&
		superficie.Y+superficie.A >= elemento.Y+elemento.A
}
