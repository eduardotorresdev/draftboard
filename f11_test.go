package main

import (
	"path/filepath"
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
