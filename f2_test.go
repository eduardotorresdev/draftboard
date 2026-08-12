package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Testes do reuso: Componente, Instância, Slot e Repetição. Como em F1, o seam
// é a CLI: nenhum teste conhece a estrutura interna da resolução.

// naPastaDeReuso leva o teste para dentro de testdata/f2, para que as mensagens
// da CLI citem o Documento pelo nome, sem diretório.
func naPastaDeReuso(t *testing.T) {
	t.Helper()
	pasta, err := filepath.Abs(filepath.Join("testdata", "f2"))
	if err != nil {
		t.Fatalf("caminho das fixtures: %v", err)
	}
	t.Chdir(pasta)
}

// linhaDe devolve a primeira linha da saída que começa com o caminho dado.
func linhaDe(t *testing.T, saida, caminho string) string {
	t.Helper()
	for _, linha := range strings.Split(saida, "\n") {
		if strings.HasPrefix(strings.TrimSpace(linha), caminho+" ") {
			return strings.TrimSpace(linha)
		}
	}
	t.Fatalf("nenhuma linha para %q na árvore:\n%s", caminho, saida)
	return ""
}

// --- Instância --------------------------------------------------------------

// TestInstanciaReescalaEmCaixaDeProporcaoDiferente fixa a conversão do espaço
// local do Componente para uma caixa que não é quadrada: os fatores dos dois
// eixos são independentes, o deslocamento da caixa entra na conta, e o Círculo
// continua redondo porque o diâmetro usa só o fator de largura.
func TestInstanciaReescalaEmCaixaDeProporcaoDiferente(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "reescala.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, queria vazio", stderr)
	}
	// A caixa é 320x100 px na origem 40,40: o Retângulo que ocupa o espaço
	// local inteiro tem que cair exatamente sobre ela.
	if linha := linhaDe(t, stdout, "cartao/fundo"); !strings.Contains(linha, "40,40 320x100") {
		t.Errorf("o Componente não cobriu a caixa da Instância: %s", linha)
	}
	// Diâmetro de 20 sobre uma caixa de 320 px de largura: 64 px nos dois
	// eixos. Usar o fator de altura daria 20x20, e o Círculo viraria elipse.
	if linha := linhaDe(t, stdout, "cartao/avatar"); !strings.Contains(linha, "circulo 72,50 64x64") {
		t.Errorf("o Círculo perdeu a redondeza na reescala: %s", linha)
	}
	conferaGolden(t, "reescala.txt", stdout)
}

// TestInstanciaDentroDeInstancia cobre o Componente que instancia outro
// Componente, com o caminho resolvido relativo ao arquivo que referencia, e o
// caminho da árvore ganhando um segmento por nível de Componente.
func TestInstanciaDentroDeInstancia(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "aninhado.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	linha := linhaDe(t, stdout, "painel/selo/marca")
	if !strings.Contains(linha, "de=pecas/selo.yaml") {
		t.Errorf("a Origem do Elemento não aponta o Componente: %s", linha)
	}
	conferaGolden(t, "aninhado.txt", stdout)
}

// TestElevacaoAtravessaAFronteiraDoComponente fixa a story central do reuso: o
// Tom de um Elemento do Componente é calculado contra a Superfície real onde a
// Instância foi colocada, não contra o contexto onde o Componente foi escrito.
func TestElevacaoAtravessaAFronteiraDoComponente(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "aninhado.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	casos := []struct{ caminho, queria string }{
		// Elemento do Documento, apoiado no Frame.
		{"moldura", "tom=300 elev=1"},
		// Elemento do Componente, contido por `moldura`.
		{"painel/fundo", "tom=500 elev=2"},
		// Elemento de um Componente dentro do Componente.
		{"painel/selo/marca", "tom=700 elev=3"},
	}
	for _, c := range casos {
		if linha := linhaDe(t, stdout, c.caminho); !strings.Contains(linha, c.queria) {
			t.Errorf("%s: queria %q, obteve %q", c.caminho, c.queria, linha)
		}
	}
}

// TestUseResolveRelativoAoArquivoQueReferencia prova que a referência é
// resolvida contra o arquivo que a escreve: rodando de um diretório de trabalho
// que não tem nenhuma das fixtures, a cadeia inteira ainda resolve.
func TestUseResolveRelativoAoArquivoQueReferencia(t *testing.T) {
	documento, err := filepath.Abs(filepath.Join("testdata", "f2", "aninhado.yaml"))
	if err != nil {
		t.Fatalf("caminho do Documento: %v", err)
	}
	t.Chdir(t.TempDir())
	codigo, stdout, stderr := executa("inspect", documento)
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if !strings.Contains(stdout, "painel/selo/marca") {
		t.Errorf("a cadeia de Componentes não resolveu fora do diretório do Documento:\n%s", stdout)
	}
}

// --- Slot -------------------------------------------------------------------

// TestSlotRecebePreenchimentoPadraoEVazio cobre as quatro situações de um Slot
// numa só árvore: preenchido por Componente, preenchido por Elementos inline,
// sem preenchimento mas com conteúdo padrão, e vazio.
func TestSlotRecebePreenchimentoPadraoEVazio(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "slots.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}

	// Preenchido por Componente.
	if linha := linhaDe(t, stdout, "pagina/cabeca/faixa"); !strings.Contains(linha, "de=emblema.yaml") {
		t.Errorf("o Slot não recebeu o Componente: %s", linha)
	}
	// Preenchido por Elementos inline: o conteúdo é do Documento, sem Origem.
	linha := linhaDe(t, stdout, "pagina/corpo/direita")
	if strings.Contains(linha, "de=") {
		t.Errorf("Elemento inline ganhou Origem de Componente: %s", linha)
	}
	// A região do Slot vira um novo espaço de 0 a 100: `direita` ocupa a
	// metade direita da região do Slot (20,120 360x120), não a metade
	// direita do Componente.
	if !strings.Contains(linha, "200,120 180x120") {
		t.Errorf("a região do Slot não virou um espaço local de 0 a 100: %s", linha)
	}
	// Conteúdo padrão, usado porque ninguém preencheu o Slot.
	if linha := linhaDe(t, stdout, "pagina/rodape/padrao"); !strings.Contains(linha, "20,260 360x60") {
		t.Errorf("o conteúdo padrão do Slot não foi usado: %s", linha)
	}
	// O preenchimento é resolvido contra o arquivo que o escreveu — o
	// Documento —, e não contra a pasta do Componente que declara o Slot.
	if linha := linhaDe(t, stdout, "caixilho/miolo/faixa"); !strings.Contains(linha, "de=emblema.yaml") {
		t.Errorf("o preenchimento não resolveu contra quem o declarou: %s", linha)
	}
	// Slot vazio: Superfície visível, com degrau de Elevação, e aviso.
	if linha := linhaDe(t, stdout, "pagina/livre"); !strings.Contains(linha, "retangulo 20,340 360x40 tom=500 elev=2") {
		t.Errorf("o Slot vazio não virou Superfície vazia: %s", linha)
	}
	queria := `aviso: slots.yaml: frames[0].layers[0].elements[0] -> ./layout.yaml: elements[4]: ` +
		`Slot "livre" do Componente layout.yaml sem preenchimento e sem conteúdo padrão: renderiza uma Superfície vazia` + "\n"
	if stderr != queria {
		t.Errorf("stderr = %q, queria %q", stderr, queria)
	}
	conferaGolden(t, "slots.txt", stdout)
}

// --- Repetição --------------------------------------------------------------

// TestRepeticaoDeslocaCadaCloneEmTamanhoMaisIntervalo fixa o passo da Repetição
// nos dois eixos, para Elemento e para Instância, e que cada clone calcula sua
// própria Elevação.
func TestRepeticaoDeslocaCadaCloneEmTamanhoMaisIntervalo(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "repeticao.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, queria vazio", stderr)
	}
	casos := []struct{ caminho, queria string }{
		// Retângulo no eixo y: passo de (10 + 2)% de 200 px = 24 px.
		{"item#0", "0,0 20x20"},
		{"item#1", "0,24 20x20"},
		{"item#2", "0,48 20x20"},
		// Círculo no eixo x: passo de (10 + 5)% de 200 px = 30 px.
		{"e1#0", "circulo 40,100 20x20"},
		{"e1#1", "circulo 70,100 20x20"},
		// Instância no eixo x: passo de (box.w 20 + 5)% de 200 px = 50 px.
		{"cartela#0/faixa", "0,140 40x40"},
		{"cartela#1/faixa", "50,140 40x40"},
		{"cartela#2/faixa", "100,140 40x40"},
		// Instância no eixo y, caixa não quadrada: passo de
		// (box.h 15 + 5)% de 200 px = 40 px.
		{"coluna#0/faixa", "120,0 60x30"},
		{"coluna#1/faixa", "120,40 60x30"},
		// Cada clone da Instância calcula sua Elevação normalmente: o
		// Círculo de cada clone se apoia na faixa do seu próprio clone.
		{"cartela#0/ponto", "tom=500 elev=2"},
		{"cartela#1/ponto", "tom=500 elev=2"},
		{"cartela#2/ponto", "tom=500 elev=2"},
	}
	for _, c := range casos {
		if linha := linhaDe(t, stdout, c.caminho); !strings.Contains(linha, c.queria) {
			t.Errorf("%s: queria %q, obteve %q", c.caminho, c.queria, linha)
		}
	}
	conferaGolden(t, "repeticao.txt", stdout)
}

// --- Erros ------------------------------------------------------------------

// TestValidateReprovaReusoInvalido cobre as regras de erro do reuso pela
// mensagem exata que o usuário vê e pelo código de saída 1.
func TestValidateReprovaReusoInvalido(t *testing.T) {
	casos := []struct{ nome, fixture, golden string }{
		{"Componente inexistente", "inexistente.yaml", "inexistente.txt"},
		{"Componente inexistente dentro de outro Componente", "inexistente-aninhado.yaml", "inexistente-aninhado.txt"},
		{"ciclo de referência entre Componentes", "ciclo.yaml", "ciclo.txt"},
		{"profundidade de aninhamento acima do limite", "profundidade-17.yaml", "profundidade-17.txt"},
		{"campo desconhecido dentro do Componente", "campo-no-componente.yaml", "campo-no-componente.txt"},
		{"Documento usado como Componente", "usa-documento.yaml", "usa-documento.txt"},
		{"Componente passado como Documento", "cartao.yaml", "cartao.txt"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			naPastaDeReuso(t)
			codigo, stdout, stderr := executa("validate", c.fixture)
			if codigo != 1 {
				t.Errorf("código de saída = %d, queria 1", codigo)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, queria vazio", stdout)
			}
			conferaGolden(t, c.golden, stderr)
		})
	}
}

// TestValidateAceitaProfundidadeNoLimite fixa que o limite de aninhamento é
// inclusivo: 16 níveis de Componente ainda resolvem.
func TestValidateAceitaProfundidadeNoLimite(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("validate", "profundidade-16.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("saída = (%q, %q), queria vazia nos dois", stdout, stderr)
	}
}

// --- render -----------------------------------------------------------------

// TestRenderDesenhaDocumentoComReuso fecha o caminho completo: um Documento que
// só existe através de Componentes chega até a imagem.
func TestRenderDesenhaDocumentoComReuso(t *testing.T) {
	documento, err := filepath.Abs(filepath.Join("testdata", "f2", "aninhado.yaml"))
	if err != nil {
		t.Fatalf("caminho do Documento: %v", err)
	}
	pasta := t.TempDir()
	t.Chdir(pasta)
	codigo, stdout, stderr := executa("render", documento)
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if stdout != "aninhado-tela.webp\n" {
		t.Errorf("stdout = %q, queria %q", stdout, "aninhado-tela.webp\n")
	}
	if nomes := listaDeArquivos(t, pasta); len(nomes) != 1 || nomes[0] != "aninhado-tela.webp" {
		t.Errorf("arquivos escritos = %v", nomes)
	}
}
