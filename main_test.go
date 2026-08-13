package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/notes"
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
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
	var saida, erros bytes.Buffer
	codigo = run(args, &saida, &erros)
	return codigo, saida.String(), erros.String()
}

// naPastaDeFixtures leva o teste para dentro de testdata/f1, para que as
// mensagens da CLI citem o Documento pelo nome, sem diretório.
func naPastaDeFixtures(t *testing.T) {
	t.Helper()
	pasta, err := filepath.Abs(filepath.Join("testdata", "f1"))
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
	pasta := t.TempDir()
	for _, nome := range fixtures {
		origem, err := filepath.Abs(filepath.Join("testdata", "f1", nome))
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
	if o.notas != notes.Margem {
		t.Errorf("--notes padrão = %v, queria notes.Margem", o.notas)
	}
	if o.camadas {
		t.Error("--layers padrão ligado, queria desligado")
	}
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
		{"modo de Nota inexistente", []string{"render", "dois-frames.yaml", "--notes", "verde"}, `erro: opção "--notes" espera margin, float ou off, encontrou "verde"`},
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
