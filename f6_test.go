package main

import (
	"strings"
	"testing"
	"time"
)

// naPastaDeControles leva o teste para dentro de testdata/f6, para que as
// mensagens da CLI citem o Documento pelo nome, sem diretório.
func naPastaDeControles(t *testing.T) {
	t.Helper()
	naPasta(t, "f6")
}

// TestControleImprimeUmaLinhaPorNo prova que o Controle é fechado também na
// leitura: oito Controles declarados produzem oito linhas de Elemento, e nenhuma
// das peças que eles materializaram aparece na árvore.
func TestControleImprimeUmaLinhaPorNo(t *testing.T) {
	naPastaDeControles(t)

	codigo, saida, erros := executa("inspect", "controles.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	conferaGolden(t, "controles.txt", saida)

	elementos := 0
	for _, linha := range strings.Split(saida, "\n") {
		if strings.Contains(linha, "tom=") {
			elementos++
		}
	}
	if elementos != 8 {
		t.Errorf("linhas de Elemento = %d, quer 8 (uma por Controle declarado)", elementos)
	}
}

// TestControleAnotaUmaVezSo protege o plano de anotação: a Nota fica na cabeça
// do Controle, e as peças internas não a repetem. Se um dia uma peça herdar a
// Nota, o mesmo Controle será anotado várias vezes na margem.
func TestControleAnotaUmaVezSo(t *testing.T) {
	naPastaDeControles(t)

	_, saida, _ := executa("inspect", "controles.yaml")
	if n := strings.Count(saida, "nota:"); n != 1 {
		t.Errorf("linhas de nota = %d, quer 1", n)
	}
}

// TestRotuloNaoEhSuperficie fixa a regra de que o Rótulo é tinta, e não
// Superfície: um Retângulo desenhado por cima do Rótulo se apoia no Controle,
// não no texto, e portanto não ganha um degrau de Elevação a mais.
func TestRotuloNaoEhSuperficie(t *testing.T) {
	naPastaDeControles(t)

	codigo, saida, erros := executa("inspect", "elevacao.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	conferaGolden(t, "elevacao.txt", saida)

	casos := []struct {
		caminho string
		elev    string
	}{
		{"painel", "elev=1"},
		{"botao", "elev=2"},
		{"sobre-o-rotulo", "elev=3"},
	}
	for _, caso := range casos {
		linha := linhaDe(t, saida, caso.caminho)
		if !strings.Contains(linha, caso.elev) {
			t.Errorf("%s: %q, quer conter %q", caso.caminho, linha, caso.elev)
		}
	}
}

// TestControleDentroDeComponente prova que o Controle atravessa a fronteira do
// Componente como qualquer nó: é reescalado pela caixa da Instância e mantém a
// Origem que o inspect imprime em de=.
func TestControleDentroDeComponente(t *testing.T) {
	naPastaDeControles(t)

	codigo, saida, erros := executa("inspect", "usa-componente.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	conferaGolden(t, "usa-componente.txt", saida)
}

// TestControleRepetido prova que a Repetição anda pela box do Controle: três
// clones, cada um com a sua cabeça e o seu caminho com sufixo de clone.
func TestControleRepetido(t *testing.T) {
	naPastaDeControles(t)

	codigo, saida, erros := executa("inspect", "repetido.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	conferaGolden(t, "repetido.txt", saida)
}

// TestControleCobraOOrcamento prova que as peças internas são debitadas do teto
// de Elementos do Frame. Sem isso o teto seria ficção: um Controle materializa
// dezenas de Elementos a partir de um nó só.
func TestControleCobraOOrcamento(t *testing.T) {
	naPastaDeControles(t)

	codigo, _, erros, dentroDoPrazo := executaComPrazo(5*time.Second, "validate", "bomba.yaml")
	if !dentroDoPrazo {
		t.Fatal("validate não terminou em 5s: o orçamento não está sendo cobrado das peças do Controle")
	}
	if codigo != 1 {
		t.Fatalf("código de saída = %d, queria 1; stderr: %s", codigo, erros)
	}
	if !strings.Contains(erros, "Elementos materializados") {
		t.Errorf("stderr = %q, quer citar o teto de Elementos materializados", erros)
	}
}

// TestErrosDeControle cobre as recusas do schema. Cada uma existe para que um
// erro de escrita vire mensagem apontando o campo, e não desenho errado em
// silêncio.
func TestErrosDeControle(t *testing.T) {
	casos := []struct {
		nome    string
		fixture string
		golden  string
	}{
		{"nome fora do catálogo", "desconhecido.yaml", "desconhecido.txt"},
		{"campo de outro nó", "campo-proibido.yaml", "campo-proibido.txt"},
		{"campo de outro Controle", "campo-de-outro.yaml", "campo-de-outro.txt"},
		{"campo de Controle em Retângulo", "label-em-rect.yaml", "label-em-rect.txt"},
		{"Controle sem box", "sem-box.yaml", "sem-box.txt"},
		{"item ativo inexistente", "ativo-alem.yaml", "ativo-alem.txt"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			naPastaDeControles(t)
			codigo, saida, erros := executa("validate", caso.fixture)
			if codigo != 1 {
				t.Fatalf("código de saída = %d, queria 1", codigo)
			}
			if saida != "" {
				t.Errorf("stdout = %q, queria vazio", saida)
			}
			conferaGolden(t, caso.golden, erros)
		})
	}
}
