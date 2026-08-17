// Package controls é o catálogo embutido de Controles: as peças fechadas que o
// Documento invoca por nome, em vez de por arquivo, e que se materializam em
// vários Elementos antes de qualquer cálculo de Elevação.
//
// O pacote é deliberadamente a única casa do catálogo. Acrescentar um Controle
// novo é acrescentar uma Definicao aqui — schema, resolução, rasterização e
// inspect não têm caso por Controle e não precisam mudar.
//
// Nenhuma função daqui conhece métrica de fonte: o Rótulo recebe a área em que
// deve caber, e só o rasterizador sabe a largura real da string.
package controls

import (
	"fmt"
	"strings"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// LimiteDeItens é o número máximo de itens de um Controle. Existe pelo mesmo
// motivo do teto de clones da Repetição: um número grande no YAML não pode
// virar alocação sem fim antes do orçamento do Frame ser cobrado.
const LimiteDeItens = 1_000

// Parametros são os campos declarados no nó do Controle, já decodificados.
// O mesmo struct serve todos os Controles; cada Definicao lê só o que usa.
type Parametros struct {
	// Nome é o nome de catálogo, como escrito no YAML.
	Nome string
	// Rotulo é o campo "label".
	Rotulo string
	// Itens são os rótulos do campo "items" quando ele veio como lista.
	Itens []string
	// Quantos é a quantidade de itens: o número do campo "items", ou o
	// comprimento de Itens quando ele veio como lista.
	Quantos int
	// Ativo é o campo "active", em base 1. Zero significa nenhum item ativo.
	Ativo int
	// Valor é o campo "value", de 0 a 100.
	Valor float64
}

// Peca é um Elemento que o Controle materializa, com geometria já em pixels
// absolutos relativos ao Frame. A primeira Peca devolvida pelo layout é sempre
// a cabeça: ela ocupa a box declarada e é quem aparece na árvore do inspect.
type Peca struct {
	// Segmento é o sufixo do caminho da Peca dentro do Controle, como "trilho"
	// ou "item#0". A cabeça tem Segmento vazio: ela usa o caminho do próprio nó.
	Segmento    string
	Forma       scene.Forma
	X, Y, L, A  float64
	Arredondado bool
	Rotulo      string
	Alinhamento scene.Alinhamento
}

// Definicao é a entrada do catálogo de um Controle.
type Definicao struct {
	// Nome é a chave do catálogo, como se escreve em `control:`.
	Nome string
	// Chaves são os campos que este Controle aceita além dos comuns a todo nó
	// (box, id, note, repeat).
	Chaves []string

	padroes func(*Parametros)
	valida  func(Parametros) string
	layout  func(Parametros, float64, float64, float64, float64) []Peca
	detalhe func(Parametros) string
}

// Padroes preenche os campos que o YAML não declarou.
func (d Definicao) Padroes(p *Parametros) {
	if d.padroes != nil {
		d.padroes(p)
	}
}

// Valida devolve a mensagem de erro do Controle, ou vazio quando os parâmetros
// são aceitáveis. A mensagem já vem no vocabulário do domínio, sem prefixo.
func (d Definicao) Valida(p Parametros) string {
	if p.Quantos > LimiteDeItens {
		return fmt.Sprintf("campo %q do Controle deve estar entre 1 e %d, encontrou %d",
			"items", LimiteDeItens, p.Quantos)
	}
	if d.valida != nil {
		return d.valida(p)
	}
	return ""
}

// Layout materializa o Controle na caixa em pixels de Frame que a resolução
// abriu. A primeira Peca é a cabeça.
func (d Definicao) Layout(p Parametros, x, y, l, a float64) []Peca {
	return d.layout(p, x, y, l, a)
}

// Detalhe formata os parâmetros para a linha do inspect. Como o Controle é
// opaco na árvore, esta string é a única forma de o agente ler o que foi
// declarado sem reabrir o YAML.
func (d Definicao) Detalhe(p Parametros) string {
	if d.detalhe == nil {
		return ""
	}
	return d.detalhe(p)
}

// Definido busca um Controle no catálogo.
func Definido(nome string) (Definicao, bool) {
	d, ok := catalogo[nome]
	return d, ok
}

// Nomes devolve os nomes do catálogo em ordem alfabética. É o que alimenta a
// sugestão de nome próximo quando o Controle não existe.
func Nomes() []string {
	fora := make([]string, 0, len(catalogo))
	for nome := range catalogo {
		fora = append(fora, nome)
	}
	ordena(fora)
	return fora
}

// ordena é uma inserção simples: o catálogo tem dezenas de entradas, não vale
// arrastar o pacote sort para isso.
func ordena(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// rotulos formata a lista de rótulos para o Detalhe do inspect.
func rotulos(itens []string) string {
	return "[" + strings.Join(itens, ", ") + "]"
}

// rotuloDoItem devolve o rótulo do item i, ou vazio quando os itens vieram
// como número e portanto não têm texto.
func rotuloDoItem(p Parametros, i int) string {
	if i < len(p.Itens) {
		return p.Itens[i]
	}
	return ""
}

// numero formata um float sem casas decimais inúteis, para o Detalhe.
func numero(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}
