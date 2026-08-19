package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/resolve"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// naPastaDeRotulos leva o teste para dentro de testdata/f9, para que as
// mensagens da CLI citem o Documento pelo nome, sem diretório.
func naPastaDeRotulos(t *testing.T) {
	t.Helper()
	naPasta(t, "f9")
}

// resolveDeF9 resolve uma fixture de f9 sem passar pela CLI. O Elemento do
// Rótulo é Interno, e o `inspect` o esconde de propósito: as afirmações de
// geometria não têm outro observável.
func resolveDeF9(t *testing.T, fixture string) *scene.Documento {
	t.Helper()
	caminho, err := filepath.Abs(filepath.Join("testdata", "f9", fixture))
	if err != nil {
		t.Fatalf("caminho da fixture %s: %v", fixture, err)
	}
	doc, avisos, err := resolve.Arquivo(caminho)
	if err != nil {
		t.Fatalf("resolvendo %s: %v", fixture, err)
	}
	if len(avisos) != 0 {
		t.Fatalf("avisos inesperados em %s: %v", fixture, avisos)
	}
	return doc
}

// elementos devolve os Elementos de todas as Camadas do primeiro Frame, na
// ordem de pintura.
func elementos(d *scene.Documento) []scene.Elemento {
	var todos []scene.Elemento
	for _, c := range d.Frames[0].Camadas {
		todos = append(todos, c.Elementos...)
	}
	return todos
}

// porCaminho acha o Elemento de caminho exato, incluindo os Internos.
func porCaminho(t *testing.T, d *scene.Documento, caminho string) scene.Elemento {
	t.Helper()
	for _, e := range elementos(d) {
		if e.Caminho == caminho {
			return e
		}
	}
	t.Fatalf("nenhum Elemento de caminho %q; achei %v", caminho, caminhos(d))
	return scene.Elemento{}
}

func caminhos(d *scene.Documento) []string {
	var lista []string
	for _, e := range elementos(d) {
		lista = append(lista, e.Caminho)
	}
	return lista
}

// TestRotuloDoRetanguloApareceNaLinhaDoRetangulo prova o que o `inspect`
// mostra e o que ele esconde: o texto vira sufixo da linha do Retângulo que o
// declarou, e o Elemento de Forma Texto que o desenha não vira linha nenhuma.
// Sem isso, um Documento com vinte blocos rotulados custaria quarenta linhas
// para ser lido.
func TestRotuloDoRetanguloApareceNaLinhaDoRetangulo(t *testing.T) {
	naPastaDeRotulos(t)

	codigo, saida, erros := executa("inspect", "rotulos.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	conferaGolden(t, "rotulos.txt", saida)

	for _, linha := range strings.Split(saida, "\n") {
		if strings.Contains(linha, " texto ") {
			t.Errorf("a árvore mostrou a peça de Texto do Rótulo: %q", linha)
		}
	}
	if !strings.Contains(saida, `regiao retangulo 0,0 200x400 tom=300 elev=1 rotulo="Grade"`) {
		t.Errorf("o sufixo rotulo= não fechou a linha do Retângulo; saída:\n%s", saida)
	}
}

// TestRotuloDeRetanguloComFilhoVaiParaOTopo fixa metade da regra de posição: o
// bloco que contém outro Elemento apoia o Rótulo numa faixa no topo, alinhada à
// esquerda, para sair do caminho dos filhos.
func TestRotuloDeRetanguloComFilhoVaiParaOTopo(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")

	retangulo := porCaminho(t, doc, "regiao")
	rotulo := porCaminho(t, doc, "regiao/rotulo")

	if rotulo.Alinhamento != scene.AEsquerda {
		t.Errorf("Alinhamento = %v, quer AEsquerda: o Retângulo contém um filho", rotulo.Alinhamento)
	}
	if rotulo.Y != retangulo.Y {
		t.Errorf("Y = %v, quer %v (o topo do Retângulo)", rotulo.Y, retangulo.Y)
	}
	if rotulo.Forma != scene.Texto {
		t.Errorf("Forma = %v, quer Texto", rotulo.Forma)
	}
	if !rotulo.Interno {
		t.Error("o Rótulo tem de ser Interno: ele não é uma linha da árvore")
	}
	if rotulo.Controle != "" {
		t.Errorf("Controle = %q, quer vazio: este Rótulo não veio do catálogo", rotulo.Controle)
	}
}

// TestRotuloDeRetanguloVazioFicaCentrado é a outra metade, e é ela que prova
// que a varredura exclui o próprio Retângulo: contemGeometricamente é inclusiva
// nas quatro bordas, então todo Retângulo contém a si mesmo, e sem a exclusão
// nenhum bloco seria vazio.
func TestRotuloDeRetanguloVazioFicaCentrado(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")

	retangulo := porCaminho(t, doc, "centrado")
	rotulo := porCaminho(t, doc, "centrado/rotulo")

	if rotulo.Alinhamento != scene.AoCentro {
		t.Errorf("Alinhamento = %v, quer AoCentro: o Retângulo não contém ninguém", rotulo.Alinhamento)
	}
	esperado := retangulo.Y + (retangulo.A-rotulo.A)/2
	if rotulo.Y != esperado {
		t.Errorf("Y = %v, quer %v (centrado na vertical)", rotulo.Y, esperado)
	}
	if rotulo.Y <= retangulo.Y {
		t.Errorf("Y = %v, mas o Rótulo centrado tem de descer do topo %v", rotulo.Y, retangulo.Y)
	}
}

// TestAlturaDoRotuloNaoAcompanhaOBloco protege a regra de tamanho da fonte: a
// caixa é fixa em px do Frame, não uma fração do Retângulo. Num bloco de 400 px
// a fração daria uma fonte de 180 px, que é mockup, não wireframe.
func TestAlturaDoRotuloNaoAcompanhaOBloco(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")

	alto := porCaminho(t, doc, "regiao")
	if alto.A != 400 {
		t.Fatalf("a fixture mudou: o Retângulo alto tem %v px de altura, queria 400", alto.A)
	}
	if a := porCaminho(t, doc, "regiao/rotulo").A; a != 28 {
		t.Errorf("altura do Rótulo = %v, quer 28 num Retângulo de 400", a)
	}

	baixo := porCaminho(t, doc, "baixo")
	if baixo.A != 10 {
		t.Fatalf("a fixture mudou: o Retângulo baixo tem %v px de altura, queria 10", baixo.A)
	}
	if a := porCaminho(t, doc, "baixo/rotulo").A; a != 10 {
		t.Errorf("altura do Rótulo = %v, quer 10: a caixa satura na altura do Retângulo", a)
	}
}

// TestRotuloEmRetanguloEstreitoNaoFicaNegativo cobre o respiro contra um bloco
// mais estreito que ele: a largura satura em zero, sem pânico, e sem aviso — a
// geometria do Rótulo é derivada, e o autor não a declarou.
func TestRotuloEmRetanguloEstreitoNaoFicaNegativo(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")

	estreito := porCaminho(t, doc, "estreito")
	if estreito.L >= 12 {
		t.Fatalf("a fixture mudou: o Retângulo estreito tem %v px de largura, queria menos que o respiro dobrado", estreito.L)
	}
	if l := porCaminho(t, doc, "estreito/rotulo").L; l != 0 {
		t.Errorf("largura do Rótulo = %v, quer 0", l)
	}
}

// TestCaminhoDoRotuloTemSegmentoProprio prova que o Rótulo não depende do
// desempate de caminhoUnico. Sem o segmento fixo ele herdaria o caminho do
// Retângulo com sufixo `~2`, e o painel da Prancheta mostraria um Elemento
// fantasma ao lado de cada bloco rotulado.
func TestCaminhoDoRotuloTemSegmentoProprio(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")

	for _, base := range []string{"regiao", "baixo", "estreito", "centrado"} {
		porCaminho(t, doc, base+"/rotulo")
	}
	for _, c := range caminhos(doc) {
		if strings.Contains(c, "~") {
			t.Errorf("caminho desambiguado por sufixo: %q", c)
		}
	}
}

// TestRotuloSeApoiaNoRetanguloQueOCarrega é a razão de o Rótulo ser emitido
// imediatamente depois do seu Retângulo, e não no fim da Camada: a Superfície
// dele tem de ser o bloco que o carrega, é de lá que vêm a Elevação e o Tom que
// dão contraste ao texto.
//
// A fixture é a barra de cabeçalho, e é ela que dá mordida ao teste: o
// `cabecalho` cobre exatamente a faixa onde o Rótulo se apoia. Fosse a Elevação
// calculada depois de o Rótulo subir para o fim da Camada, a Superfície dele
// seria o `cabecalho`, e o texto ganharia um degrau a mais e sumiria dentro
// dele.
func TestRotuloSeApoiaNoRetanguloQueOCarrega(t *testing.T) {
	doc := resolveDeF9(t, "cabecalho.yaml")

	regiao := porCaminho(t, doc, "regiao")
	cabecalho := porCaminho(t, doc, "cabecalho")
	rotulo := porCaminho(t, doc, "regiao/rotulo")

	if rotulo.Y != regiao.Y || rotulo.Y+rotulo.A > cabecalho.Y+cabecalho.A {
		t.Fatalf("a fixture mudou: o Rótulo em %v..%v não está dentro da barra %v..%v",
			rotulo.Y, rotulo.Y+rotulo.A, cabecalho.Y, cabecalho.Y+cabecalho.A)
	}
	if rotulo.Elevacao != regiao.Elevacao+1 {
		t.Errorf("Elevação do Rótulo = %d, quer %d (um degrau sobre o Retângulo que o carrega)",
			rotulo.Elevacao, regiao.Elevacao+1)
	}
	if rotulo.Elevacao == cabecalho.Elevacao+1 {
		t.Errorf("Elevação do Rótulo = %d: ele se apoiou na barra que o cobre, não no Retângulo dono",
			rotulo.Elevacao)
	}
	if rotulo.Tom != cabecalho.Tom {
		t.Errorf("Tom do Rótulo = %d, quer %d: os dois se apoiam no mesmo Retângulo",
			int(rotulo.Tom), int(cabecalho.Tom))
	}
	if rotulo.Tom == regiao.Tom {
		t.Errorf("Tom do Rótulo = %d, igual ao do Retângulo: o texto não teria contraste", int(rotulo.Tom))
	}
}

// TestRotuloEPintadoDepoisDosFilhos protege a segunda passagem. Emitido junto do
// dono, o Rótulo é pintado antes dos filhos, e a barra de cabeçalho mais trivial
// que existe o apaga da imagem em silêncio: o `inspect` continua dizendo
// `rotulo="Resultados"`, a geometria e o Tom saem certos, e o WebP sai sem texto
// nenhum.
func TestRotuloEPintadoDepoisDosFilhos(t *testing.T) {
	doc := resolveDeF9(t, "cabecalho.yaml")

	camada := doc.Frames[0].Camadas[0]
	ultimo := camada.Elementos[len(camada.Elementos)-1]
	if ultimo.Caminho != "regiao/rotulo" {
		t.Errorf("o último Elemento pintado da Camada é %q, quer %q: o Rótulo tem de ir por cima dos filhos",
			ultimo.Caminho, "regiao/rotulo")
	}
	if indiceNaCamada(t, camada, "regiao/rotulo") < indiceNaCamada(t, camada, "cabecalho") {
		t.Error("o Rótulo é pintado antes da barra que cobre a faixa dele: sumiria da imagem")
	}
}

// TestRotuloDeControleNaoSobeParaOFimDaCamada delimita a segunda passagem: ela
// é do Rótulo de Retângulo, e as peças de um Controle mantêm a ordem que o
// catálogo montou. Subir o Rótulo do Controle o tiraria de cima da sua própria
// peça de destaque.
func TestRotuloDeControleNaoSobeParaOFimDaCamada(t *testing.T) {
	doc := resolveDeF9(t, "tudo.yaml")

	camada := doc.Frames[0].Camadas[0]
	if indiceNaCamada(t, camada, "e2/item#0") > indiceNaCamada(t, camada, "e2/item#0/ativo") {
		t.Error("o Rótulo do Controle mudou de lugar na Camada: a ordem das peças é do catálogo")
	}
}

// indiceNaCamada devolve a posição de um Elemento na ordem de pintura da Camada.
func indiceNaCamada(t *testing.T, c scene.Camada, caminho string) int {
	t.Helper()
	for i, e := range c.Elementos {
		if e.Caminho == caminho {
			return i
		}
	}
	t.Fatalf("nenhum Elemento de caminho %q na Camada %q", caminho, c.Nome)
	return -1
}

// TestRotuloPagaOTetoDeElementos protege o orçamento do Frame: sem o débito,
// `rect` com `label` dentro de uma Repetição materializa o dobro de Elementos
// sob o mesmo teto, e o teto vira ficção.
func TestRotuloPagaOTetoDeElementos(t *testing.T) {
	naPastaDeRotulos(t)

	if codigo, _, erros := executa("validate", "repete-sem-rotulo.yaml"); codigo != 0 {
		t.Fatalf("a fixture sem Rótulo já não cabe no teto: código %d; stderr: %s", codigo, erros)
	}
	codigo, _, erros := executa("validate", "repete-com-rotulo.yaml")
	if codigo != 1 {
		t.Fatalf("código de saída = %d, queria 1: o Rótulo não foi debitado", codigo)
	}
	if !strings.Contains(erros, "passou do teto de 10000 Elementos materializados") {
		t.Errorf("stderr não citou o teto de Elementos:\n%s", erros)
	}
}

// TestLabelSoEmRetanguloOuControle fixa as recusas. `circle` não entra porque a
// faixa retangular no topo cairia fora da forma; `use` e `slot` não entram
// porque não deixam Elemento seu no Documento resolvido para carregar o texto.
func TestLabelSoEmRetanguloOuControle(t *testing.T) {
	casos := []struct {
		nome    string
		fixture string
		local   string
	}{
		{"Círculo", "so-circulo.yaml", "frames[0].layers[0].elements[0]"},
		{"Instância", "so-instancia.yaml", "frames[0].layers[0].elements[0]"},
		{"Slot", "usa-slot.yaml", "frames[0].layers[0].elements[0] -> ./com-slot.yaml: elements[0]"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			naPastaDeRotulos(t)

			codigo, saida, erros := executa("validate", caso.fixture)
			if codigo != 1 {
				t.Fatalf("código de saída = %d, queria 1", codigo)
			}
			if saida != "" {
				t.Errorf("stdout = %q, queria vazio", saida)
			}
			esperado := "erro: " + caso.fixture + ": " + caso.local +
				`: campo "label" só é permitido em Retângulo ou Controle` + "\n"
			if erros != esperado {
				t.Errorf("stderr = %q, quer %q", erros, esperado)
			}
		})
	}

	t.Run("Controle continua aceitando", func(t *testing.T) {
		naPastaDeRotulos(t)

		codigo, saida, erros := executa("inspect", "controle-com-label.yaml")
		if codigo != 0 {
			t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
		}
		if !strings.Contains(saida, `rotulo="Salvar"`) {
			t.Errorf("o Controle perdeu o seu Rótulo; saída:\n%s", saida)
		}
	})
}

// TestTodoElementoCarregaLocal protege o campo que o diagnóstico vai consumir:
// sem Local em algum caminho de materialização, o Aviso de um Controle ou de um
// Componente não teria como apontar o nó que o autor escreveu.
func TestTodoElementoCarregaLocal(t *testing.T) {
	doc := resolveDeF9(t, "tudo.yaml")

	vistos := 0
	for _, e := range elementos(doc) {
		if e.Local == "" {
			t.Errorf("Elemento %q sem Local", e.Caminho)
		}
		vistos++
	}
	if vistos < 8 {
		t.Fatalf("a fixture cobriu só %d Elementos: ela precisa passar por Retângulo, Círculo, Controle, Componente, Slot e Repetição", vistos)
	}
}

// TestLocalDoRotuloApontaOMesmoNoDoRetangulo registra que Local não é único: os
// dois Elementos de um Retângulo rotulado vêm do mesmo nó, e é por Local que
// quem quiser falar do nó escrito pelo autor deduplica.
func TestLocalDoRotuloApontaOMesmoNoDoRetangulo(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")

	retangulo := porCaminho(t, doc, "regiao")
	rotulo := porCaminho(t, doc, "regiao/rotulo")
	if rotulo.Local != retangulo.Local {
		t.Errorf("Local do Rótulo = %q, quer %q", rotulo.Local, retangulo.Local)
	}
	if retangulo.Local != "frames[0].layers[0].elements[0]" {
		t.Errorf("Local = %q, quer o caminho de chaves YAML do nó", retangulo.Local)
	}
}

var (
	textoDeRotulo   = regexp.MustCompile(`<text class="[^"]*rotulo"[^>]*>`)
	recorteAplicado = regexp.MustCompile(`clip-path="url\(#(rotulo-\d+-\d+)\)"`)
)

// TestPranchetaRecortaORotuloNaAreaDele fecha a divergência entre os dois
// desenhistas: o raster já recorta o Rótulo na própria área, e o SVG só era
// recortado na borda do Frame. Com `label` livre do autor, um texto largo demais
// passaria por cima dos vizinhos na Prancheta e sairia cortado no WebP — o mesmo
// Documento com dois desenhos.
func TestPranchetaRecortaORotuloNaAreaDele(t *testing.T) {
	doc := resolveDeF9(t, "rotulos.yaml")
	rotulo := porCaminho(t, doc, "regiao/rotulo")

	naPastaDeRotulos(t)
	pasta := t.TempDir()
	if codigo, _, erros := executa("board", "rotulos.yaml", "--out", pasta); codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	html := string(leArquivo(t, filepath.Join(pasta, "rotulos.html")))

	textos := textoDeRotulo.FindAllString(html, -1)
	if len(textos) != 4 {
		t.Fatalf("Rótulos desenhados = %d, quer 4", len(textos))
	}
	for _, tag := range textos {
		if !recorteAplicado.MatchString(tag) {
			t.Errorf("Rótulo sem recorte próprio: %s", tag)
		}
	}

	recorte := fmt.Sprintf(`<clipPath id="rotulo-0-0"><rect x="%s" y="%s" width="%s" height="%s"/></clipPath>`,
		coordenada(rotulo.X), coordenada(rotulo.Y), coordenada(rotulo.L), coordenada(rotulo.A))
	if !strings.Contains(html, recorte) {
		t.Errorf("a área recortada não é a do Rótulo; esperava a linha:\n%s", recorte)
	}
}

// coordenada formata um valor como a Prancheta o escreve no SVG.
func coordenada(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// TestCaminhoDoRotuloUsaOCaminhoDesambiguadoDoDono protege o pareamento
// Rótulo↔dono por prefixo, de que vivem o painel da Prancheta e o diagnóstico.
// Montado sobre o caminho bruto, o Rótulo do segundo bloco ficaria pendurado no
// caminho do primeiro, e o texto seria atribuído ao Retângulo errado.
func TestCaminhoDoRotuloUsaOCaminhoDesambiguadoDoDono(t *testing.T) {
	doc := resolveDeF9(t, "ids-repetidos.yaml")

	primeiro := porCaminho(t, doc, "bloco/rotulo")
	if primeiro.Rotulo != "Primeiro" {
		t.Errorf("Rótulo de %q = %q, quer %q", "bloco/rotulo", primeiro.Rotulo, "Primeiro")
	}
	segundo := porCaminho(t, doc, "bloco~2/rotulo")
	if segundo.Rotulo != "Segundo" {
		t.Errorf("Rótulo de %q = %q, quer %q", "bloco~2/rotulo", segundo.Rotulo, "Segundo")
	}
	dono := porCaminho(t, doc, "bloco~2")
	if segundo.Y < dono.Y || segundo.Y > dono.Y+dono.A {
		t.Errorf("o Rótulo em Y=%v não está dentro do Retângulo %q (%v..%v): ele pendurou no bloco errado",
			segundo.Y, dono.Caminho, dono.Y, dono.Y+dono.A)
	}
}

// TestLabelTemTetoDeComprimento fecha a última entrada de texto do módulo sem
// limite. O rasterizador desenha glifo a glifo mesmo quando a máscara descarta
// tudo: um `label` de 200 000 runas dentro de uma Repetição custa horas de CPU
// para produzir uma imagem onde nada aparece.
func TestLabelTemTetoDeComprimento(t *testing.T) {
	t.Run("no teto passa", func(t *testing.T) {
		naPastaDeRotulos(t)

		if codigo, _, erros := executa("validate", "label-no-limite.yaml"); codigo != 0 {
			t.Fatalf("código de saída = %d, queria 0: 200 runas cabem; stderr: %s", codigo, erros)
		}
	})

	t.Run("uma runa acima recusa", func(t *testing.T) {
		naPastaDeRotulos(t)

		codigo, saida, erros := executa("validate", "label-longo.yaml")
		if codigo != 1 {
			t.Fatalf("código de saída = %d, queria 1", codigo)
		}
		if saida != "" {
			t.Errorf("stdout = %q, queria vazio", saida)
		}
		esperado := "erro: label-longo.yaml: frames[0].layers[0].elements[0]: " +
			`campo "label" passou do teto de 200 runas: encontrou 201` + "\n"
		if erros != esperado {
			t.Errorf("stderr = %q, quer %q", erros, esperado)
		}
	})
}

// TestNotaSaiEntreAspasNaArvore fecha a assimetria com o Rótulo. Uma Nota com
// quebra de linha seguida de seis espaços produz, na árvore, uma linha
// indistinguível de um Elemento real para o agente que a lê.
func TestNotaSaiEntreAspasNaArvore(t *testing.T) {
	naPastaDeRotulos(t)

	codigo, saida, erros := executa("inspect", "nota-forjada.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d, queria 0; stderr: %s", codigo, erros)
	}
	esperado := `        nota: "primeira linha\n      fantasma retangulo 0,0 100x100 tom=100 elev=9"`
	if !strings.Contains(saida, esperado) {
		t.Errorf("a Nota não saiu entre aspas; queria a linha:\n%s\nsaída:\n%s", esperado, saida)
	}
	for _, linha := range strings.Split(saida, "\n") {
		if strings.HasPrefix(linha, "      fantasma ") {
			t.Errorf("a Nota forjou uma linha de Elemento na árvore: %q", linha)
		}
	}
}
