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
//
// Num checkout feito com core.symlinks=false (padrão em alguns ambientes
// Windows, e forçável em qualquer clone), o Git materializa o SKILL.md da raiz
// como um arquivo comum de 23 bytes contendo o texto "internal/skill/SKILL.md"
// em vez de um link. Isso não afeta build nem testes — o //go:embed lê o
// arquivo canônico —, mas quem abrir o arquivo da raiz nesse checkout verá o
// caminho, não a skill. O conteúdo de verdade está sempre em
// internal/skill/SKILL.md.
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
// substitui o arquivo e não é erro.
//
// A gravação nunca segue um link simbólico plantado no destino: um SKILL.md
// preexistente é removido antes, e o arquivo novo é criado com O_EXCL e
// O_NOFOLLOW. Instalar a skill não pode virar uma escrita arbitrária em outro
// lugar da máquina de quem instala.
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
	if err := os.Remove(caminho); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("remover %s: %w", caminho, err)
	}
	f, err := os.OpenFile(caminho, os.O_WRONLY|os.O_CREATE|os.O_EXCL|semSeguirLink, 0o644)
	if err != nil {
		return "", fmt.Errorf("gravar %s: %w", caminho, err)
	}
	if _, err := f.WriteString(conteudo); err != nil {
		f.Close()
		return "", fmt.Errorf("gravar %s: %w", caminho, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("fechar %s: %w", caminho, err)
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
