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
	"fmt"
	"path/filepath"

	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/schema"
)

// Arquivo lê o Documento no caminho dado, valida, achata Instâncias, Slots e
// Repetições, e calcula Elevação e Tom de cada Elemento.
//
// Devolve erro do tipo *scene.Erro quando a resolução falha. Os avisos são
// devolvidos mesmo quando não há erro.
func Arquivo(caminho string) (*scene.Documento, []scene.Aviso, error) {
	doc, err := schema.LeDocumento(caminho)
	if err != nil {
		return nil, nil, err
	}

	r := &resolucao{
		arquivo:        caminho,
		dirDoDocumento: filepath.Dir(caminho),
		componentes:    map[string]*schema.Componente{},
	}
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
	// dirDoDocumento é o diretório do Documento raiz: contra ele se mede a
	// Origem que o `inspect` imprime.
	dirDoDocumento string
	avisos         []scene.Aviso
	// componentes guarda cada Componente já decodificado pelo seu caminho
	// absoluto, para que um Componente usado muitas vezes seja lido uma só.
	componentes map[string]*schema.Componente
	// frameL e frameA são as dimensões em pixels do Frame sendo achatado,
	// usadas para detectar Elementos fora do Frame.
	frameL, frameA float64
}

func (r *resolucao) aviso(local, msg string) {
	r.avisos = append(r.avisos, scene.Aviso{Arquivo: r.arquivo, Local: local, Msg: msg})
}

// erro devolve um erro de resolução já posicionado no Documento raiz. local
// atravessa a cadeia de Componentes.
func (r *resolucao) erro(local, formato string, args ...any) error {
	return &scene.Erro{Arquivo: r.arquivo, Local: local, Msg: fmt.Sprintf(formato, args...)}
}

// frame achata um Frame declarado: cada Camada vira uma lista plana de
// Elementos com geometria absoluta, ainda sem Elevação nem Tom.
func (r *resolucao) frame(f schema.Frame) (scene.Frame, error) {
	r.frameL, r.frameA = float64(f.L), float64(f.A)
	resolvido := scene.Frame{Nome: f.Nome, L: f.L, A: f.A}
	// O espaço do Frame é o próprio Frame na origem: todo valor declarado é
	// porcentagem do eixo correspondente.
	esp := espaco{L: r.frameL, A: r.frameA}
	// Os nós do Documento são inline: sem Origem, sem Componente na pilha, e
	// com `use` resolvido contra o diretório do próprio Documento.
	raiz := contexto{dir: r.dirDoDocumento}
	for _, c := range f.Camadas {
		var elementos []scene.Elemento
		if err := r.achata(c.Elementos, esp, "", raiz, &elementos); err != nil {
			return scene.Frame{}, err
		}
		resolvido.Camadas = append(resolvido.Camadas, scene.Camada{Nome: c.Nome, Elementos: elementos})
	}
	return resolvido, nil
}
