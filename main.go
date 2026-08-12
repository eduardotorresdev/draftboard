// Comando draftboard lê wireframes declarativos em YAML e produz imagens WebP
// em escala de cinza, mais uma árvore textual da estrutura resolvida.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run despacha o verbo da linha de comando e devolve o código de saída: 0 em
// sucesso, 1 em erro. Avisos e erros vão para stderr; só o resultado útil de
// cada verbo vai para stdout.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		imprimeUso(stderr)
		return 1
	}
	switch args[0] {
	case "render":
		return comandoRender(args[1:], stdout, stderr)
	case "inspect":
		return comandoInspect(args[1:], stdout, stderr)
	case "validate":
		return comandoValidate(args[1:], stderr)
	case "skill":
		return comandoSkill(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		imprimeUso(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "erro: verbo desconhecido %q\n", args[0])
		imprimeUso(stderr)
		return 1
	}
}

func imprimeUso(w io.Writer) {
	fmt.Fprint(w, `uso:
  draftboard render   <arquivo.yaml> [--out DIR] [--scale N] [--notes margin|float|off] [--layers]
  draftboard inspect  <arquivo.yaml>
  draftboard validate <arquivo.yaml>
  draftboard skill    [--install [DIR]]
`)
}

// imprimeErro escreve um erro no formato `erro: <arquivo>: <local>: <msg>` e
// devolve o código de saída.
func imprimeErro(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "erro: "+err.Error())
	return 1
}

// imprimeAvisos escreve os avisos acumulados na resolução, um por linha.
func imprimeAvisos(stderr io.Writer, avisos []scene.Aviso) {
	for _, a := range avisos {
		fmt.Fprintln(stderr, a.String())
	}
}

// usoInvalido reporta um erro de linha de comando, que não tem arquivo nem
// localização.
func usoInvalido(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "erro: "+err.Error())
	imprimeUso(stderr)
	return 1
}
