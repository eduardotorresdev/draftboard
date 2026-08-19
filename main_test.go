package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/skill"
)

// atualiza regrava os golden files a partir da saída observada.
var atualiza = flag.Bool("atualiza", false, "regrava os golden files de testdata")

// TestMain garante que nenhum teste deixe imagem no diretório do pacote: todo
// caso que invoca `render` tem de escrever numa pasta descartável.
func TestMain(m *testing.M) {
	flag.Parse()
	codigo := m.Run()
	if sujeira, err := filepath.Glob("*.webp"); err == nil && len(sujeira) > 0 {
		fmt.Fprintf(os.Stderr, "testes deixaram imagens no diretório do pacote: %v\n", sujeira)
		codigo = 1
	}
	os.Exit(codigo)
}

// executa roda a CLI como se fosse a linha de comando e devolve o código de
// saída, o stdout e o stderr. É o seam primário dos testes: nenhum deles
// conhece a estrutura interna da resolução nem como a Elevação é computada.
func executa(args ...string) (codigo int, stdout, stderr string) {
	return executaComEntrada("", args...)
}

// executaComEntrada é executa com uma entrada padrão, para os verbos que
// perguntam algo. A entrada é um strings.Reader, que não é dispositivo de
// caractere: os testes nunca são vistos como terminal, o que é exatamente o
// comportamento que eles precisam afirmar.
func executaComEntrada(entrada string, args ...string) (codigo int, stdout, stderr string) {
	var saida, erros bytes.Buffer
	codigo = run(args, strings.NewReader(entrada), &saida, &erros)
	return codigo, saida.String(), erros.String()
}

// naPastaDeFixtures leva o teste para dentro de testdata/f1, para que as
// mensagens da CLI citem o Documento pelo nome, sem diretório.
func naPastaDeFixtures(t *testing.T) {
	t.Helper()
	naPasta(t, "f1")
}

// naPasta leva o teste para dentro da pasta de fixtures da funcionalidade.
func naPasta(t *testing.T, funcionalidade string) {
	t.Helper()
	pasta, err := filepath.Abs(filepath.Join("testdata", funcionalidade))
	if err != nil {
		t.Fatalf("caminho das fixtures: %v", err)
	}
	t.Chdir(pasta)
}

// numaPastaTemporaria copia as fixtures pedidas para uma pasta descartável e
// leva o teste para dentro dela, para observar o que cada verbo escreve em
// disco.
func numaPastaTemporaria(t *testing.T, fixtures ...string) string {
	t.Helper()
	return numaPastaTemporariaDe(t, "f1", fixtures...)
}

// numaPastaTemporariaDe é numaPastaTemporaria para uma funcionalidade
// qualquer: copia as fixtures de testdata/<funcionalidade> e vai para lá.
func numaPastaTemporariaDe(t *testing.T, funcionalidade string, fixtures ...string) string {
	t.Helper()
	pasta := t.TempDir()
	for _, nome := range fixtures {
		origem, err := filepath.Abs(filepath.Join("testdata", funcionalidade, nome))
		if err != nil {
			t.Fatalf("caminho da fixture %s: %v", nome, err)
		}
		dados, err := os.ReadFile(origem)
		if err != nil {
			t.Fatalf("lendo a fixture %s: %v", nome, err)
		}
		if err := os.WriteFile(filepath.Join(pasta, nome), dados, 0o644); err != nil {
			t.Fatalf("copiando a fixture %s: %v", nome, err)
		}
	}
	t.Chdir(pasta)
	return pasta
}

// caminhoDosGoldens devolve o caminho absoluto da pasta de goldens, para os
// testes que rodam numa pasta temporária. Precisa ser chamado antes do Chdir.
func caminhoDosGoldens(t *testing.T) string {
	t.Helper()
	pasta, err := filepath.Abs(filepath.Join("testdata", "f1", "golden"))
	if err != nil {
		t.Fatalf("caminho dos goldens: %v", err)
	}
	return pasta
}

// conferaGolden compara o observado com o golden file de mesmo nome, relativo
// à pasta de fixtures onde o teste está.
func conferaGolden(t *testing.T, nome, observado string) {
	t.Helper()
	conferaGoldenEm(t, "golden", nome, observado)
}

func conferaGoldenEm(t *testing.T, pastaGolden, nome, observado string) {
	t.Helper()
	caminho := filepath.Join(pastaGolden, nome)
	if *atualiza {
		if err := os.MkdirAll(pastaGolden, 0o755); err != nil {
			t.Fatalf("criando golden: %v", err)
		}
		if err := os.WriteFile(caminho, []byte(observado), 0o644); err != nil {
			t.Fatalf("gravando golden: %v", err)
		}
		return
	}
	esperado, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo golden: %v", err)
	}
	if observado != string(esperado) {
		t.Errorf("saída diferente do golden %s\n--- esperado ---\n%s\n--- observado ---\n%s", nome, esperado, observado)
	}
}

// listaDeArquivos devolve os nomes das entradas da pasta, em ordem.
func listaDeArquivos(t *testing.T, pasta string) []string {
	t.Helper()
	entradas, err := os.ReadDir(pasta)
	if err != nil {
		t.Fatalf("lendo o diretório %s: %v", pasta, err)
	}
	nomes := make([]string, 0, len(entradas))
	for _, e := range entradas {
		nomes = append(nomes, e.Name())
	}
	return nomes
}

// --- inspect: geometria, Elevação e Tom ------------------------------------

func TestInspectResolveGeometriaElevacaoETom(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("inspect", "basico.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, queria vazio", stderr)
	}
	conferaGolden(t, "basico.txt", stdout)
}

// TestInspectDaUmDegrauDeElevacaoPorCamada fixa o degrau que uma Camada
// acrescenta sobre tudo abaixo dela: `caixa` está contido em `fundo`, que tem
// Elevação 1, mas nasce acima da base da sua própria Camada.
func TestInspectDaUmDegrauDeElevacaoPorCamada(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("inspect", "camadas.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	conferaGolden(t, "camadas.txt", stdout)
}

func TestInspectRevertaAEscalaNoExtremo(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("inspect", "profundo.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	conferaGolden(t, "profundo.txt", stdout)
}

func TestInspectNaoEscreveEmDisco(t *testing.T) {
	pasta := numaPastaTemporaria(t, "basico.yaml")
	codigo, stdout, _ := executa("inspect", "basico.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0", codigo)
	}
	if stdout == "" {
		t.Fatal("inspect não imprimiu a árvore")
	}
	if nomes := listaDeArquivos(t, pasta); len(nomes) != 1 || nomes[0] != "basico.yaml" {
		t.Errorf("inspect tocou em disco: %v", nomes)
	}
}

// --- validate --------------------------------------------------------------

func TestValidateAvisaRecorteEAreaZeroSemFalhar(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("validate", "avisos.yaml")
	if codigo != 0 {
		t.Errorf("código de saída = %d, queria 0: aviso não reprova o Documento", codigo)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, queria vazio", stdout)
	}
	conferaGolden(t, "avisos.txt", stderr)
}

func TestValidateNaoImprimeNadaEmSucesso(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("validate", "basico.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("saída = (%q, %q), queria vazia nos dois", stdout, stderr)
	}
}

// TestValidateReprovaDocumentoInvalido cobre cada regra de erro do contrato
// pela mensagem exata que o usuário vê e pelo código de saída 1.
//
// As Instâncias e Repetições que F1 recusava com "ainda não implementada" agora
// resolvem: a cobertura delas está em f2_test.go, junto do resto do reuso.
func TestValidateReprovaDocumentoInvalido(t *testing.T) {
	casos := []struct{ nome, fixture, golden string }{
		{"campo desconhecido sugere a chave próxima", "campo-desconhecido.yaml", "campo-desconhecido.txt"},
		{"tipo inválido", "tipo-invalido.yaml", "tipo-invalido.txt"},
		{"frames vazio", "frames-vazio.yaml", "frames-vazio.txt"},
		{"largura do Frame ausente", "frame-sem-largura.yaml", "frame-sem-largura.txt"},
		{"largura do Frame igual a zero", "frame-largura-zero.yaml", "frame-largura-zero.txt"},
		{"largura do Frame acima do máximo em pixels", "frame-largura-absurda.yaml", "frame-largura-absurda.txt"},
		{"mais de uma chave discriminante", "duas-discriminantes.yaml", "duas-discriminantes.txt"},
		{"n da Repetição menor que 1", "repeticao-n.yaml", "repeticao-n.txt"},
		{"eixo da Repetição fora de x e y", "repeticao-eixo.yaml", "repeticao-eixo.txt"},
		{"Instância sem box", "instancia-sem-box.yaml", "instancia-sem-box.txt"},
		{"Slot declarado em Documento", "slot-em-documento.yaml", "slot-em-documento.txt"},
		{"dimensão infinita", "nao-finito-infinito.yaml", "nao-finito-infinito.txt"},
		{"coordenada infinita negativa", "nao-finito-infinito-negativo.yaml", "nao-finito-infinito-negativo.txt"},
		{"dimensão NaN", "nao-finito-nan.yaml", "nao-finito-nan.txt"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			naPastaDeFixtures(t)
			codigo, stdout, stderr := executa("validate", c.fixture)
			if codigo != 1 {
				t.Errorf("código de saída = %d, queria 1", codigo)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, queria vazio", stdout)
			}
			if strings.Contains(stderr, "campo desconhecido") && c.fixture != "campo-desconhecido.yaml" {
				t.Errorf("mensagem genérica de campo desconhecido no lugar do erro específico: %s", stderr)
			}
			conferaGolden(t, c.golden, stderr)
		})
	}
}

// --- render ----------------------------------------------------------------

func TestRenderEscreveUmaImagemPorFrame(t *testing.T) {
	pasta := numaPastaTemporaria(t, "dois-frames.yaml")
	codigo, stdout, stderr := executa("render", "dois-frames.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	queria := "dois-frames-home.webp\ndois-frames-perfil.webp\n"
	if stdout != queria {
		t.Errorf("stdout = %q, queria %q", stdout, queria)
	}
	nomes := listaDeArquivos(t, pasta)
	queriaArquivos := []string{"dois-frames-home.webp", "dois-frames-perfil.webp", "dois-frames.yaml"}
	if strings.Join(nomes, ",") != strings.Join(queriaArquivos, ",") {
		t.Errorf("arquivos escritos = %v, queria %v", nomes, queriaArquivos)
	}
}

func TestRenderComLayersNumeraCadaCamadaAPartirDeUm(t *testing.T) {
	numaPastaTemporaria(t, "tres-camadas.yaml")
	codigo, stdout, stderr := executa("render", "tres-camadas.yaml", "--layers", "--out", "imagens")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	queria := strings.Join([]string{
		filepath.Join("imagens", "tres-camadas-home-page-01-fundo.webp"),
		filepath.Join("imagens", "tres-camadas-home-page-02-conteudo.webp"),
		filepath.Join("imagens", "tres-camadas-home-page-03-modal.webp"),
		"",
	}, "\n")
	if stdout != queria {
		t.Errorf("stdout = %q, queria %q", stdout, queria)
	}
	nomes := listaDeArquivos(t, "imagens")
	if len(nomes) != 3 || nomes[0] != "tres-camadas-home-page-01-fundo.webp" {
		t.Errorf("arquivos escritos = %v", nomes)
	}
}

// TestRenderUsaOsPadroesDaLinhaDeComando fixa os padrões prometidos pelo
// contrato para as opções que o esqueleto de compilação ainda não deixa
// observar na imagem.
func TestRenderUsaOsPadroesDaLinhaDeComando(t *testing.T) {
	o, err := interpretaRender([]string{"documento.yaml"})
	if err != nil {
		t.Fatalf("interpretando a linha de comando: %v", err)
	}
	if o.saida != "." {
		t.Errorf("--out padrão = %q, queria %q", o.saida, ".")
	}
	if o.escala != 1 {
		t.Errorf("--scale padrão = %v, queria 1", o.escala)
	}
	if o.notas {
		t.Error("--notes padrão ligado, queria desligado: a imagem sai sem Nota até que se peça")
	}
	if o.camadas {
		t.Error("--layers padrão ligado, queria desligado")
	}
}

// TestNotaSoApareceQuandoPedida prova as duas pontas da flag booleana pela
// imagem, que é o único observável de "desenhou Nota": o balão usa o Tom
// reservado, que nenhum Elemento pode alcançar, então contar pixels dele
// responde à pergunta sem conhecer o layout da anotação.
func TestNotaSoApareceQuandoPedida(t *testing.T) {
	pasta := numaPastaTemporariaDe(t, "f4", "notas.yaml")
	const imagem = "notas-home.webp"

	codigo, _, erros := executa("render", "notas.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	semNota := decodificaWebP(t, filepath.Join(pasta, imagem))
	if semNota.Bounds().Dx() != 200 || semNota.Bounds().Dy() != 120 {
		t.Errorf("tela sem Notas = %v, queria as dimensões do Frame, 200x120", semNota.Bounds())
	}
	if n := pixelsDoBalao(semNota); n != 0 {
		t.Errorf("%d pixels de balão sem --notes: a Nota deixou de ser opt-in", n)
	}

	codigo, _, erros = executa("render", "notas.yaml", "--notes")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	comNota := decodificaWebP(t, filepath.Join(pasta, imagem))
	if comNota.Bounds() != semNota.Bounds() {
		t.Errorf("tela com --notes = %v, queria a mesma de sem Notas, %v", comNota.Bounds(), semNota.Bounds())
	}
	if pixelsDoBalao(comNota) == 0 {
		t.Error("--notes não desenhou balão nenhum")
	}
}

// TestNotesNaoConsomeOCaminhoDoDocumento fecha a armadilha da flag booleana: o
// argumento seguinte a `--notes` só é consumido quando é um dos três modos
// aposentados, então o Documento pode vir logo depois dela.
func TestNotesNaoConsomeOCaminhoDoDocumento(t *testing.T) {
	pasta := numaPastaTemporariaDe(t, "f4", "notas.yaml")

	codigo, saida, erros := executa("render", "--notes", "notas.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	if !strings.Contains(saida, "notas-home.webp") {
		t.Errorf("stdout = %q, queria o caminho da imagem", saida)
	}
	if n := pixelsDoBalao(decodificaWebP(t, filepath.Join(pasta, "notas-home.webp"))); n == 0 {
		t.Error("o Documento foi consumido como valor de --notes: nenhum balão desenhado")
	}
}

// pixelsDoBalao conta os pixels no Tom reservado ao plano de anotação. A escada
// de Elevação nunca o alcança, então todo pixel assim é balão de Nota.
func pixelsDoBalao(img image.Image) int {
	reservado := scene.TomChrome.Cinza()
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if uint8(r>>8) == reservado {
				n++
			}
		}
	}
	return n
}

func TestRenderRejeitaOpcaoInvalida(t *testing.T) {
	casos := []struct {
		nome string
		args []string
		// contem é o trecho que a mensagem precisa trazer para que o
		// usuário saiba qual opção recusou o comando.
		contem string
	}{
		{"diretório de saída vazio", []string{"render", "dois-frames.yaml", "--out="}, `erro: opção "--out" espera um diretório, encontrou valor vazio`},
		{"escala zero", []string{"render", "dois-frames.yaml", "--scale", "0"}, `erro: opção "--scale" espera um número maior que zero, encontrou "0"`},
		{"escala não numérica", []string{"render", "dois-frames.yaml", "--scale", "grande"}, `erro: opção "--scale" espera um número maior que zero, encontrou "grande"`},
		{"modo de Nota aposentado, solto", []string{"render", "dois-frames.yaml", "--notes", "off"}, `erro: opção "--notes" não aceita mais valor: os modos margin, float e off acabaram; use "--notes" sozinho para os balões flutuantes, ou omita a opção para renderizar sem Notas`},
		{"modo de Nota aposentado, com igual", []string{"render", "dois-frames.yaml", "--notes=margin"}, `erro: opção "--notes" não aceita mais valor: os modos margin, float e off acabaram; use "--notes" sozinho para os balões flutuantes, ou omita a opção para renderizar sem Notas`},
		// `--notes` é booleana, então a forma `=` não tem valor a
		// consumir — mas o valor também não é do comando. Sem uma recusa
		// própria ele escorria para os posicionais, e a mensagem
		// acusava o Documento, ou saía sem arquivo nem motivo.
		{"--notes com valor vazio", []string{"render", "dois-frames.yaml", "--notes="}, `erro: opção "--notes" não aceita valor, encontrou ""`},
		{"--notes com valor booleano", []string{"render", "dois-frames.yaml", "--notes=true"}, `erro: opção "--notes" não aceita valor, encontrou "true"`},
		{"--notes com valor antes do Documento", []string{"render", "--notes=true", "dois-frames.yaml"}, `erro: opção "--notes" não aceita valor, encontrou "true"`},
		{"--notes com valor vazio e sem Documento", []string{"render", "--notes="}, `erro: opção "--notes" não aceita valor, encontrou ""`},
		{"opção desconhecida", []string{"render", "dois-frames.yaml", "--turbo"}, `erro: opção desconhecida "--turbo"`},
		{"sem Documento", []string{"render"}, "erro: informe o caminho do Documento"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			pasta := numaPastaTemporaria(t, "dois-frames.yaml")
			codigo, stdout, stderr := executa(c.args...)
			if codigo != 1 {
				t.Errorf("código de saída = %d, queria 1", codigo)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, queria vazio", stdout)
			}
			if strings.HasPrefix(stderr, "erro: :") {
				t.Errorf("mensagem de erro sem arquivo nem motivo: %q", stderr)
			}
			if !strings.Contains(stderr, c.contem) {
				t.Errorf("stderr = %q, queria conter %q", stderr, c.contem)
			}
			if nomes := listaDeArquivos(t, pasta); len(nomes) != 1 {
				t.Errorf("render inválido escreveu em disco: %v", nomes)
			}
		})
	}
}

// TestRenderRecusaTelaAcimaDoLimiteDeArea fixa o teto de área da CLI: a recusa
// vem antes de qualquer alocação, com código 1 e mensagem no formato do
// contrato, em vez de swap, SIGKILL ou pânico do alocador.
func TestRenderRecusaTelaAcimaDoLimiteDeArea(t *testing.T) {
	casos := []struct {
		nome   string
		args   []string
		golden string
	}{
		{"escala 100", []string{"render", "basico.yaml", "--scale", "100"}, "area-escala-100.txt"},
		{"escala 10000", []string{"render", "basico.yaml", "--scale", "10000"}, "area-escala-10000.txt"},
		// O export por Camada não tem Chrome — Notas não aparecem nele — então
		// a área recusada é a do Frame puro, menor que a do render anotado.
		{"escala 100 com export por Camada", []string{"render", "basico.yaml", "--scale", "100", "--layers"}, "area-escala-100-camadas.txt"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			goldens := caminhoDosGoldens(t)
			pasta := numaPastaTemporaria(t, "basico.yaml")
			codigo, stdout, stderr := executa(c.args...)
			if codigo != 1 {
				t.Errorf("código de saída = %d, queria 1", codigo)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, queria vazio", stdout)
			}
			if nomes := listaDeArquivos(t, pasta); len(nomes) != 1 {
				t.Errorf("render recusado escreveu imagem: %v", nomes)
			}
			conferaGoldenEm(t, goldens, c.golden, stderr)
		})
	}
}

// TestTetoDeAreaRecusaAntesDePlanejarAsNotas: a recusa pelo teto de área vem
// antes de qualquer alocação, e `--notes` não é exceção.
//
// A ordem antiga planejava a anotação primeiro, porque as margens do plano
// entravam na conta do teto. Elas não entram mais — o balão é preso dentro do
// Frame e Margens() é sempre 0 —, e planejar antes só custava: a régua do plano
// é um Canvas na escala pedida, e `--scale 9000` a fazia alocar centenas de
// megabytes para chegar exatamente ao mesmo erro.
//
// A medida é de bytes alocados no processo, não de residente: TotalAlloc é
// cumulativo e não depende do coletor ter passado.
func TestTetoDeAreaRecusaAntesDePlanejarAsNotas(t *testing.T) {
	// Folga generosa sobre o custo real da recusa, que é de kilobytes, e
	// duas ordens de grandeza abaixo dos ~280 MB da ordem antiga.
	const teto = 32 << 20

	numaPastaTemporariaDe(t, "f4", "notas.yaml")

	var antes, depois runtime.MemStats
	runtime.ReadMemStats(&antes)
	codigo, stdout, stderr := executa("render", "notas.yaml", "--notes", "--scale", "9000")
	runtime.ReadMemStats(&depois)

	if codigo != 1 {
		t.Fatalf("código de saída = %d, queria 1; stderr: %s", codigo, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, queria vazio", stdout)
	}
	if !strings.Contains(stderr, "acima do limite de") {
		t.Errorf("stderr = %q, queria o erro de teto de área", stderr)
	}
	if gasto := depois.TotalAlloc - antes.TotalAlloc; gasto > teto {
		t.Errorf("a recusa alocou %d bytes, acima de %d: o plano de anotação foi calculado antes do teto de área", gasto, teto)
	}
}

// TestTetoDeAreaFechaNoLimite fixa a borda do teto nas três posições que
// importam. A aceitação não passa pelo rasterizador de propósito: uma tela de
// render.LimiteDeArea pixels leva minutos para virar WebP, e o que está sob
// teste é a decisão, não o desenho. A recusa acima do teto continua sendo
// exercitada ponta a ponta pela CLI, onde é rápida porque nada é alocado.
func TestTetoDeAreaFechaNoLimite(t *testing.T) {
	casos := []struct {
		nome                           string
		l, a                           int
		escala                         float64
		margemT, margemD, margemB, arE float64
		cabe                           bool
	}{
		{nome: "um pixel abaixo do teto", l: render.LimiteDeArea - 1, a: 1, escala: 1, cabe: true},
		{nome: "exatamente no teto", l: render.LimiteDeArea, a: 1, escala: 1, cabe: true},
		{nome: "um pixel acima do teto", l: render.LimiteDeArea + 1, a: 1, escala: 1, cabe: false},
		{nome: "o Chrome empurra a tela acima do teto", l: render.LimiteDeArea, a: 1, escala: 1, arE: 1, cabe: false},
		{nome: "a escala empurra a tela acima do teto", l: render.LimiteDeArea / 4, a: 1, escala: 2.1, cabe: false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			o := opcoes{arquivo: "documento.yaml", saida: ".", escala: c.escala}
			f := scene.Frame{Nome: "tela", L: c.l, A: c.a}
			err := cabeNaTela(o, 0, f, c.margemT, c.margemD, c.margemB, c.arE)
			if c.cabe && err != nil {
				t.Errorf("tela recusada, queria aceita: %v", err)
			}
			if !c.cabe && err == nil {
				t.Error("tela aceita, queria recusada pelo teto de área")
			}
		})
	}
}

func TestVerboDesconhecidoFalha(t *testing.T) {
	codigo, stdout, stderr := executa("desenhar", "x.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1", codigo)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, queria vazio", stdout)
	}
	if stderr == "" {
		t.Error("stderr vazio: o verbo desconhecido não foi reportado")
	}
}

// servidorDeLancamentos sobe um servidor local que imita a API do GitHub e
// aponta a CLI para ele pelo seam documentado no CONTRATO.
func servidorDeLancamentos(t *testing.T) {
	t.Helper()
	bruto, err := os.ReadFile(filepath.Join("testdata", "f7", "releases-latest.json"))
	if err != nil {
		t.Fatalf("não foi possível ler a fixture: %v", err)
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.ReplaceAll(string(bruto), "%BASE%", srv.URL)))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DRAFTBOARD_LANCAMENTOS_URL", srv.URL)
}

func TestVersionImprimeVersaoCommitEData(t *testing.T) {
	codigo, stdout, stderr := executa("version")
	if codigo != 0 {
		t.Fatalf("código = %d, stderr = %q", codigo, stderr)
	}
	if stderr != "" {
		t.Errorf("version escreveu no stderr: %q", stderr)
	}
	linhas := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(linhas) != 3 {
		t.Fatalf("version escreveu %d linhas, esperado 3:\n%s", len(linhas), stdout)
	}
	// Sem os -X do release, o binário se identifica como "dev". É esse o
	// estado de quem instalou por `go install`.
	if linhas[0] != "draftboard dev" {
		t.Errorf("primeira linha = %q, esperado \"draftboard dev\"", linhas[0])
	}
}

// TestUpdateCheckReportaVersaoDisponivel também fixa que a comparação
// impossível ("dev" contra uma tag) vira aviso, e não erro.
func TestUpdateCheckReportaVersaoDisponivel(t *testing.T) {
	servidorDeLancamentos(t)
	codigo, stdout, stderr := executa("update", "--check")
	if codigo != 0 {
		t.Fatalf("código = %d, stderr = %q", codigo, stderr)
	}
	if !strings.Contains(stdout, "atualização disponível: v1.4.0") {
		t.Errorf("stdout = %q, esperado a linha de atualização disponível", stdout)
	}
	if !strings.HasPrefix(stderr, "aviso: ") {
		t.Errorf("stderr = %q, esperado o aviso da versão \"dev\"", stderr)
	}
}

// TestUpdateCheckNaoEscreveEmDisco é o que faz `--check` ser seguro de rodar em
// qualquer lugar, inclusive num diretório de trabalho de alguém.
func TestUpdateCheckNaoEscreveEmDisco(t *testing.T) {
	servidorDeLancamentos(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if codigo, _, stderr := executa("update", "--check"); codigo != 0 {
		t.Fatalf("código = %d, stderr = %q", codigo, stderr)
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("não foi possível listar %s: %v", dir, err)
	}
	if len(entradas) != 0 {
		t.Errorf("update --check deixou %d entrada(s) em disco", len(entradas))
	}
}

func TestUpdateRejeitaOpcaoInvalida(t *testing.T) {
	codigo, stdout, stderr := executa("update", "--forca")
	if codigo != 1 {
		t.Errorf("código = %d, esperado 1", codigo)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, esperado vazio", stdout)
	}
	if !strings.HasPrefix(stderr, "erro: ") {
		t.Errorf("stderr = %q, esperado começar com \"erro: \"", stderr)
	}
}

func TestUpdateRejeitaYesComNo(t *testing.T) {
	if codigo, _, _ := executa("update", "--yes", "--no"); codigo != 1 {
		t.Errorf("código = %d, esperado 1", codigo)
	}
}

func TestSkillRejeitaInstallComSync(t *testing.T) {
	if codigo, _, _ := executa("skill", "--install", "--sync"); codigo != 1 {
		t.Errorf("código = %d, esperado 1", codigo)
	}
}

// TestSkillSyncNaoImprimeNadaQuandoJaSincronizada é o caminho normal de um
// update: a skill não mudou, então o verbo é silencioso e um update de rotina
// cabe em duas linhas.
func TestSkillSyncNaoImprimeNadaQuandoJaSincronizada(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	if codigo, _, stderr := executa("skill", "--install"); codigo != 0 {
		t.Fatalf("instalação falhou: código = %d, stderr = %q", codigo, stderr)
	}
	codigo, stdout, stderr := executa("skill", "--sync")
	if codigo != 0 {
		t.Fatalf("código = %d, stderr = %q", codigo, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("--sync falou com a skill já sincronizada: stdout = %q, stderr = %q", stdout, stderr)
	}
}

// TestSkillSyncNaoGravaQuandoAEntradaNaoEUmTerminal: gravar em ~/.claude numa
// invocação canalizada violaria o padrão de só reportar; errar quebraria o
// caminho dirigido por agente. Então o verbo avisa e sai 0.
func TestSkillSyncNaoGravaQuandoAEntradaNaoEUmTerminal(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	codigo, stdout, stderr := executaComEntrada("s\n", "skill", "--sync")
	if codigo != 0 {
		t.Fatalf("código = %d, stderr = %q", codigo, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, esperado vazio: nada foi gravado", stdout)
	}
	if !strings.HasPrefix(stderr, "aviso: ") {
		t.Errorf("stderr = %q, esperado o aviso de entrada não interativa", stderr)
	}
	if _, err := os.Stat(filepath.Join(casa, ".claude", "skills", "draftboard", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("a skill foi gravada sem confirmação: %v", err)
	}
}

func TestSkillSyncGravaComYes(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	codigo, stdout, stderr := executa("skill", "--sync", "--yes")
	if codigo != 0 {
		t.Fatalf("código = %d, stderr = %q", codigo, stderr)
	}
	esperado := filepath.Join(casa, ".claude", "skills", "draftboard", "SKILL.md")
	if strings.TrimSpace(stdout) != esperado {
		t.Errorf("stdout = %q, esperado o caminho gravado %q", stdout, esperado)
	}
	if _, err := os.Stat(esperado); err != nil {
		t.Errorf("a skill não foi gravada: %v", err)
	}
}

func TestSkillSyncNaoGravaComNo(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	codigo, stdout, _ := executa("skill", "--sync", "--no")
	if codigo != 0 {
		t.Fatalf("código = %d", codigo)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, esperado vazio", stdout)
	}
	if _, err := os.Stat(filepath.Join(casa, ".claude", "skills", "draftboard", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("--no gravou a skill: %v", err)
	}
}

// TestConfirmaSoAceitaSim cobre a leitura da resposta, que a suíte não alcança
// pela CLI: em teste a entrada nunca é um terminal, e sem terminal o verbo não
// pergunta. O padrão é sempre o que NÃO escreve em disco.
func TestConfirmaSoAceitaSim(t *testing.T) {
	sim := []string{"s\n", "sim\n", "y\n", "yes\n", "S\n", "  Sim  \n", "SIM"}
	for _, e := range sim {
		if !confirma(strings.NewReader(e)) {
			t.Errorf("confirma(%q) = false, esperado true", e)
		}
	}
	nao := []string{"", "\n", "n\n", "nao\n", "no\n", "talvez\n", "sim mais tarde\n"}
	for _, e := range nao {
		if confirma(strings.NewReader(e)) {
			t.Errorf("confirma(%q) = true, esperado false", e)
		}
	}
}

// TestUsoDaCLIBateComASkill fecha a duplicação mais provável de dessincronizar:
// o bloco de uso de imprimeUso e o bloco de CLI da skill são duas cópias à mão
// das mesmas linhas, e nada mais confere que elas concordam.
func TestUsoDaCLIBateComASkill(t *testing.T) {
	var uso bytes.Buffer
	imprimeUso(&uso)
	conteudo := skill.Conteudo()
	for _, linha := range strings.Split(uso.String(), "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" || linha == "uso:" {
			continue
		}
		if !strings.Contains(conteudo, linha) {
			t.Errorf("a skill não traz a linha de uso:\n%s", linha)
		}
	}
}
