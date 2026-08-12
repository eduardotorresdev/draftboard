// Package resolve transforma o Documento declarado no modelo resolvido de
// internal/scene.
//
// A resolução tem duas fases independentes. O achatamento converte a árvore
// declarada em listas planas de Elementos com geometria absoluta em pixels do
// Frame — é nele que Instâncias, Slots e Repetições são materializados. A
// segunda fase percorre essas listas já planas na ordem de pintura e atribui a
// Elevação e o Tom de cada Elemento, sem saber de onde ele veio.
package resolve

import (
	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/schema"
)

// Arquivo lê o Documento no caminho dado, valida, achata Instâncias, Slots e
// Repetições, e calcula Elevação e Tom de cada Elemento.
//
// Devolve erro do tipo *scene.Erro quando a resolução falha. Os avisos são
// devolvidos mesmo quando não há erro.
func Arquivo(caminho string) (*scene.Documento, []scene.Aviso, error) {
	doc, comp, err := schema.Arquivo(caminho)
	if err != nil {
		return nil, nil, err
	}
	if comp != nil {
		return nil, nil, &scene.Erro{
			Arquivo: caminho,
			Msg:     "esperava um Documento, mas o arquivo não declara `frames`; Componente só pode ser usado por uma Instância",
		}
	}

	r := &resolucao{arquivo: caminho}
	resolvido := &scene.Documento{Nome: doc.Nome}
	for _, f := range doc.Frames {
		frame, err := r.frame(f)
		if err != nil {
			return nil, r.avisos, err
		}
		atribuiElevacao(frame.Camadas)
		resolvido.Frames = append(resolvido.Frames, frame)
	}
	return resolvido, r.avisos, nil
}

// resolucao carrega o estado comum das duas fases: o arquivo de origem, os
// avisos acumulados e as dimensões do Frame em resolução.
type resolucao struct {
	arquivo string
	avisos  []scene.Aviso
	// frameL e frameA são as dimensões em pixels do Frame sendo achatado,
	// usadas para detectar Elementos fora do Frame.
	frameL, frameA float64
}

func (r *resolucao) aviso(local, msg string) {
	r.avisos = append(r.avisos, scene.Aviso{Arquivo: r.arquivo, Local: local, Msg: msg})
}

// frame achata um Frame declarado: cada Camada vira uma lista plana de
// Elementos com geometria absoluta, ainda sem Elevação nem Tom.
func (r *resolucao) frame(f schema.Frame) (scene.Frame, error) {
	r.frameL, r.frameA = float64(f.L), float64(f.A)
	resolvido := scene.Frame{Nome: f.Nome, L: f.L, A: f.A}
	// O espaço do Frame é o próprio Frame na origem: todo valor declarado é
	// porcentagem do eixo correspondente.
	esp := espaco{L: r.frameL, A: r.frameA}
	for _, c := range f.Camadas {
		var elementos []scene.Elemento
		if err := r.achata(c.Elementos, esp, "", "", &elementos); err != nil {
			return scene.Frame{}, err
		}
		resolvido.Camadas = append(resolvido.Camadas, scene.Camada{Nome: c.Nome, Elementos: elementos})
	}
	return resolvido, nil
}
