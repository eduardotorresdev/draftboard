package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// baseURLPadrao é o repositório consultado quando Opcoes.BaseURL é vazia.
const baseURLPadrao = "https://api.github.com/repos/eduardotorresdev/draftboard"

// nomeDasSomas é o Ativo que carrega as somas SHA-256 de todos os outros.
const nomeDasSomas = "checksums.txt"

// nomeNoTar é o único membro que um Ativo pode conter.
const nomeNoTar = "draftboard"

// tetoBytesPadrao limita o tamanho do binário descompactado. O pacote
// compress/gzip não tem limite de razão de compressão, então o teto é a única
// defesa contra um Ativo que expande sem parar.
const tetoBytesPadrao = 64 << 20

// tetoDaResposta limita o JSON da consulta, que na prática tem alguns KB.
const tetoDaResposta = 4 << 20

// Opcoes são as opções da consulta e da troca. O valor zero é o de produção:
// cada campo vazio cai no padrão documentado ao lado.
type Opcoes struct {
	// BaseURL é o repositório consultado. Padrão: a API do GitHub do
	// draftboard.
	BaseURL string
	// Destino é o caminho do binário a substituir. Padrão: Executavel().
	Destino string
	// Atual é a Info do binário em execução. Padrão: Atual().
	Atual Info
	// SO e Arquitetura escolhem o Ativo. Padrão: runtime.GOOS/GOARCH.
	SO          string
	Arquitetura string
	// Cliente faz as requisições. Padrão: http.Client com timeout de 90s.
	Cliente *http.Client
	// TetoBytes é o tamanho máximo do binário descompactado. Padrão: 64 MiB.
	TetoBytes int64
}

// Lancamento é o Lançamento remoto já interpretado e validado: a Versão existe,
// o Ativo desta plataforma existe, e as somas existem.
type Lancamento struct {
	// Versao é a tag do Lançamento, verbatim.
	Versao string
	// NomeAtivo é o nome do Ativo desta plataforma.
	NomeAtivo string
	// URLAtivo e URLSomas são os endereços de download.
	URLAtivo string
	URLSomas string
	// Bytes é o tamanho do Ativo comprimido, como publicado.
	Bytes int64
}

// Ativo devolve o nome do Ativo de uma plataforma. A tag entra verbatim, com o
// "v" e tudo: a função é concatenação pura, sem normalização, para que não
// exista uma segunda regra capaz de divergir do que o workflow publica.
func Ativo(tag, so, arquitetura string) string {
	return "draftboard_" + tag + "_" + so + "_" + arquitetura + ".tar.gz"
}

// Verifica consulta o último Lançamento e diz se ele é mais novo que o atual.
// Não escreve nada em disco e não toca no binário.
//
// maisNova é verdadeiro quando a Versão atual não é reconhecível — o caso do
// binário "dev", instalado por `go install`. A decisão é deliberada: exigir uma
// flag extra no primeiro uso real do verbo seria hostil, e a garantia contra
// binário adulterado é a soma SHA-256, não a comparação de Versão.
func Verifica(o Opcoes) (Lancamento, bool, error) {
	o = comPadroesDeConsulta(o)
	var l Lancamento
	corpo, err := baixa(o, o.BaseURL+"/releases/latest", "application/vnd.github+json")
	if err != nil {
		if errors.Is(err, errNaoEncontrado) {
			return l, false, errors.New("nenhum lançamento publicado ainda")
		}
		return l, false, fmt.Errorf("não foi possível consultar os lançamentos: %w", err)
	}
	defer corpo.Close()

	var remoto struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(corpo, tetoDaResposta)).Decode(&remoto); err != nil {
		return l, false, fmt.Errorf("não foi possível interpretar a resposta dos lançamentos: %w", err)
	}
	if remoto.TagName == "" {
		return l, false, errors.New("nenhum lançamento publicado ainda")
	}
	if _, ok := interpreta(remoto.TagName); !ok {
		return l, false, fmt.Errorf("não reconheço a versão do lançamento %q", remoto.TagName)
	}

	l.Versao = remoto.TagName
	l.NomeAtivo = Ativo(remoto.TagName, o.SO, o.Arquitetura)
	// A lista de plataformas não é embutida no cliente: ele calcula o nome
	// esperado e o procura entre os Ativos publicados. Publicar uma plataforma
	// nova passa a funcionar sem tocar neste código.
	for _, a := range remoto.Assets {
		switch a.Name {
		case l.NomeAtivo:
			l.URLAtivo = a.URL
			l.Bytes = a.Size
		case nomeDasSomas:
			l.URLSomas = a.URL
		}
	}
	if l.URLAtivo == "" {
		return Lancamento{}, false, fmt.Errorf(
			"o lançamento %s não publica um ativo para %s/%s (esperado %s)",
			l.Versao, o.SO, o.Arquitetura, l.NomeAtivo)
	}
	if l.URLSomas == "" {
		return Lancamento{}, false, fmt.Errorf("o lançamento %s não publica %s", l.Versao, nomeDasSomas)
	}

	ordem, ok := Compara(o.Atual.Versao, l.Versao)
	return l, !ok || ordem < 0, nil
}

// somaEsperada baixa o arquivo de somas do Lançamento e devolve a soma do
// Ativo desta plataforma.
func somaEsperada(o Opcoes, l Lancamento) (string, error) {
	corpo, err := baixa(o, l.URLSomas, "application/octet-stream")
	if err != nil {
		return "", fmt.Errorf("não foi possível baixar %s: %w", nomeDasSomas, err)
	}
	defer corpo.Close()
	texto, err := io.ReadAll(io.LimitReader(corpo, tetoDaResposta))
	if err != nil {
		return "", fmt.Errorf("não foi possível ler %s: %w", nomeDasSomas, err)
	}
	return somaDe(string(texto), l.NomeAtivo)
}

// somaDe extrai a soma de um nome de Ativo num arquivo no formato do
// sha256sum. Um nome que aparece zero ou duas-ou-mais vezes é Lançamento
// quebrado, não empate a resolver por sorteio: as duas situações são erro.
func somaDe(texto, nome string) (string, error) {
	var soma string
	achados := 0
	for _, linha := range strings.Split(texto, "\n") {
		campos := strings.Fields(linha)
		if len(campos) != 2 || campos[1] != nome {
			continue
		}
		if !ehSoma(campos[0]) {
			return "", fmt.Errorf("%s tem uma soma inválida para %s: %q", nomeDasSomas, nome, campos[0])
		}
		soma = campos[0]
		achados++
	}
	if achados != 1 {
		return "", fmt.Errorf("%s não tem exatamente uma linha para %s", nomeDasSomas, nome)
	}
	return soma, nil
}

// ehSoma reporta se s é uma soma SHA-256 em hexadecimal minúsculo.
func ehSoma(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// errNaoEncontrado marca um 404, que na consulta de Lançamento significa
// "nenhum Lançamento publicado".
var errNaoEncontrado = errors.New("não encontrado")

// baixa faz a requisição e devolve o corpo, que o chamador fecha. Só devolve
// corpo em resposta 200.
func baixa(o Opcoes, url, aceita string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("endereço inválido %q: %w", url, err)
	}
	req.Header.Set("Accept", aceita)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	// O GitHub responde 403 a requisição sem User-Agent.
	req.Header.Set("User-Agent", "draftboard/"+o.Atual.Versao)
	resp, err := o.Cliente.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNaoEncontrado
	}
	limitado := resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests
	if limitado && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		return nil, errors.New("limite de requisições da API do GitHub atingido; tente de novo mais tarde")
	}
	return nil, fmt.Errorf("resposta %s de %s", resp.Status, url)
}

// comPadroesDeConsulta preenche os campos vazios que a consulta usa. Não toca
// em Destino: consultar não abre o binário.
func comPadroesDeConsulta(o Opcoes) Opcoes {
	if o.BaseURL == "" {
		o.BaseURL = baseURLPadrao
	}
	o.BaseURL = strings.TrimSuffix(o.BaseURL, "/")
	if o.Atual == (Info{}) {
		o.Atual = Atual()
	}
	if o.SO == "" {
		o.SO = runtime.GOOS
	}
	if o.Arquitetura == "" {
		o.Arquitetura = runtime.GOARCH
	}
	if o.Cliente == nil {
		o.Cliente = &http.Client{Timeout: 90 * time.Second}
	}
	if o.TetoBytes == 0 {
		o.TetoBytes = tetoBytesPadrao
	}
	return o
}
