package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// atualiza regrava os golden files a partir da saída observada.
var atualiza = flag.Bool("atualiza", false, "regrava os golden files de testdata/f1")

// executa roda a CLI como se fosse a linha de comando e devolve o código de
// saída, o stdout e o stderr. É o único seam usado pelos testes: nenhum deles
// conhece a estrutura interna da resolução.
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

// conferaGolden compara o observado com o golden file de mesmo nome.
func conferaGolden(t *testing.T, nome, observado string) {
	t.Helper()
	caminho := filepath.Join("golden", nome)
	if *atualiza {
		if err := os.MkdirAll("golden", 0o755); err != nil {
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

func TestInspectRevertaAEscalaNoExtremo(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("inspect", "profundo.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, stderr)
	}
	conferaGolden(t, "profundo.txt", stdout)
}

func TestInspectNaoEscreveEmDisco(t *testing.T) {
	origem, err := filepath.Abs(filepath.Join("testdata", "f1", "basico.yaml"))
	if err != nil {
		t.Fatalf("caminho da fixture: %v", err)
	}
	dados, err := os.ReadFile(origem)
	if err != nil {
		t.Fatalf("lendo a fixture: %v", err)
	}
	pasta := t.TempDir()
	if err := os.WriteFile(filepath.Join(pasta, "basico.yaml"), dados, 0o644); err != nil {
		t.Fatalf("copiando a fixture: %v", err)
	}
	t.Chdir(pasta)

	codigo, stdout, _ := executa("inspect", "basico.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0", codigo)
	}
	if stdout == "" {
		t.Fatal("inspect não imprimiu a árvore")
	}
	entradas, err := os.ReadDir(pasta)
	if err != nil {
		t.Fatalf("lendo o diretório: %v", err)
	}
	if len(entradas) != 1 || entradas[0].Name() != "basico.yaml" {
		nomes := make([]string, 0, len(entradas))
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Errorf("inspect tocou em disco: %v", nomes)
	}
}

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

func TestValidateFalhaComCampoDesconhecido(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, stdout, stderr := executa("validate", "campo-desconhecido.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1", codigo)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, queria vazio", stdout)
	}
	conferaGolden(t, "campo-desconhecido.txt", stderr)
}

func TestValidateFalhaComTipoInvalido(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, _, stderr := executa("validate", "tipo-invalido.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1", codigo)
	}
	conferaGolden(t, "tipo-invalido.txt", stderr)
}

func TestValidateRecusaInstanciaAindaNaoImplementada(t *testing.T) {
	naPastaDeFixtures(t)
	codigo, _, stderr := executa("validate", "instancia.yaml")
	if codigo != 1 {
		t.Errorf("código de saída = %d, queria 1", codigo)
	}
	conferaGolden(t, "instancia.txt", stderr)
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
