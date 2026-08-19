package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/resolve"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// naPastaDeDiagnostico leva o teste para dentro de testdata/f11, para que as
// mensagens da CLI citem o Documento pelo nome, sem diretório.
func naPastaDeDiagnostico(t *testing.T) {
	t.Helper()
	naPasta(t, "f11")
}

// resolveDeF11 resolve uma fixture de f11 sem passar pela CLI: o espaço de
// projeção não tem observável na árvore do `inspect`.
func resolveDeF11(t *testing.T, fixture string) *scene.Documento {
	t.Helper()
	caminho, err := filepath.Abs(filepath.Join("testdata", "f11", fixture))
	if err != nil {
		t.Fatalf("caminho da fixture %s: %v", fixture, err)
	}
	doc, _, err := resolve.Arquivo(caminho)
	if err != nil {
		t.Fatalf("resolvendo %s: %v", fixture, err)
	}
	return doc
}

// TestEspacoDeProjecaoChegaEmTodoElemento protege o único dado que permite
// desfazer a projeção e devolver ao autor uma porcentagem em vez de um pixel.
// Um Elemento sem espaço faria o diagnóstico dividir por zero e sugerir um `w`
// infinito no arquivo de quem só escreveu um Retângulo estreito.
func TestEspacoDeProjecaoChegaEmTodoElemento(t *testing.T) {
	doc := resolveDeF11(t, "espacos.yaml")

	for _, e := range elementos(doc) {
		if e.Espaco.L <= 0 || e.Espaco.A <= 0 {
			t.Errorf("Elemento %q sem espaço de projeção: %+v", e.Caminho, e.Espaco)
		}
	}

	doFrame := scene.Espaco{X: 0, Y: 0, L: 400, A: 200}
	for _, caminho := range []string{"bloco", "bloco/rotulo", "ponto"} {
		if got := porCaminho(t, doc, caminho).Espaco; got != doFrame {
			t.Errorf("espaço de %q = %+v, quer %+v", caminho, got, doFrame)
		}
	}

	// O nó do Componente foi projetado na caixa da Instância, não no Frame:
	// sugerir `w` contra o Frame daria uma porcentagem oito vezes menor que a
	// necessária.
	daInstancia := scene.Espaco{X: 40, Y: 20, L: 200, A: 100}
	if got := porCaminho(t, doc, "e2/fundo").Espaco; got != daInstancia {
		t.Errorf("espaço do nó do Componente = %+v, quer %+v", got, daInstancia)
	}
}

// TestRenderDiagnosticaRotuloCortadoESaiZero é o caso que dá nome à entrega: o
// Rótulo largo demais era cortado em silêncio — o `inspect` dizia que ele
// estava lá e a imagem não o mostrava inteiro. Agora a imagem sai do mesmo
// jeito, com o Aviso e o `w` mínimo ao lado.
func TestRenderDiagnosticaRotuloCortadoESaiZero(t *testing.T) {
	pasta := numaPastaTemporariaDe(t, "f11", "largo.yaml")

	codigo, stdout, stderr := executa("render", "largo.yaml")
	if codigo != 0 {
		t.Errorf("código de saída = %d, queria 0: o Rótulo cortado é Aviso, não Erro; stderr: %s", codigo, stderr)
	}
	if !strings.Contains(stderr, `o Rótulo "Resultados da busca" não cabe no Retângulo`) {
		t.Errorf("stderr = %q, queria o Aviso do Rótulo cortado", stderr)
	}
	if !strings.Contains(stderr, "use w: ") {
		t.Errorf("stderr = %q, queria a largura sugerida", stderr)
	}
	if !strings.Contains(stdout, ".webp") {
		t.Errorf("stdout = %q, queria o caminho da imagem escrita", stdout)
	}
	if nomes := listaDeArquivos(t, pasta); len(nomes) != 2 {
		t.Errorf("arquivos = %v, queria o Documento mais a imagem", nomes)
	}
}

// TestOsQuatroComandosDiagnosticamESaemUm fecha a natureza do Erro de
// diagnóstico: ele descreve um desenho que já existe e está errado, então o
// comando faz o seu trabalho e só o código de saída muda. Os quatro verbos
// respondem igual, porque o artefato diagnosticado é o Documento.
func TestOsQuatroComandosDiagnosticamESaemUm(t *testing.T) {
	casos := []struct {
		nome    string
		args    []string
		escreve bool
	}{
		{"render", []string{"render", "no-componente.yaml"}, true},
		{"board", []string{"board", "no-componente.yaml"}, true},
		{"inspect", []string{"inspect", "no-componente.yaml"}, false},
		{"validate", []string{"validate", "no-componente.yaml"}, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			pasta := numaPastaTemporariaDe(t, "f11", "no-componente.yaml", "bloco-rotulado.yaml")

			codigo, stdout, stderr := executa(c.args...)
			if codigo != 1 {
				t.Errorf("código de saída = %d, queria 1; stderr: %s", codigo, stderr)
			}
			if !strings.Contains(stderr, "o Retângulo vem de um Componente, e alargá-lo lá muda todas as Instâncias") {
				t.Errorf("stderr = %q, queria a razão do Componente", stderr)
			}
			escreveu := len(listaDeArquivos(t, pasta)) > 2
			if escreveu != c.escreve {
				t.Errorf("escreveu = %v, queria %v: o Erro de diagnóstico não aborta o comando", escreveu, c.escreve)
			}
			if c.nome == "inspect" && !strings.Contains(stdout, "retangulo") {
				t.Errorf("stdout = %q, queria a árvore impressa apesar do Erro", stdout)
			}
		})
	}
}

// TestFixConsertaOWEImprimeAArvoreCorrigida: consertar e imprimir a árvore
// velha diria ao agente que o conserto não aconteceu.
func TestFixConsertaOWEImprimeAArvoreCorrigida(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "largo.yaml")

	codigo, stdout, stderr := executa("inspect", "--fix", "largo.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if !strings.Contains(stderr, "frames[0].layers[0].elements[0]: w 20 → 33") {
		t.Errorf("stderr = %q, queria a linha de troca", stderr)
	}
	if strings.Contains(stderr, "não cabe no Retângulo") {
		t.Errorf("stderr = %q: o diagnóstico da árvore corrigida ainda acusa o corte", stderr)
	}
	if !strings.Contains(stdout, "132x45") {
		t.Errorf("stdout = %q, queria a árvore com a geometria JÁ corrigida", stdout)
	}
	depois, err := os.ReadFile("largo.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	if !strings.Contains(string(depois), "w: 33") {
		t.Errorf("Documento gravado:\n%s\nqueria o w alargado", depois)
	}

	// Rodar de novo não acha nada: o `w` sugerido conserta de primeira, e um
	// segundo `--fix` não pode nem tocar no arquivo.
	antesDaSegunda, err := os.Stat("largo.yaml")
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	codigo, _, stderr = executa("inspect", "--fix", "largo.yaml")
	if codigo != 0 || strings.Contains(stderr, "→") {
		t.Errorf("segunda passada: código = %d, stderr = %q; queria nada a consertar", codigo, stderr)
	}
	depoisDaSegunda, err := os.Stat("largo.yaml")
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !depoisDaSegunda.ModTime().Equal(antesDaSegunda.ModTime()) {
		t.Error("a segunda passada reescreveu o arquivo sem ter o que consertar")
	}
}

// TestFixConsertaOQueDaEDeixaOResto: um Documento que mistura consertável e não
// consertável não pode travar por causa dos nós que a máquina não sabe
// endereçar — senão um único `rect` sem `w` custaria as outras cinco correções.
func TestFixConsertaOQueDaEDeixaOResto(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "misto.yaml")

	codigo, _, stderr := executa("inspect", "--fix", "misto.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1: sobraram Erros", codigo)
	}
	if n := strings.Count(stderr, "→"); n != 5 {
		t.Errorf("linhas de troca = %d, queria 5; stderr: %s", n, stderr)
	}
	if !strings.Contains(stderr, `o Retângulo não declara "w"`) ||
		!strings.Contains(stderr, "o Retângulo está dentro de uma Repetição") {
		t.Errorf("stderr = %q, queria as duas razões que sobraram", stderr)
	}
	depois, err := os.ReadFile("misto.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	if n := strings.Count(string(depois), "w: 33"); n != 5 {
		t.Errorf("Documento gravado com %d larguras corrigidas, queria 5:\n%s", n, depois)
	}
}

// TestFixNaoEscreveQuandoSoHaErro: sem Aviso não há conserto, e sem conserto o
// Documento do autor não pode nem mudar de mtime.
func TestFixNaoEscreveQuandoSoHaErro(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "sem-w.yaml")
	antes, err := os.ReadFile("sem-w.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	info, err := os.Stat("sem-w.yaml")
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	codigo, _, stderr := executa("inspect", "--fix", "sem-w.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1; stderr: %s", codigo, stderr)
	}
	if strings.Contains(stderr, "→") {
		t.Errorf("stderr = %q: não havia nada a trocar", stderr)
	}
	depois, err := os.ReadFile("sem-w.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	if string(depois) != string(antes) {
		t.Errorf("o Documento mudou:\n%s", depois)
	}
	if novo, err := os.Stat("sem-w.yaml"); err != nil || !novo.ModTime().Equal(info.ModTime()) {
		t.Error("o Documento foi reescrito sem haver conserto")
	}
}

// TestFixImprimeOAvisoNovoDaSegundaResolucao: alargar um Retângulo encostado na
// borda direita cria um problema novo, e escondê-lo faria o `--fix` mostrar
// menos que o `inspect` puro.
func TestFixImprimeOAvisoNovoDaSegundaResolucao(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "na-borda.yaml")

	codigo, _, stderr := executa("inspect", "--fix", "na-borda.yaml")
	if codigo != 0 {
		t.Errorf("código de saída = %d, queria 0: o Aviso novo não é Erro", codigo)
	}
	if !strings.Contains(stderr, "Elemento fora do Frame") {
		t.Errorf("stderr = %q, queria o Aviso novo da segunda resolução", stderr)
	}
	troca := strings.Index(stderr, "→")
	aviso := strings.Index(stderr, "Elemento fora do Frame")
	if troca < 0 || aviso < troca {
		t.Errorf("stderr = %q: a linha de troca tem que vir antes de tudo", stderr)
	}
}

// TestFixRecusaDocumentoSomenteLeitura: a cirurgia reescreve o arquivo do
// autor, e a recusa tem que falar a língua do domínio em vez de vazar o erro
// cru do sistema operacional.
func TestFixRecusaDocumentoSomenteLeitura(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "largo.yaml")
	antes, err := os.ReadFile("largo.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	if err := os.Chmod("largo.yaml", 0o444); err != nil {
		t.Fatalf("marcando como somente-leitura: %v", err)
	}
	t.Cleanup(func() { os.Chmod("largo.yaml", 0o644) })

	codigo, _, stderr := executa("inspect", "--fix", "largo.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1", codigo)
	}
	if !strings.Contains(stderr, "erro: largo.yaml: não foi possível gravar o Documento: permissão negada") {
		t.Errorf("stderr = %q, queria a recusa no formato do domínio", stderr)
	}
	depois, err := os.ReadFile("largo.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	if string(depois) != string(antes) {
		t.Errorf("o Documento foi mexido:\n%s", depois)
	}
}

// TestFixSoExisteNoInspect: uma flag que escrevesse no Documento a partir do
// `render` transformaria uma renderização em edição.
func TestFixSoExisteNoInspect(t *testing.T) {
	for _, verbo := range []string{"render", "board", "validate"} {
		t.Run(verbo, func(t *testing.T) {
			numaPastaTemporariaDe(t, "f11", "largo.yaml")
			codigo, _, stderr := executa(verbo, "largo.yaml", "--fix")
			if codigo != 1 {
				t.Errorf("código de saída = %d, queria 1", codigo)
			}
			if !strings.Contains(stderr, `opção desconhecida "--fix"`) {
				t.Errorf("stderr = %q, queria a recusa de uso", stderr)
			}
		})
	}
	t.Run("com valor colado", func(t *testing.T) {
		naPastaDeDiagnostico(t)
		codigo, _, stderr := executa("inspect", "largo.yaml", "--fix=talvez")
		if codigo != 1 {
			t.Errorf("código de saída = %d, queria 1", codigo)
		}
		if !strings.Contains(stderr, `opção "--fix" não aceita valor, encontrou "talvez"`) {
			t.Errorf("stderr = %q, queria a recusa do valor", stderr)
		}
	})
}

// TestDiagnosticoNaoMudaComAEscala: o diagnóstico é do Documento, não da
// invocação. Uma régua que seguisse `--scale` faria o mesmo Documento estar
// certo em 8x e errado em 1x.
func TestDiagnosticoNaoMudaComAEscala(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "largo.yaml")
	_, _, um := executa("render", "largo.yaml", "--scale", "1")
	_, _, oito := executa("render", "largo.yaml", "--scale", "8")
	if um != oito {
		t.Errorf("diagnóstico com --scale 1:\n%s\ncom --scale 8:\n%s", um, oito)
	}
	if !strings.Contains(um, "não cabe no Retângulo") {
		t.Errorf("stderr = %q, queria o diagnóstico nas duas escalas", um)
	}
}

// TestDiagnosticoDaNotaContaRunas: 200 runas acentuadas custam 400 bytes, e um
// limite medido em bytes recusaria uma Nota que o autor escreveu dentro do
// limite publicado.
func TestDiagnosticoDaNotaContaRunas(t *testing.T) {
	t.Run("no limite", func(t *testing.T) {
		naPastaDeDiagnostico(t)
		codigo, _, stderr := executa("validate", "nota-no-limite.yaml")
		if codigo != 0 || stderr != "" {
			t.Errorf("código = %d, stderr = %q; queria silêncio", codigo, stderr)
		}
	})
	// Acima do limite a imagem sai do mesmo jeito, inclusive num `render` sem
	// `--notes`, que não desenha Nota nenhuma: o artefato diagnosticado é o
	// Documento, não a invocação. Um Documento não fica correto por ser
	// renderizado com a opção que esconde o defeito.
	t.Run("acima do limite", func(t *testing.T) {
		pasta := numaPastaTemporariaDe(t, "f11", "nota-longa.yaml")
		codigo, stdout, stderr := executa("render", "nota-longa.yaml")
		if codigo != 1 {
			t.Errorf("código = %d, queria 1", codigo)
		}
		if !strings.Contains(stderr, "a Nota tem 201 runas, acima do limite de 200; encurte-a") {
			t.Errorf("stderr = %q, queria a recusa da Nota comprida", stderr)
		}
		if !strings.Contains(stdout, ".webp") || len(listaDeArquivos(t, pasta)) != 2 {
			t.Errorf("stdout = %q: a imagem tinha que sair assim mesmo", stdout)
		}
	})
}

// TestEspacoDeLarguraZeroNaoVazaInfinito: dividir a largura necessária por um
// espaço de largura zero daria `+Inf`, e um `+Inf` escrito no arquivo do autor
// seria pior que o corte que se queria diagnosticar.
func TestEspacoDeLarguraZeroNaoVazaInfinito(t *testing.T) {
	numaPastaTemporariaDe(t, "f11", "slot-zero.yaml", "slot-zero-comp.yaml")
	antes, err := os.ReadFile("slot-zero.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}

	codigo, _, stderr := executa("inspect", "--fix", "slot-zero.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1", codigo)
	}
	if !strings.Contains(stderr, "o espaço do Retângulo tem largura zero") {
		t.Errorf("stderr = %q, queria a razão do espaço sem largura", stderr)
	}
	if strings.Contains(stderr, "Inf") || strings.Contains(stderr, "NaN") {
		t.Errorf("stderr = %q: número não finito escapou para o autor", stderr)
	}
	depois, err := os.ReadFile("slot-zero.yaml")
	if err != nil {
		t.Fatalf("lendo o Documento: %v", err)
	}
	if string(depois) != string(antes) {
		t.Errorf("o Documento foi mexido:\n%s", depois)
	}
}
