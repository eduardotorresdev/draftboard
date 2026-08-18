// Package board monta a Prancheta: um arquivo HTML autocontido que dispõe todos
// os Frames de um Documento numa superfície única, desenha as Ligações entre
// eles e permite navegar e inspecionar o desenho.
//
// A Prancheta não é um formato de export de Frame: a imagem de um Frame
// continua sendo o WebP. Ela existe para mostrar o que a imagem por Frame não
// mostra — que uma tela leva a outra.
package board

import (
	"bufio"
	"fmt"
	"io"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// LimiteDeElementos é o teto de Elementos da Prancheta inteira. Cada Elemento
// vira um nó do DOM, e o navegador é o gargalo aqui, não o gerador: um
// Documento acima deste teto é recusado antes de qualquer alocação.
const LimiteDeElementos = 50_000

// Elementos conta os Elementos de todos os Frames do Documento. Existe para que
// a CLI possa recusar um Documento grande demais antes de montar o HTML.
func Elementos(d *scene.Documento) int {
	n := 0
	for _, f := range d.Frames {
		for _, c := range f.Camadas {
			n += len(c.Elementos)
		}
	}
	return n
}

// Escreve monta a Prancheta do Documento resolvido. A saída é determinística:
// o mesmo Documento produz os mesmos bytes.
func Escreve(w io.Writer, d *scene.Documento) error {
	posicoes, ligacoes := dispoe(d)
	b := bufio.NewWriter(w)

	fmt.Fprintf(b, "<!doctype html>\n<html lang=\"pt-BR\">\n<head>\n")
	fmt.Fprintf(b, "<meta charset=\"utf-8\">\n")
	fmt.Fprintf(b, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(b, "<title>prancheta %s</title>\n", escapa(d.Nome))
	fmt.Fprintf(b, "<style>\n%s</style>\n</head>\n<body>\n", estilo)

	escreveBarra(b, d, ligacoes)
	fmt.Fprintf(b, "<main id=\"prancheta\">\n")
	escreveSVG(b, d, posicoes, ligacoes)
	fmt.Fprintf(b, "</main>\n")
	escrevePainel(b)

	fmt.Fprintf(b, "<script>\n%s</script>\n</body>\n</html>\n", roteiro)
	return b.Flush()
}

// escreveBarra escreve o cabeçalho fixo: o nome do Documento, a contagem do que
// há nele e a legenda das teclas.
func escreveBarra(b *bufio.Writer, d *scene.Documento, ligacoes []ligacao) {
	fmt.Fprintf(b, "<header class=\"barra\">\n")
	fmt.Fprintf(b, "<strong>%s</strong>\n", escapa(d.Nome))
	fmt.Fprintf(b, "<span class=\"conta\">%s &middot; %s</span>\n",
		plural(len(d.Frames), "Frame", "Frames"),
		plural(len(ligacoes), "Ligação", "Ligações"))
	fmt.Fprintf(b, "<span class=\"teclas\">\n")
	fmt.Fprintf(b, "<button type=\"button\" data-acao=\"menos\" title=\"afastar\">&minus;</button>\n")
	fmt.Fprintf(b, "<button type=\"button\" data-acao=\"mais\" title=\"aproximar\">+</button>\n")
	fmt.Fprintf(b, "<button type=\"button\" data-acao=\"ajustar\" title=\"ajustar à tela (0)\">ajustar</button>\n")
	fmt.Fprintf(b, "<button type=\"button\" data-acao=\"ligacoes\" title=\"mostrar Ligações (l)\" aria-pressed=\"true\">Ligações</button>\n")
	fmt.Fprintf(b, "<button type=\"button\" data-acao=\"notas\" title=\"realçar Elementos com Nota (n)\" aria-pressed=\"false\">Notas</button>\n")
	fmt.Fprintf(b, "</span>\n</header>\n")
}

// escrevePainel escreve o painel de inspeção, vazio: o roteiro o preenche a
// cada clique.
func escrevePainel(b *bufio.Writer) {
	fmt.Fprintf(b, "<aside id=\"painel\" hidden>\n")
	fmt.Fprintf(b, "<button type=\"button\" id=\"fecha\" title=\"fechar (esc)\">&times;</button>\n")
	fmt.Fprintf(b, "<dl id=\"campos\"></dl>\n")
	fmt.Fprintf(b, "</aside>\n")
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
