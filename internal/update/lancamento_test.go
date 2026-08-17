package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// As fixtures de lançamento trazem %BASE% no lugar do endereço do servidor,
// que só existe em tempo de teste.
const marcadorDaBase = "%BASE%"

// Toda a suíte fixa a plataforma, para que o ativo escolhido não dependa da
// máquina que roda os testes.
const (
	soDeTeste          = "linux"
	arquiteturaDeTeste = "amd64"
	ativoDeTeste       = "draftboard_v1.4.0_linux_amd64.tar.gz"
)

// servidor sobe um servidor local que imita a parte da API do GitHub que o
// pacote consome: `/releases/latest` devolve a fixture, e `/download/<nome>`
// devolve um dos arquivos dados. Um nome ausente do mapa vira 404, que é o que
// um lançamento incompleto produziria de verdade.
func servidor(t *testing.T, fixture string, arquivos map[string][]byte) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	if fixture != "" {
		bruto, err := os.ReadFile(filepath.Join("..", "..", "testdata", "f7", fixture))
		if err != nil {
			t.Fatalf("não foi possível ler a fixture: %v", err)
		}
		mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(strings.ReplaceAll(string(bruto), marcadorDaBase, srv.URL)))
		})
	}
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		conteudo, ok := arquivos[strings.TrimPrefix(r.URL.Path, "/download/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(conteudo)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// opcoesDe monta as Opcoes apontadas para o servidor de teste. Destino fica
// vazio de propósito: quem precisa dele aponta para um t.TempDir().
func opcoesDe(base, versaoAtual string) Opcoes {
	return Opcoes{
		BaseURL:     base,
		Atual:       Info{Versao: versaoAtual, Commit: "abc1234", Data: "2026-08-17T00:00:00Z"},
		SO:          soDeTeste,
		Arquitetura: arquiteturaDeTeste,
	}
}

// entrada é um membro de tar, para montar ativos válidos e inválidos.
type entrada struct {
	nome     string
	conteudo string
	tipo     byte
}

// tarballFalso monta o ativo bem-formado: um único membro regular chamado
// "draftboard". Devolve os bytes comprimidos e a soma SHA-256 deles — é sobre
// os bytes COMPRIMIDOS que o checksums.txt é calculado.
func tarballFalso(t *testing.T, conteudo string) (ativo []byte, soma string) {
	t.Helper()
	return tarballComEntradas(t, []entrada{{nome: nomeNoTar, conteudo: conteudo, tipo: tar.TypeReg}})
}

func tarballComEntradas(t *testing.T, entradas []entrada) (ativo []byte, soma string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entradas {
		cabecalho := &tar.Header{
			Name:     e.nome,
			Mode:     0o755,
			Size:     int64(len(e.conteudo)),
			Typeflag: e.tipo,
		}
		if err := tw.WriteHeader(cabecalho); err != nil {
			t.Fatalf("não foi possível escrever o cabeçalho de %q: %v", e.nome, err)
		}
		if _, err := tw.Write([]byte(e.conteudo)); err != nil {
			t.Fatalf("não foi possível escrever %q: %v", e.nome, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("não foi possível fechar o tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("não foi possível fechar o gzip: %v", err)
	}
	h := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(h[:])
}

// somasDe monta um checksums.txt no formato do sha256sum.
func somasDe(pares map[string]string) []byte {
	var b strings.Builder
	for nome, soma := range pares {
		b.WriteString(soma + "  " + nome + "\n")
	}
	return []byte(b.String())
}

func TestVerificaDetectaVersaoMaisNova(t *testing.T) {
	srv := servidor(t, "releases-latest.json", nil)
	l, maisNova, err := Verifica(opcoesDe(srv.URL, "v1.3.0"))
	if err != nil {
		t.Fatalf("Verifica devolveu erro: %v", err)
	}
	if !maisNova {
		t.Errorf("v1.4.0 não foi reportada como mais nova que v1.3.0")
	}
	if l.Versao != "v1.4.0" {
		t.Errorf("Versao = %q, esperado \"v1.4.0\"", l.Versao)
	}
	if l.NomeAtivo != ativoDeTeste {
		t.Errorf("NomeAtivo = %q, esperado %q", l.NomeAtivo, ativoDeTeste)
	}
	if !strings.HasSuffix(l.URLAtivo, "/download/"+ativoDeTeste) {
		t.Errorf("URLAtivo = %q", l.URLAtivo)
	}
	if !strings.HasSuffix(l.URLSomas, "/download/"+nomeDasSomas) {
		t.Errorf("URLSomas = %q", l.URLSomas)
	}
	if l.Bytes == 0 {
		t.Errorf("Bytes = 0; o tamanho publicado alimenta a linha de status")
	}
}

func TestVerificaNaoOfereceQuandoAsVersoesSaoIguais(t *testing.T) {
	srv := servidor(t, "releases-latest.json", nil)
	if _, maisNova, err := Verifica(opcoesDe(srv.URL, "v1.4.0")); err != nil || maisNova {
		t.Errorf("Verifica ofereceu a mesma versão: maisNova = %v, err = %v", maisNova, err)
	}
}

// TestVerificaNaoOfereceQuandoAVersaoAtualEMaisNova: quem construiu do main
// não recebe proposta de downgrade. O aviso correspondente é da CLI.
func TestVerificaNaoOfereceQuandoAVersaoAtualEMaisNova(t *testing.T) {
	srv := servidor(t, "releases-latest.json", nil)
	if _, maisNova, err := Verifica(opcoesDe(srv.URL, "v1.5.0")); err != nil || maisNova {
		t.Errorf("Verifica ofereceu um downgrade: maisNova = %v, err = %v", maisNova, err)
	}
}

// TestVerificaOfereceQuandoAVersaoAtualEDev fixa a política do binário sem
// informação de versão, que é o instalado por `go install`: trata como
// desatualizado e segue.
func TestVerificaOfereceQuandoAVersaoAtualEDev(t *testing.T) {
	srv := servidor(t, "releases-latest.json", nil)
	if _, maisNova, err := Verifica(opcoesDe(srv.URL, "dev")); err != nil || !maisNova {
		t.Errorf("Verifica não ofereceu atualização para \"dev\": maisNova = %v, err = %v", maisNova, err)
	}
}

func TestVerificaRecusaLancamentoSemAtivoDaPlataforma(t *testing.T) {
	srv := servidor(t, "releases-latest-sem-ativo.json", nil)
	_, _, err := Verifica(opcoesDe(srv.URL, "v1.3.0"))
	if err == nil {
		t.Fatal("Verifica aceitou um lançamento sem o ativo desta plataforma")
	}
	// A mensagem tem de dizer qual nome faltou, senão não dá para agir.
	if !strings.Contains(err.Error(), ativoDeTeste) {
		t.Errorf("erro não nomeia o ativo esperado: %v", err)
	}
}

func TestVerificaRecusaLancamentoSemSomas(t *testing.T) {
	srv := servidor(t, "releases-latest-sem-checksums.json", nil)
	_, _, err := Verifica(opcoesDe(srv.URL, "v1.3.0"))
	if err == nil || !strings.Contains(err.Error(), nomeDasSomas) {
		t.Fatalf("Verifica aceitou lançamento sem %s: %v", nomeDasSomas, err)
	}
}

func TestVerificaRecusaTagIlegivel(t *testing.T) {
	srv := servidor(t, "releases-latest-tag-invalida.json", nil)
	_, _, err := Verifica(opcoesDe(srv.URL, "v1.3.0"))
	if err == nil || !strings.Contains(err.Error(), "latest") {
		t.Fatalf("Verifica aceitou a tag \"latest\": %v", err)
	}
}

// TestVerificaReportaAusenciaDeLancamento cobre os dois jeitos de não haver
// lançamento: 404 na consulta e resposta sem tag.
func TestVerificaReportaAusenciaDeLancamento(t *testing.T) {
	casos := map[string]string{
		"404 na consulta":  "",
		"resposta sem tag": "releases-latest-sem-tag.json",
	}
	for fato, fixture := range casos {
		srv := servidor(t, fixture, nil)
		_, _, err := Verifica(opcoesDe(srv.URL, "v1.3.0"))
		if err == nil || !strings.Contains(err.Error(), "nenhum lançamento publicado") {
			t.Errorf("%s: erro = %v, esperado \"nenhum lançamento publicado ainda\"", fato, err)
		}
	}
}

func TestVerificaReportaLimiteDeRequisicoes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	_, _, err := Verifica(opcoesDe(srv.URL, "v1.3.0"))
	if err == nil || !strings.Contains(err.Error(), "limite de requisições") {
		t.Fatalf("erro = %v, esperado a mensagem de limite de requisições", err)
	}
}

// TestVerificaNaoEscreveEmDisco é a garantia que faz `update --check` ser
// seguro de rodar em qualquer lugar.
func TestVerificaNaoEscreveEmDisco(t *testing.T) {
	dir := t.TempDir()
	srv := servidor(t, "releases-latest.json", nil)
	o := opcoesDe(srv.URL, "v1.3.0")
	o.Destino = filepath.Join(dir, "draftboard")
	if _, _, err := Verifica(o); err != nil {
		t.Fatalf("Verifica devolveu erro: %v", err)
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("não foi possível listar %s: %v", dir, err)
	}
	if len(entradas) != 0 {
		t.Errorf("Verifica deixou %d entrada(s) em disco: %v", len(entradas), entradas)
	}
}
