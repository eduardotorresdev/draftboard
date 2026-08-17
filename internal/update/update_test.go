package update

import (
	"bytes"
	"strings"
	"testing"
)

// TestComparaOrdenaVersoes é o teste que sustenta a parte mais arriscada do
// pacote. Inverter a regra de prerelease — ausência de prerelease é MAIOR que
// qualquer prerelease — faria `v1.0.0-rc.1` parecer mais nova que `v1.0.0` e o
// `update` empurraria um downgrade sem reclamar de nada.
func TestComparaOrdenaVersoes(t *testing.T) {
	casos := []struct {
		a, b  string
		ordem int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"v1.0", "v1.0.0", 0},
		{"v1", "v1.0.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.9.0", "v1.10.0", -1},
		{"v1.0.0", "v2.0.0", -1},
		{"v2.0.0", "v1.99.99", 1},
		// Build metadata é ignorado na ordenação.
		{"v1.0.0+abc", "v1.0.0+def", 0},
		{"v1.0.0+abc", "v1.0.0", 0},
		// Ausência de prerelease é maior que presença.
		{"v1.0.0-rc.1", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0-rc.1", 1},
		// Identificadores da esquerda para a direita; numéricos comparam
		// numericamente, não lexicalmente.
		{"v1.0.0-rc.1", "v1.0.0-rc.2", -1},
		{"v1.0.0-rc.2", "v1.0.0-rc.10", -1},
		{"v1.0.0-alpha", "v1.0.0-beta", -1},
		// Numérico fica abaixo de alfanumérico.
		{"v1.0.0-1", "v1.0.0-alpha", -1},
		// Empate nos compartilhados: a lista mais longa ganha.
		{"v1.0.0-rc", "v1.0.0-rc.1", -1},
		{"v1.0.0-rc.1", "v1.0.0-rc", 1},
	}
	for _, c := range casos {
		ordem, ok := Compara(c.a, c.b)
		if !ok {
			t.Errorf("Compara(%q, %q) não reconheceu as versões", c.a, c.b)
			continue
		}
		if ordem != c.ordem {
			t.Errorf("Compara(%q, %q) = %d, esperado %d", c.a, c.b, ordem, c.ordem)
		}
	}
}

// TestComparaRecusaVersaoIlegivel cobre o binário instalado por `go install`,
// que é "dev", e o lixo que uma tag mal escrita pode trazer.
func TestComparaRecusaVersaoIlegivel(t *testing.T) {
	ilegiveis := []string{"dev", "", "v", "latest", "main", "v1.2.3.4", "v1.x", "v1..0", "v1.0.0-"}
	for _, s := range ilegiveis {
		if _, ok := Compara(s, "v1.0.0"); ok {
			t.Errorf("Compara(%q, \"v1.0.0\") reconheceu uma versão ilegível", s)
		}
		if _, ok := Compara("v1.0.0", s); ok {
			t.Errorf("Compara(\"v1.0.0\", %q) reconheceu uma versão ilegível", s)
		}
	}
}

// TestAtualDevolveOsPadroesSemLdflags fixa o que "dev" significa: binário
// construído sem os -X.
func TestAtualDevolveOsPadroesSemLdflags(t *testing.T) {
	i := Atual()
	if i.Versao != "dev" {
		t.Errorf("Versao = %q, esperado \"dev\"", i.Versao)
	}
	if i.Commit != "desconhecido" || i.Data != "desconhecida" {
		t.Errorf("Commit/Data = %q/%q, esperado \"desconhecido\"/\"desconhecida\"", i.Commit, i.Data)
	}
}

func TestImprimeVersaoTrazVersaoCommitEData(t *testing.T) {
	var saida bytes.Buffer
	if err := ImprimeVersao(&saida); err != nil {
		t.Fatalf("ImprimeVersao devolveu erro: %v", err)
	}
	linhas := strings.Split(strings.TrimRight(saida.String(), "\n"), "\n")
	if len(linhas) != 3 {
		t.Fatalf("ImprimeVersao escreveu %d linhas, esperado 3:\n%s", len(linhas), saida.String())
	}
	if !strings.HasPrefix(linhas[0], "draftboard ") {
		t.Errorf("primeira linha = %q, esperado começar com \"draftboard \"", linhas[0])
	}
	if !strings.HasPrefix(linhas[1], "commit: ") || !strings.HasPrefix(linhas[2], "data: ") {
		t.Errorf("linhas de commit e data = %q, %q", linhas[1], linhas[2])
	}
}

func TestAtivoEConcatenacaoPura(t *testing.T) {
	// A tag entra verbatim, com o "v": é isso que o workflow publica.
	esperado := "draftboard_v1.4.0_darwin_arm64.tar.gz"
	if nome := Ativo("v1.4.0", "darwin", "arm64"); nome != esperado {
		t.Errorf("Ativo = %q, esperado %q", nome, esperado)
	}
}

func TestSomaDeExigeExatamenteUmaLinha(t *testing.T) {
	const soma = "3b1e2f4a5c6d7e8f90112233445566778899aabbccddeeff00112233445566aa"
	const outra = "aa11223344556677889900ffeeddccbbaa9988776655443322110a5c4f2e1b3b"
	nome := "draftboard_v1.4.0_linux_amd64.tar.gz"

	achada, err := somaDe(soma+"  "+nome+"\n"+outra+"  outro.tar.gz\n", nome)
	if err != nil {
		t.Fatalf("somaDe devolveu erro: %v", err)
	}
	if achada != soma {
		t.Errorf("soma = %q, esperado %q", achada, soma)
	}

	// Ausente e ambíguo são as duas caras do mesmo defeito: lançamento
	// quebrado. Nenhum dos dois pode virar sorteio.
	casos := map[string]string{
		"nome ausente":   outra + "  outro.tar.gz\n",
		"nome duplicado": soma + "  " + nome + "\n" + outra + "  " + nome + "\n",
		"soma curta":     "abc  " + nome + "\n",
		"soma não hexa":  strings.Repeat("z", 64) + "  " + nome + "\n",
	}
	for fato, texto := range casos {
		if _, err := somaDe(texto, nome); err == nil {
			t.Errorf("somaDe aceitou %s", fato)
		}
	}
}
