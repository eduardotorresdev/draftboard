// Package skill guarda a skill embutida no binário: o documento que ensina a
// um agente a CLI do draftboard e o formato YAML do Documento e do Componente.
//
// # Onde mora o SKILL.md
//
// A diretiva //go:embed não consegue subir de diretório, então o arquivo
// canônico é internal/skill/SKILL.md — o mesmo diretório deste pacote. O
// SKILL.md da raiz do repositório é um link simbólico relativo para ele, de
// modo que existe um único arquivo, sem cópia que possa dessincronizar: editar
// qualquer um dos dois caminhos edita o mesmo conteúdo, e é esse conteúdo que
// entra no binário.
package skill

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed SKILL.md
var conteudo string

// nomeDoDiretorio é o subdiretório criado dentro do destino da instalação.
const nomeDoDiretorio = "draftboard"

// nomeDoArquivo é o nome do arquivo gravado pela instalação.
const nomeDoArquivo = "SKILL.md"

// Conteudo devolve o texto integral da skill embutida.
func Conteudo() string {
	return conteudo
}

// Imprime escreve a skill embutida no writer dado.
func Imprime(w io.Writer) error {
	if _, err := io.WriteString(w, conteudo); err != nil {
		return fmt.Errorf("imprimir skill: %w", err)
	}
	return nil
}

// Instala grava a skill em <dir>/draftboard/SKILL.md, criando os diretórios
// necessários, e devolve o caminho escrito. Quando dir é vazio, o destino
// padrão é ~/.claude/skills. Reinstalar sobre uma instalação existente
// sobrescreve o arquivo e não é erro.
func Instala(dir string) (string, error) {
	if dir == "" {
		padrao, err := diretorioPadrao()
		if err != nil {
			return "", err
		}
		dir = padrao
	}
	destino := filepath.Join(dir, nomeDoDiretorio)
	if err := os.MkdirAll(destino, 0o755); err != nil {
		return "", fmt.Errorf("criar diretório %s: %w", destino, err)
	}
	caminho := filepath.Join(destino, nomeDoArquivo)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		return "", fmt.Errorf("gravar %s: %w", caminho, err)
	}
	return caminho, nil
}

// diretorioPadrao devolve ~/.claude/skills.
func diretorioPadrao() (string, error) {
	casa, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("localizar diretório do usuário: %w", err)
	}
	return filepath.Join(casa, ".claude", "skills"), nil
}
