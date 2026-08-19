package diag

import (
	"math"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// A geometria do Rótulo é a que a resolução entrega: uma faixa de 28 px de
// altura, encolhida em 6 px de cada ponta. Repetida aqui de propósito — o
// diagnóstico mede o que recebe, e o teste tem que poder discordar da
// resolução em vez de concordar com ela por construção.
const (
	alturaDoRotulo  = 28.0
	respiroDoRotulo = 6.0
)

// frame monta um Frame com um Retângulo e o Rótulo dele, como a resolução os
// entregaria. espacoL é a largura do espaço em que o `w` do nó foi projetado.
func frame(texto string, x, l, espacoL float64, origem string) scene.Frame {
	esp := scene.Espaco{L: espacoL, A: 200}
	retangulo := scene.Elemento{
		Caminho: "bloco", Local: "frames[0].layers[0].elements[0]",
		Forma: scene.Retangulo, X: x, Y: 0, L: l, A: 100,
		Rotulo: texto, Origem: origem, Espaco: esp,
	}
	rotulo := scene.Elemento{
		Caminho: "bloco/rotulo", Local: retangulo.Local,
		Forma: scene.Texto, Interno: true,
		X:      x + respiroDoRotulo,
		Y:      0,
		L:      math.Max(0, l-2*respiroDoRotulo),
		A:      alturaDoRotulo,
		Rotulo: texto, Origem: origem, Espaco: esp,
	}
	return scene.Frame{Nome: "tela", L: 400, A: 200,
		Camadas: []scene.Camada{{Nome: "base", Elementos: []scene.Elemento{retangulo, rotulo}}}}
}

func documento(frames ...scene.Frame) *scene.Documento {
	return &scene.Documento{Nome: "doc", Frames: frames}
}

// sempreAlargavel e nuncaAlargavel são os predicados de mentira que separam
// este pacote de internal/fix: a corrigibilidade é entrada, não descoberta.
func sempreAlargavel(string) (bool, string) { return true, "" }

func nuncaAlargavel(razao string) func(string) (bool, string) {
	return func(string) (bool, string) { return false, razao }
}

// TestRotuloQueCabeNaoDiagnostica: o diagnóstico só fala do que está cortado.
// Um Aviso por Retângulo rotulado tornaria a saída ilegível e o silêncio da
// ferramenta sem valor.
func TestRotuloQueCabeNaoDiagnostica(t *testing.T) {
	avisos, erros := Confere("doc.yaml", documento(frame("Ok", 0, 300, 400, "")), sempreAlargavel)
	if len(avisos) != 0 || len(erros) != 0 {
		t.Errorf("avisos = %v, erros = %v; queria silêncio", avisos, erros)
	}
}

// TestRotuloCortadoViraAvisoQueConsertaDePrimeira é a promessa central de F11:
// o `w` sugerido tem que fazer o Rótulo caber. Um `w` que arredonda para baixo
// continuaria cortando, e o autor aplicaria a sugestão sem ganhar nada.
func TestRotuloCortadoViraAvisoQueConsertaDePrimeira(t *testing.T) {
	const espacoL = 400
	avisos, erros := Confere("doc.yaml", documento(frame("Resultados da busca", 0, 60, espacoL, "")), sempreAlargavel)
	if len(erros) != 0 {
		t.Fatalf("erros = %v; queria só Aviso", erros)
	}
	if len(avisos) != 1 {
		t.Fatalf("avisos = %v; queria exatamente um", avisos)
	}
	if !strings.Contains(avisos[0].Msg, `o Rótulo "Resultados da busca" não cabe no Retângulo`) {
		t.Errorf("mensagem = %q, queria falar do Rótulo que não cabe", avisos[0].Msg)
	}
	if !strings.Contains(avisos[0].Msg, "use w: ") {
		t.Errorf("mensagem = %q, queria terminar na largura sugerida", avisos[0].Msg)
	}

	consertos := Alargamentos(documento(frame("Resultados da busca", 0, 60, espacoL, "")), sempreAlargavel)
	if len(consertos) != 1 {
		t.Fatalf("consertos = %v; queria exatamente um", consertos)
	}
	// O `w` é porcentagem do espaço do nó: aplicá-lo é reprojetar o Retângulo.
	larguraNova := consertos[0].W / 100 * espacoL
	avisos, erros = Confere("doc.yaml", documento(frame("Resultados da busca", 0, larguraNova, espacoL, "")), sempreAlargavel)
	if len(avisos) != 0 || len(erros) != 0 {
		t.Errorf("depois de aplicar w=%v: avisos = %v, erros = %v; queria silêncio", consertos[0].W, avisos, erros)
	}
}

// TestCategoriaVemDaCorrigibilidade fixa a inversão que dá nome à entrega: o
// mesmo Rótulo cortado é Aviso ou Erro conforme a máquina consiga ou não
// consertá-lo, e a razão viaja na mensagem.
func TestCategoriaVemDaCorrigibilidade(t *testing.T) {
	casos := []struct {
		nome      string
		frame     scene.Frame
		alargavel func(string) (bool, string)
		aviso     bool
		trecho    string
	}{
		{
			nome:      "escrito no Documento e alargável",
			frame:     frame("Resultados da busca", 0, 60, 400, ""),
			alargavel: sempreAlargavel, aviso: true, trecho: "use w: ",
		},
		{
			nome:      "vindo de Componente",
			frame:     frame("Resultados da busca", 0, 60, 400, "cartao.yaml"),
			alargavel: sempreAlargavel,
			trecho:    "o Retângulo vem de um Componente, e alargá-lo lá muda todas as Instâncias",
		},
		{
			nome:      "sem w declarado",
			frame:     frame("Resultados da busca", 0, 60, 400, ""),
			alargavel: nuncaAlargavel(`o Retângulo não declara "w"`),
			trecho:    `o Retângulo não declara "w"`,
		},
		{
			nome:      "dentro de uma Repetição",
			frame:     frame("Resultados da busca", 0, 60, 400, ""),
			alargavel: nuncaAlargavel("o Retângulo está dentro de uma Repetição, e alargá-lo reposiciona os clones"),
			trecho:    "o Retângulo está dentro de uma Repetição, e alargá-lo reposiciona os clones",
		},
		{
			nome:      "espaço de largura zero",
			frame:     frame("Resultados da busca", 0, 60, 0, ""),
			alargavel: sempreAlargavel,
			trecho:    "o espaço do Retângulo tem largura zero",
		},
		{
			nome:      "sem predicado nenhum",
			frame:     frame("Resultados da busca", 0, 60, 400, ""),
			alargavel: nil,
			trecho:    `o Retângulo não declara "w"`,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			avisos, erros := Confere("doc.yaml", documento(c.frame), c.alargavel)
			var msg string
			switch {
			case c.aviso:
				if len(avisos) != 1 || len(erros) != 0 {
					t.Fatalf("avisos = %v, erros = %v; queria um Aviso", avisos, erros)
				}
				msg = avisos[0].Msg
			default:
				if len(erros) != 1 || len(avisos) != 0 {
					t.Fatalf("avisos = %v, erros = %v; queria um Erro", avisos, erros)
				}
				msg = erros[0].Msg
			}
			if !strings.Contains(msg, c.trecho) {
				t.Errorf("mensagem = %q, queria conter %q", msg, c.trecho)
			}
			if strings.Contains(msg, "Inf") || strings.Contains(msg, "NaN") {
				t.Errorf("mensagem = %q: número não finito escapou para o autor", msg)
			}
			// O Erro nunca vira conserto: alargar um Retângulo que a máquina
			// não sabe endereçar escreveria no arquivo errado.
			if consertos := Alargamentos(documento(c.frame), c.alargavel); c.aviso != (len(consertos) == 1) {
				t.Errorf("consertos = %v, com aviso = %v", consertos, c.aviso)
			}
		})
	}
}

// TestRotuloDeControleFicaDeFora: a caixa do Rótulo de Controle é escolha do
// catálogo, e o autor não tem no YAML um `w` de Retângulo para alargar.
// Classificá-lo junto travaria o `--fix` do Documento inteiro.
func TestRotuloDeControleFicaDeFora(t *testing.T) {
	f := frame("Salvar alterações agora", 0, 40, 400, "")
	for i := range f.Camadas[0].Elementos {
		f.Camadas[0].Elementos[i].Controle = "button"
	}
	avisos, erros := Confere("doc.yaml", documento(f), sempreAlargavel)
	if len(avisos) != 0 || len(erros) != 0 {
		t.Errorf("avisos = %v, erros = %v; queria silêncio no Rótulo de Controle", avisos, erros)
	}
}

// TestNotaAcimaDoLimiteViraErro fecha a conta em runas: 200 runas acentuadas
// custam 400 bytes, e um limite medido em bytes recusaria um texto que o autor
// escreveu dentro do limite publicado.
func TestNotaAcimaDoLimiteViraErro(t *testing.T) {
	casos := []struct {
		nome  string
		nota  string
		erros int
	}{
		{"200 runas", strings.Repeat("a", 200), 0},
		{"200 runas acentuadas", strings.Repeat("ç", 200), 0},
		{"201 runas", strings.Repeat("a", 201), 1},
		{"201 runas acentuadas", strings.Repeat("ç", 201), 1},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			f := frame("Ok", 0, 300, 400, "")
			f.Camadas[0].Elementos[0].Nota = c.nota
			avisos, erros := Confere("doc.yaml", documento(f), sempreAlargavel)
			if len(avisos) != 0 {
				t.Errorf("avisos = %v; a Nota comprida nunca é Aviso: não há largura a alargar", avisos)
			}
			if len(erros) != c.erros {
				t.Fatalf("erros = %v, queria %d", erros, c.erros)
			}
			if c.erros == 1 && !strings.Contains(erros[0].Msg, "acima do limite de 200; encurte-a") {
				t.Errorf("mensagem = %q, queria a recusa da Nota comprida", erros[0].Msg)
			}
		})
	}
}

// TestConsertosSaoDeduplicadosPorLocal: um nó materializa vários Elementos, e
// dois consertos do mesmo Local fariam a segunda cirurgia medir bytes que a
// primeira já moveu.
func TestConsertosSaoDeduplicadosPorLocal(t *testing.T) {
	// O par mais ALTO vem primeiro: a fonte do Rótulo é fração da altura da
	// faixa, então o Retângulo baixo precisa de menos largura. É o que separa
	// "vence a maior largura" de "vence a última que apareceu".
	alto := frame("Resultados da busca", 0, 60, 400, "")
	baixo := frame("Resultados da busca", 0, 60, 400, "")
	for i := range baixo.Camadas[0].Elementos {
		baixo.Camadas[0].Elementos[i].A = 12
		baixo.Camadas[0].Elementos[i].Caminho = "outro" +
			strings.TrimPrefix(baixo.Camadas[0].Elementos[i].Caminho, "bloco")
	}
	juntos := alto
	juntos.Camadas = []scene.Camada{{Nome: "base", Elementos: append(
		append([]scene.Elemento{}, alto.Camadas[0].Elementos...),
		baixo.Camadas[0].Elementos...)}}

	consertos := Alargamentos(documento(juntos), sempreAlargavel)
	if len(consertos) != 1 {
		t.Fatalf("consertos = %v; queria um só, deduplicado por Local", consertos)
	}
	// Vence a maior largura: a menor deixaria o Rótulo cortado do mesmo jeito.
	sozinho := Alargamentos(documento(alto), sempreAlargavel)
	if len(sozinho) != 1 || consertos[0].W != sozinho[0].W {
		t.Errorf("w deduplicado = %v, queria o maior (%v)", consertos[0].W, sozinho)
	}
	// E o caso só prova alguma coisa se as duas larguras forem mesmo
	// diferentes.
	if menor := Alargamentos(documento(baixo), sempreAlargavel); len(menor) != 1 || menor[0].W >= sozinho[0].W {
		t.Fatalf("as duas alturas pedem o mesmo w (%v e %v): o caso não distingue nada", menor, sozinho)
	}
}
