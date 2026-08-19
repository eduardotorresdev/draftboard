// Package inspect imprime a árvore textual de um Documento já resolvido: a
// mesma informação da imagem, legível por agente sem custo de visão.
package inspect

import (
	"bufio"
	"fmt"
	"io"
	"math"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Arvore escreve a árvore resolvida do Documento. Nada é escrito em disco.
func Arvore(w io.Writer, d *scene.Documento) error {
	b := bufio.NewWriter(w)
	fmt.Fprintf(b, "documento %s\n", d.Nome)
	for _, f := range d.Frames {
		fmt.Fprintf(b, "  frame %s %dx%d\n", f.Nome, f.L, f.A)
		for _, c := range f.Camadas {
			fmt.Fprintf(b, "    camada %s\n", c.Nome)
			for _, e := range c.Elementos {
				// O Controle é fechado também na leitura: a árvore mostra a
				// sua cabeça e os parâmetros que o descrevem, e omite as peças
				// que ele materializou. Quem escreveu `control:` não escreveu
				// aquelas peças e não paga tokens para lê-las.
				if e.Interno {
					continue
				}
				fmt.Fprintf(b, "      %s\n", linhaDoElemento(e))
				// A Nota sai entre aspas pela mesma razão que o Rótulo: é
				// texto livre do autor, e crua ela forja a árvore — uma Nota
				// com quebra de linha seguida de seis espaços produz uma linha
				// indistinguível de um Elemento real para quem lê a árvore.
				if e.Nota != "" {
					fmt.Fprintf(b, "        nota: %q\n", e.Nota)
				}
			}
		}
	}
	return b.Flush()
}

// linhaDoElemento formata um Elemento com as coordenadas arredondadas para
// inteiro.
func linhaDoElemento(e scene.Elemento) string {
	linha := fmt.Sprintf("%s %s %d,%d %dx%d tom=%d elev=%d",
		e.Caminho, e.Forma,
		arredonda(e.X), arredonda(e.Y), arredonda(e.L), arredonda(e.A),
		int(e.Tom), e.Elevacao)
	if e.Arredondado {
		linha += " round"
	}
	if e.Origem != "" {
		linha += " de=" + e.Origem
	}
	if e.Controle != "" {
		linha += " controle=" + e.Controle
	}
	if e.Detalhe != "" {
		linha += " " + e.Detalhe
	}
	if e.Destino != "" {
		linha += " para=" + e.Destino
	}
	// O Rótulo vai por último e entre aspas: é o único sufixo cujo conteúdo é
	// texto livre do autor, e um sufixo depois dele não teria como se
	// distinguir do texto.
	if e.Rotulo != "" {
		linha += fmt.Sprintf(" rotulo=%q", e.Rotulo)
	}
	return linha
}

func arredonda(v float64) int {
	return int(math.Round(v))
}
