package update

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// REGRA DESTA SUÍTE: nenhum teste chama Executavel(), e Opcoes.Destino sempre
// aponta para dentro de um t.TempDir(). Um teste que deixe o Destino no padrão
// substituiria o binário de teste em execução.
//
// A asserção que mais importa aqui é dupla e se repete: em toda falha, os bytes
// originais do destino sobrevivem E o diretório continua com exatamente uma
// entrada. A primeira metade prova que nada foi trocado; a segunda, que nenhum
// temporário ficou para trás.

const conteudoOriginal = "binário antigo\n"

// preparaDestino cria um destino com conteúdo e modo conhecidos e devolve o
// diretório e o caminho.
func preparaDestino(t *testing.T, modo os.FileMode) (dir, destino string) {
	t.Helper()
	dir = t.TempDir()
	destino = filepath.Join(dir, "draftboard")
	if err := os.WriteFile(destino, []byte(conteudoOriginal), modo); err != nil {
		t.Fatalf("não foi possível criar o destino: %v", err)
	}
	if err := os.Chmod(destino, modo); err != nil {
		t.Fatalf("não foi possível ajustar o modo do destino: %v", err)
	}
	return dir, destino
}

// resolvido devolve o caminho com os links simbólicos resolvidos, que é onde a
// troca de verdade acontece.
func resolvido(t *testing.T, caminho string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(caminho)
	if err != nil {
		t.Fatalf("não foi possível resolver %s: %v", caminho, err)
	}
	return real
}

// lancamentoDe sobe o servidor com o ativo e as somas dados e devolve as Opcoes
// e o Lancamento já validados por Verifica.
func lancamentoDe(t *testing.T, ativo, somas []byte) (Opcoes, Lancamento) {
	t.Helper()
	srv := servidor(t, "releases-latest.json", map[string][]byte{
		ativoDeTeste: ativo,
		nomeDasSomas: somas,
	})
	o := opcoesDe(srv.URL, "v1.3.0")
	l, _, err := Verifica(o)
	if err != nil {
		t.Fatalf("Verifica devolveu erro: %v", err)
	}
	return o, l
}

// conferePreservado afirma as duas metades: destino intacto e nenhum
// temporário.
func conferePreservado(t *testing.T, dir, destino string) {
	t.Helper()
	bytesAtuais, err := os.ReadFile(destino)
	if err != nil {
		t.Fatalf("não foi possível ler o destino: %v", err)
	}
	if string(bytesAtuais) != conteudoOriginal {
		t.Errorf("o destino foi alterado: %q", string(bytesAtuais))
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("não foi possível listar %s: %v", dir, err)
	}
	if len(entradas) != 1 {
		var nomes []string
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Errorf("o diretório tem %d entradas, esperado 1: %v", len(entradas), nomes)
	}
}

func TestAplicaTrocaOBinarioEPreservaOModo(t *testing.T) {
	const novo = "binário novo\n"
	ativo, soma := tarballFalso(t, novo)
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))
	// Modo com bit de execução mas fora do 0755, para provar que o modo é
	// copiado do original em vez de chutado.
	dir, destino := preparaDestino(t, 0o750)
	o.Destino = destino

	var progresso bytes.Buffer
	escrito, err := Aplica(o, l, &progresso)
	if err != nil {
		t.Fatalf("Aplica devolveu erro: %v", err)
	}
	// Em macOS o próprio t.TempDir() mora atrás de um link (/var ->
	// /private/var), então a comparação é contra o caminho resolvido.
	if escrito != resolvido(t, destino) {
		t.Errorf("Aplica devolveu %q, esperado %q", escrito, resolvido(t, destino))
	}
	conteudo, err := os.ReadFile(destino)
	if err != nil {
		t.Fatalf("não foi possível ler o destino: %v", err)
	}
	if string(conteudo) != novo {
		t.Errorf("conteúdo do destino = %q, esperado %q", string(conteudo), novo)
	}
	info, err := os.Stat(destino)
	if err != nil {
		t.Fatalf("não foi possível inspecionar o destino: %v", err)
	}
	if info.Mode().Perm() != 0o750 {
		t.Errorf("modo = %v, esperado -rwxr-x---", info.Mode().Perm())
	}
	// Nenhum temporário sobrevive à troca bem-sucedida.
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("não foi possível listar %s: %v", dir, err)
	}
	if len(entradas) != 1 {
		t.Errorf("o diretório tem %d entradas depois da troca, esperado 1", len(entradas))
	}
	if !strings.Contains(progresso.String(), ativoDeTeste) {
		t.Errorf("a linha de status não nomeia o ativo baixado: %q", progresso.String())
	}
}

// TestAplicaNaoTocaODestinoQuandoASomaNaoConfere é o teste mais importante do
// pacote: é ele que sustenta a invariante de que nada é renomeado antes de a
// soma conferir.
func TestAplicaNaoTocaODestinoQuandoASomaNaoConfere(t *testing.T) {
	ativo, _ := tarballFalso(t, "binário novo\n")
	somaErrada := strings.Repeat("ab", 32)
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: somaErrada}))
	dir, destino := preparaDestino(t, 0o755)
	o.Destino = destino

	_, err := Aplica(o, l, nil)
	if err == nil {
		t.Fatal("Aplica aceitou um ativo com soma errada")
	}
	if !strings.Contains(err.Error(), "não confere") || !strings.Contains(err.Error(), "não foi alterado") {
		t.Errorf("a mensagem não diz que a soma falhou e que o binário está intacto: %v", err)
	}
	conferePreservado(t, dir, destino)
}

func TestAplicaNaoTocaODestinoQuandoODownloadCortaNoMeio(t *testing.T) {
	ativo, soma := tarballFalso(t, strings.Repeat("conteúdo do binário novo\n", 200))
	cortado := ativo[:len(ativo)/2]
	// A soma publicada é a do ativo INTEIRO, como no lançamento de verdade.
	o, l := lancamentoDe(t, cortado, somasDe(map[string]string{ativoDeTeste: soma}))
	dir, destino := preparaDestino(t, 0o755)
	o.Destino = destino

	if _, err := Aplica(o, l, nil); err == nil {
		t.Fatal("Aplica aceitou um download cortado no meio")
	}
	conferePreservado(t, dir, destino)
}

func TestAplicaRecusaAtivoMalFormado(t *testing.T) {
	casos := map[string][]entrada{
		"mais de uma entrada": {
			{nome: nomeNoTar, conteudo: "binário\n", tipo: 0x30},
			{nome: "LICENSE", conteudo: "licença\n", tipo: 0x30},
		},
		"nome inesperado": {
			{nome: "draftboard-linux", conteudo: "binário\n", tipo: 0x30},
		},
		"nome com diretório": {
			{nome: "dist/draftboard", conteudo: "binário\n", tipo: 0x30},
		},
		"nenhuma entrada": {},
	}
	for fato, entradas := range casos {
		t.Run(fato, func(t *testing.T) {
			ativo, soma := tarballComEntradas(t, entradas)
			o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))
			dir, destino := preparaDestino(t, 0o755)
			o.Destino = destino
			if _, err := Aplica(o, l, nil); err == nil {
				t.Fatalf("Aplica aceitou ativo com %s", fato)
			}
			conferePreservado(t, dir, destino)
		})
	}
}

func TestAplicaRecusaAtivoAcimaDoTeto(t *testing.T) {
	ativo, soma := tarballFalso(t, strings.Repeat("x", 4096))
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))
	dir, destino := preparaDestino(t, 0o755)
	o.Destino = destino
	o.TetoBytes = 1024

	_, err := Aplica(o, l, nil)
	if err == nil || !strings.Contains(err.Error(), "descompactado") {
		t.Fatalf("erro = %v, esperado recusa pelo teto de tamanho", err)
	}
	conferePreservado(t, dir, destino)
}

// TestAplicaSubstituiOAlvoDoLinkSimbolicoENaoOLink: renomear sobre o caminho do
// link trocaria o link por um arquivo comum e orfanaria o binário real. A troca
// tem de acontecer no arquivo apontado.
func TestAplicaSubstituiOAlvoDoLinkSimbolicoENaoOLink(t *testing.T) {
	const novo = "binário novo\n"
	ativo, soma := tarballFalso(t, novo)
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))

	dir := t.TempDir()
	real := filepath.Join(dir, "draftboard-1.3.0")
	link := filepath.Join(dir, "draftboard")
	if err := os.WriteFile(real, []byte(conteudoOriginal), 0o755); err != nil {
		t.Fatalf("não foi possível criar o arquivo real: %v", err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("não foi possível criar o link: %v", err)
	}
	o.Destino = link

	escrito, err := Aplica(o, l, nil)
	if err != nil {
		t.Fatalf("Aplica devolveu erro: %v", err)
	}
	if escrito != resolvido(t, real) {
		t.Errorf("Aplica devolveu %q, esperado o alvo do link %q", escrito, resolvido(t, real))
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("não foi possível inspecionar o link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("o link simbólico foi substituído por um arquivo comum")
	}
	conteudo, err := os.ReadFile(real)
	if err != nil {
		t.Fatalf("não foi possível ler o arquivo real: %v", err)
	}
	if string(conteudo) != novo {
		t.Errorf("o alvo do link não foi atualizado: %q", string(conteudo))
	}
}

// TestAplicaAvisaQuandoODestinoNaoTemBitDeExecucao: sem o resgate para 0755, o
// substituto sairia sem permissão de execução e o usuário ficaria com um
// arquivo inútil no $PATH.
func TestAplicaAvisaQuandoODestinoNaoTemBitDeExecucao(t *testing.T) {
	ativo, soma := tarballFalso(t, "binário novo\n")
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))
	_, destino := preparaDestino(t, 0o644)
	o.Destino = destino

	var progresso bytes.Buffer
	if _, err := Aplica(o, l, &progresso); err != nil {
		t.Fatalf("Aplica devolveu erro: %v", err)
	}
	info, err := os.Stat(destino)
	if err != nil {
		t.Fatalf("não foi possível inspecionar o destino: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("modo = %v, esperado 0755", info.Mode().Perm())
	}
	if !strings.Contains(progresso.String(), "aviso: ") {
		t.Errorf("Aplica não avisou sobre o bit de execução: %q", progresso.String())
	}
}

// TestAplicaReportaFaltaDePermissao: a mensagem tem de dizer o que fazer, não
// devolver o erro cru do sistema.
func TestAplicaReportaFaltaDePermissao(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("rodando como root: permissão de diretório não se aplica")
	}
	ativo, soma := tarballFalso(t, "binário novo\n")
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))
	dir, destino := preparaDestino(t, 0o755)
	o.Destino = destino
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("não foi possível tornar o diretório somente-leitura: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	_, err := Aplica(o, l, nil)
	if err == nil {
		t.Fatal("Aplica gravou num diretório sem permissão de escrita")
	}
	if !strings.Contains(err.Error(), "sem permissão de escrita") {
		t.Errorf("a mensagem não é acionável: %v", err)
	}
	conteudo, lerErr := os.ReadFile(destino)
	if lerErr != nil {
		t.Fatalf("não foi possível ler o destino: %v", lerErr)
	}
	if string(conteudo) != conteudoOriginal {
		t.Errorf("o destino foi alterado: %q", string(conteudo))
	}
}

func TestAplicaRecusaDestinoInexistente(t *testing.T) {
	ativo, soma := tarballFalso(t, "binário novo\n")
	o, l := lancamentoDe(t, ativo, somasDe(map[string]string{ativoDeTeste: soma}))
	o.Destino = filepath.Join(t.TempDir(), "nao-existe")
	if _, err := Aplica(o, l, nil); err == nil {
		t.Fatal("Aplica aceitou um destino inexistente")
	}
}
