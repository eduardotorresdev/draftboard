package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestRepeticaoDentroDoComponenteUsaOEspacoLocal fixa que o intervalo de uma
// Repetição escrita num Componente é medido no espaço local de 0 a 100, e só
// depois convertido pela caixa da Instância — e que `n: 1` produz um clone.
func TestRepeticaoDentroDoComponenteUsaOEspacoLocal(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "repeticao-em-componente.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	casos := []struct{ caminho, queria string }{
		// Passo local de (10 + 10)% sobre uma caixa de 200 px: 40 px.
		{"bloco/barra#0", "0,0 20x100"},
		{"bloco/barra#1", "40,0 20x100"},
		{"bloco/barra#2", "80,0 20x100"},
		// n: 1 materializa um único clone, sem deslocamento.
		{"bloco/solo#0", "120,0 80x100"},
	}
	for _, c := range casos {
		if linha := linhaDe(t, stdout, c.caminho); !strings.Contains(linha, c.queria) {
			t.Errorf("%s: queria %q, obteve %q", c.caminho, c.queria, linha)
		}
	}
	if strings.Contains(stdout, "solo#1") {
		t.Error("Repetição com n: 1 materializou mais de um clone")
	}
	conferaGolden(t, "repeticao-em-componente.txt", stdout)
}

// TestIdentidadeDoComponenteEOCaminhoAbsoluto fixa que o Componente é
// identificado pelo caminho absoluto: duas grafias do mesmo arquivo são o mesmo
// Componente, e dois arquivos de mesmo nome em pastas diferentes não são.
func TestIdentidadeDoComponenteEOCaminhoAbsoluto(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "identidade.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	// "./emblema.yaml" e "./pecas/../emblema.yaml" são o mesmo Componente:
	// mesmo conteúdo e mesma Origem, apesar da grafia diferente.
	if linha := linhaDe(t, stdout, "rodeio/faixa"); !strings.Contains(linha, "de=emblema.yaml") {
		t.Errorf("grafias diferentes do mesmo Componente divergiram: %s", linha)
	}
	// O homônimo em pecas/ tem conteúdo próprio: o cache não pode confundir
	// os dois pelo nome do arquivo.
	linha := linhaDe(t, stdout, "homonimo/bolha")
	if !strings.Contains(linha, "circulo") || !strings.Contains(linha, "de=pecas/emblema.yaml") {
		t.Errorf("homônimos em pastas diferentes colidiram: %s", linha)
	}
	conferaGolden(t, "identidade.txt", stdout)
}

// TestOrigemEhSempreRelativa fixa o que `scene.Elemento.Origem` promete: mesmo
// com o `use` escrito em caminho absoluto, o `de=` da árvore sai relativo.
func TestOrigemEhSempreRelativa(t *testing.T) {
	componente, err := filepath.Abs(filepath.Join("testdata", "f2", "emblema.yaml"))
	if err != nil {
		t.Fatalf("caminho do Componente: %v", err)
	}
	pasta := t.TempDir()
	documento := "documento.yaml"
	conteudo := "frames:\n  - name: tela\n    w: 100\n    h: 100\n    layers:\n      - name: base\n" +
		"        elements:\n          - use: \"" + componente + "\"\n            box: {x: 0, y: 0, w: 50, h: 50}\n"
	if err := os.WriteFile(filepath.Join(pasta, documento), []byte(conteudo), 0o644); err != nil {
		t.Fatalf("escrevendo o Documento: %v", err)
	}
	t.Chdir(pasta)
	codigo, stdout, stderr := executa("inspect", documento)
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	linha := linhaDe(t, stdout, "e0/faixa")
	if strings.Contains(linha, "de="+string(filepath.Separator)) {
		t.Errorf("a Origem vazou caminho absoluto: %s", linha)
	}
	if !strings.HasSuffix(linha, "emblema.yaml") {
		t.Errorf("a Origem não aponta o Componente: %s", linha)
	}
}

// --- Unicidade do caminho ---------------------------------------------------

// TestCaminhoNuncaSeRepeteNoFrame cobre a colisão entre as duas regras de
// segmento — um Elemento com `id: corpo` e um Slot chamado `corpo` no mesmo
// espaço — e a precedência do nome do Slot sobre um id declarado no mesmo nó.
func TestCaminhoNuncaSeRepeteNoFrame(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "colisao.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	// O Elemento vem antes na ordem de pintura e fica com o caminho puro; a
	// Superfície do Slot homônimo ganha o sufixo.
	if linha := linhaDe(t, stdout, "c/corpo"); !strings.Contains(linha, "0,0 10x10") {
		t.Errorf("o Elemento perdeu o caminho para o Slot homônimo: %s", linha)
	}
	if linha := linhaDe(t, stdout, "c/corpo~2"); !strings.Contains(linha, "20,20 50x50") {
		t.Errorf("o Slot homônimo não foi desambiguado: %s", linha)
	}
	// O segmento de um Slot é o nome do Slot, mesmo quando o nó declara id.
	linhaDe(t, stdout, "c/pe")
	if strings.Contains(stdout, "c/outro") {
		t.Error("o id do nó venceu o nome do Slot no segmento do caminho")
	}
	// Nenhum caminho pode aparecer duas vezes no Frame.
	vistos := map[string]bool{}
	for _, linha := range strings.Split(stdout, "\n") {
		campos := strings.Fields(linha)
		if len(campos) < 2 || !strings.HasPrefix(campos[0], "c/") {
			continue
		}
		if vistos[campos[0]] {
			t.Errorf("caminho repetido no Frame: %s", campos[0])
		}
		vistos[campos[0]] = true
	}
	conferaGolden(t, "colisao.txt", stderr+stdout)
}

// TestCaminhoEUnicoPorFrame fixa o alcance da regra: a unicidade é dentro de
// cada Frame, então dois Frames que instanciam o mesmo Componente com o mesmo
// id têm os mesmos caminhos, sem sufixo nenhum.
func TestCaminhoEUnicoPorFrame(t *testing.T) {
	naPastaDeReuso(t)
	codigo, stdout, stderr := executa("inspect", "dois-frames.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if strings.Contains(stdout, "~") {
		t.Errorf("caminho desambiguado entre Frames diferentes:\n%s", stdout)
	}
	if n := strings.Count(stdout, "c/faixa "); n != 2 {
		t.Errorf("o caminho apareceu %d vezes, queria uma em cada Frame:\n%s", n, stdout)
	}
	conferaGolden(t, "dois-frames.txt", stdout)
}

// --- Tetos de materialização ------------------------------------------------

// TestValidateRecusaMaterializacaoAcimaDoTeto fixa os dois tetos do contrato: o
// de clones por Repetição e o de Elementos por Frame. Os dois existem porque
// Repetições encadeadas por Componentes multiplicam, e a recusa tem que ser
// imediata em vez de consumir memória até o processo morrer.
func TestValidateRecusaMaterializacaoAcimaDoTeto(t *testing.T) {
	casos := []struct{ nome, fixture, golden string }{
		{"clones acima do teto", "clones-demais.yaml", "clones-demais.txt"},
		{"um clone além do teto", "clones-1001.yaml", "clones-1001.txt"},
		{"clones que estouram o int64", "clones-1e19.yaml", "clones-1e19.txt"},
		{"clones em ordem de grandeza absurda", "clones-1e30.yaml", "clones-1e30.txt"},
		{"Elementos acima do teto do Frame", "dez-mil-e-um.yaml", "dez-mil-e-um.txt"},
		{"Repetições multiplicadas por Componentes", "multiplica.yaml", "multiplica.txt"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			naPastaDeReuso(t)
			inicio := time.Now()
			codigo, stdout, stderr := executa("validate", c.fixture)
			if decorrido := time.Since(inicio); decorrido > 10*time.Second {
				t.Errorf("a recusa levou %v: o teto não está falhando rápido", decorrido)
			}
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

// TestValidateAceitaMaterializacaoNoTeto fixa que os dois tetos são inclusivos.
func TestValidateAceitaMaterializacaoNoTeto(t *testing.T) {
	casos := []struct{ nome, fixture string }{
		{"clones exatamente no teto", "clones-limite.yaml"},
		{"Elementos exatamente no teto do Frame", "dez-mil.yaml"},
		// O teto é por Frame: a soma dos dois passa de 10000, e nenhum
		// deles passa sozinho.
		{"dois Frames abaixo do teto cada um", "dois-frames-cheios.yaml"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			naPastaDeReuso(t)
			codigo, stdout, stderr := executa("validate", c.fixture)
			if codigo != 0 {
				t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("saída = (%q, %q), queria vazia nos dois", stdout, stderr)
			}
		})
	}
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
