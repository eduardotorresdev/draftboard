package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Executavel devolve o caminho real do binário em execução.
//
// A resolução de link simbólico não é detalhe: `~/bin/draftboard` muitas vezes
// é um link para o arquivo de verdade, e renomear sobre o CAMINHO DO LINK
// trocaria o link por um arquivo comum, órfanando o binário real e quebrando
// todo outro nome que apontava para ele. A troca acontece sempre no arquivo
// real.
func Executavel() (string, error) {
	caminho, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o binário em execução: %w", err)
	}
	real, err := filepath.EvalSymlinks(caminho)
	if err != nil {
		return "", fmt.Errorf("não foi possível resolver o caminho de %s: %w", caminho, err)
	}
	return real, nil
}

// Aplica baixa o Ativo do Lançamento, confere a soma SHA-256 contra o arquivo
// de somas do próprio Lançamento e substitui o binário em Opcoes.Destino.
// Devolve o caminho substituído. Avisos e a linha de status vão para progresso,
// que pode ser nil.
//
// # A invariante
//
// **Nada é renomeado antes de a soma conferir.** Isso precisa ser dito porque a
// sequência parece violar o contrário: o Ativo é baixado, descomprimido e
// escrito em disco num único passe, e a soma só é conhecida no último byte —
// quando os bytes extraídos já estão gravados. É seguro só porque eles caem num
// arquivo temporário privado que nada renomeia até a conferência passar. A
// alternativa (bufferizar o tarball inteiro em memória, conferir, e só então
// extrair) custaria alguns MB e um segundo passe sem ganhar nada.
//
// # Por que a troca é sempre rename
//
// No Linux, abrir para escrita um arquivo em execução devolve ETXTBSY;
// rename(2) não sofre disso, porque desvincula a entrada de diretório enquanto
// a imagem em execução segue com o vnode. Então o binário é sempre SUBSTITUÍDO
// por rename, nunca truncado no lugar. É o mesmo raciocínio de
// skill.gravaAtomico, um nível acima.
//
// Não existe arquivo de backup. O rename já é atômico, então não há instante em
// que o destino esteja ausente; a versão em dois passos (destino -> destino.old,
// tmp -> destino) só acrescentaria lixo `.old` num diretório do $PATH quando o
// segundo rename falha.
func Aplica(o Opcoes, l Lancamento, progresso io.Writer) (string, error) {
	o = comPadroesDeConsulta(o)
	if o.Destino == "" {
		destino, err := Executavel()
		if err != nil {
			return "", err
		}
		o.Destino = destino
	}
	// A resolução vale também para um Destino dado por quem chama, e não só
	// para o que veio de Executavel: a garantia "a troca acontece no arquivo
	// real, nunca no link" não pode depender de quem montou as Opcoes.
	real, err := filepath.EvalSymlinks(o.Destino)
	if err != nil {
		return "", fmt.Errorf("não foi possível resolver o caminho de %s: %w", o.Destino, err)
	}
	o.Destino = real

	info, err := os.Stat(o.Destino)
	if err != nil {
		return "", fmt.Errorf("não foi possível inspecionar %s: %w", o.Destino, err)
	}
	permissao := info.Mode().Perm()
	if permissao&0o111 == 0 {
		// Sem esse resgate, o substituto sairia sem bit de execução e o
		// usuário ficaria com um arquivo no $PATH que não roda.
		avisa(progresso, "o binário atual não tem bit de execução; o substituto será gravado com modo 0755")
		permissao = 0o755
	}

	// O temporário nasce no diretório do destino, e não em os.TempDir(), por
	// três motivos de uma só chamada: prova que o diretório é gravável, garante
	// que o rename final não pode falhar com EXDEV — /tmp e /usr/local/bin são
	// devices diferentes na maioria dos containers, e a biblioteca padrão não
	// tem move atômico entre devices — e falha barato, antes de baixar
	// qualquer byte de payload.
	tmp, err := os.CreateTemp(filepath.Dir(o.Destino), ".draftboard-*")
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return "", fmt.Errorf(
				"sem permissão de escrita em %s; repita com sudo ou reinstale com %q",
				filepath.Dir(o.Destino),
				"go install github.com/eduardotorresdev/draftboard@latest")
		}
		return "", fmt.Errorf("não foi possível criar arquivo temporário em %s: %w", filepath.Dir(o.Destino), err)
	}
	nomeTmp := tmp.Name()
	defer os.Remove(nomeTmp) // no-op quando o rename já aconteceu

	// As somas vêm primeiro: são algumas centenas de bytes, então um Lançamento
	// malformado falha em milissegundos em vez de depois de vários MB.
	esperada, err := somaEsperada(o, l)
	if err != nil {
		tmp.Close()
		return "", err
	}

	if progresso != nil {
		fmt.Fprintf(progresso, "baixando %s (%s)\n", l.NomeAtivo, tamanho(l.Bytes))
	}
	obtida, err := extrai(o, l, tmp)
	if err != nil {
		tmp.Close()
		return "", err
	}
	if obtida != esperada {
		tmp.Close()
		return "", fmt.Errorf(
			"soma SHA-256 de %s não confere: esperado %s, obtido %s; o binário não foi alterado",
			l.NomeAtivo, esperada, obtida)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("não foi possível gravar %s: %w", nomeTmp, err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("não foi possível fechar %s: %w", nomeTmp, err)
	}
	// CreateTemp cria com 0600; sem este chmod o substituto não roda.
	if err := os.Chmod(nomeTmp, permissao); err != nil {
		return "", fmt.Errorf("não foi possível ajustar permissões de %s: %w", nomeTmp, err)
	}
	if err := os.Rename(nomeTmp, o.Destino); err != nil {
		return "", fmt.Errorf("não foi possível substituir %s: %w", o.Destino, err)
	}
	return o.Destino, nil
}

// extrai baixa o Ativo, escreve o binário em dst e devolve a soma SHA-256 dos
// bytes COMPRIMIDOS, como publicados — é sobre eles que o arquivo de somas é
// calculado, então o TeeReader envolve o corpo da resposta, não a saída do
// gzip.
func extrai(o Opcoes, l Lancamento, dst *os.File) (string, error) {
	corpo, err := baixa(o, l.URLAtivo, "application/octet-stream")
	if err != nil {
		return "", fmt.Errorf("não foi possível baixar %s: %w", l.NomeAtivo, err)
	}
	defer corpo.Close()

	soma := sha256.New()
	bruto := io.TeeReader(corpo, soma)
	gz, err := gzip.NewReader(bruto)
	if err != nil {
		return "", fmt.Errorf("o ativo %s não é um gzip válido: %w", l.NomeAtivo, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	cabecalho, err := tr.Next()
	if err != nil {
		return "", fmt.Errorf("o ativo %s não contém nenhum arquivo: %w", l.NomeAtivo, err)
	}
	if err := conferePrimeiroMembro(l.NomeAtivo, cabecalho); err != nil {
		return "", err
	}
	// O teto é conferido lendo um byte além dele: se sobrou byte, passou.
	gravados, err := io.Copy(dst, io.LimitReader(tr, o.TetoBytes+1))
	if err != nil {
		return "", fmt.Errorf("não foi possível extrair %s: %w", l.NomeAtivo, err)
	}
	if gravados > o.TetoBytes {
		return "", fmt.Errorf("o ativo passou de %s descompactado; recusado", tamanho(o.TetoBytes))
	}
	if _, err := tr.Next(); !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("o ativo %s não contém exatamente um arquivo chamado %q", l.NomeAtivo, nomeNoTar)
	}
	// O tar pode parar antes do fim do gzip. Drenar o resto é o que faz a soma
	// cobrir o Ativo inteiro, e é também o que pega download cortado no meio.
	if _, err := io.Copy(io.Discard, bruto); err != nil {
		return "", fmt.Errorf("não foi possível baixar %s: %w", l.NomeAtivo, err)
	}
	return hex.EncodeToString(soma.Sum(nil)), nil
}

// conferePrimeiroMembro recusa qualquer coisa que não seja o binário esperado.
// O nome do cabeçalho nunca decide onde o conteúdo é gravado — o destino é o
// temporário já aberto —, mas um Ativo mal construído tem de falhar alto em vez
// de instalar silenciosamente o LICENSE no lugar do binário.
func conferePrimeiroMembro(ativo string, c *tar.Header) error {
	nome := c.Name
	if c.Typeflag != tar.TypeReg ||
		nome != nomeNoTar ||
		strings.ContainsRune(nome, '/') ||
		strings.Contains(nome, "..") {
		return fmt.Errorf("o ativo %s não contém exatamente um arquivo chamado %q", ativo, nomeNoTar)
	}
	return nil
}

// avisa escreve um aviso na saída de progresso, com o mesmo prefixo da CLI.
func avisa(progresso io.Writer, msg string) {
	if progresso == nil {
		return
	}
	fmt.Fprintln(progresso, "aviso: "+msg)
}

// tamanho formata bytes em MiB ou KiB, para a linha de status.
func tamanho(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(bytes)/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(bytes)/(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
