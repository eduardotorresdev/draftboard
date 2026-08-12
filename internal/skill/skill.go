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
// A gravação é atômica: o conteúdo vai para um arquivo temporário no mesmo
// diretório e só então é renomeado por cima do destino. Disso saem três
// garantias. Nunca existe um SKILL.md truncado ou ausente, nem quando a
// gravação falha no meio — a instalação anterior sobrevive intacta. Um link
// simbólico plantado em <dir>/draftboard/SKILL.md é substituído pelo arquivo,
// não seguido, então instalar a skill não vira escrita no alvo do link. E não
// há caso especial por sistema operacional.
//
// Um link simbólico no diretório <dir>/draftboard, porém, É seguido, e isso é
// deliberado: apontar o diretório de skills para outro lugar (um repositório
// de dotfiles, um volume compartilhado) é arranjo legítimo de operador, e
// nesse caso gravar no alvo do link é exatamente o comportamento esperado. A
// assimetria é intencional — o link no arquivo final é acidente ou ataque, o
// link no diretório é configuração.
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
	if err := gravaAtomico(caminho, destino); err != nil {
		return "", err
	}
	return caminho, nil
}

// gravaAtomico escreve a skill em caminho passando por um temporário em dir.
// Em caso de falha, remove o temporário e deixa o destino como estava.
func gravaAtomico(caminho, dir string) error {
	tmp, err := os.CreateTemp(dir, ".SKILL.md-*")
	if err != nil {
		return fmt.Errorf("criar arquivo temporário em %s: %w", dir, err)
	}
	nomeTmp := tmp.Name()
	defer os.Remove(nomeTmp) // no-op quando o rename já aconteceu

	if _, err := tmp.WriteString(conteudo); err != nil {
		tmp.Close()
		return fmt.Errorf("gravar %s: %w", nomeTmp, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fechar %s: %w", nomeTmp, err)
	}
	// CreateTemp cria com 0600; a skill é legível como qualquer doc instalado.
	if err := os.Chmod(nomeTmp, 0o644); err != nil {
		return fmt.Errorf("ajustar permissões de %s: %w", nomeTmp, err)
	}
	if err := os.Rename(nomeTmp, caminho); err != nil {
		return fmt.Errorf("gravar %s: %w", caminho, err)
	}
	return nil
}

// diretorioPadrao devolve ~/.claude/skills.
func diretorioPadrao() (string, error) {
	casa, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("localizar diretório do usuário: %w", err)
	}
	return filepath.Join(casa, ".claude", "skills"), nil
}
