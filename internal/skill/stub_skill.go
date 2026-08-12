// Package skill carrega a skill embutida que ensina outro agente a escrever
// Documentos. Este arquivo é um esqueleto de compilação escrito por F1 para
// fiar a CLI; F5 o substitui pela implementação real.
package skill

import "io"

// Conteudo devolve o texto da skill embutida.
func Conteudo() string { return "" }

// Imprime escreve o texto da skill embutida.
func Imprime(w io.Writer) error { return nil }

// Instala grava a skill em <dir>/draftboard/SKILL.md e devolve o caminho
// escrito. dir vazio instala em ~/.claude/skills.
func Instala(dir string) (string, error) { return "", nil }
