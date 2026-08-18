package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// naPastaDeFluxos leva o teste para dentro de testdata/f8.
func naPastaDeFluxos(t *testing.T) {
	t.Helper()
	naPasta(t, "f8")
}

// TestErrosDeLigacao cobre as recusas da chave `to`. Cada uma existe para que
// um fluxo escrito errado vire mensagem apontando o campo, e não uma seta que
// some em silêncio da Prancheta.
func TestErrosDeLigacao(t *testing.T) {
	casos := []struct {
		nome    string
		fixture string
		golden  string
	}{
		{"Frame de destino inexistente", "destino-desconhecido.yaml", "destino-desconhecido.txt"},
		{"destino sem nome próximo não sugere nada", "destino-sem-parecido.yaml", "destino-sem-parecido.txt"},
		{"Ligação junto de Repetição", "destino-com-repeat.yaml", "destino-com-repeat.txt"},
		{"Ligação em Instância", "destino-em-instancia.yaml", "destino-em-instancia.txt"},
		{"Ligação dentro de Componente", "destino-em-componente.yaml", "destino-em-componente.txt"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			naPastaDeFluxos(t)
			codigo, saida, erros := executa("validate", caso.fixture)
			if codigo != 1 {
				t.Fatalf("código de saída = %d, esperava 1; stderr: %s", codigo, erros)
			}
			if saida != "" {
				t.Errorf("erro de Ligação escreveu no stdout: %q", saida)
			}
			conferaGolden(t, caso.golden, erros)
		})
	}
}

// TestArvoreMostraALigacao afirma que o fluxo é legível sem olhar o desenho: o
// `inspect` diz para onde cada gatilho leva, e é assim que um agente lê a
// navegação sem gastar visão.
func TestArvoreMostraALigacao(t *testing.T) {
	naPastaDeFluxos(t)
	codigo, saida, erros := executa("inspect", "fluxo.yaml")
	if codigo != 0 {
		t.Fatalf("código de saída = %d; stderr: %s", codigo, erros)
	}
	conferaGolden(t, "fluxo-inspect.txt", saida)
}

// TestLigacaoNaoMudaOWebP protege a fronteira entre a Ligação e o desenho: ela
// não participa da Elevação nem tem geometria, então a imagem do Frame tem de
// sair byte a byte igual com e sem `to`.
func TestLigacaoNaoMudaOWebP(t *testing.T) {
	pasta := numaPastaTemporariaDe(t, "f8", "par-com-to.yaml", "par-sem-to.yaml")
	for _, fixture := range []string{"par-com-to.yaml", "par-sem-to.yaml"} {
		saidaDoCaso := filepath.Join(pasta, strings.TrimSuffix(fixture, ".yaml"))
		if codigo, _, erros := executa("render", fixture, "--out", saidaDoCaso); codigo != 0 {
			t.Fatalf("render de %s falhou: %s", fixture, erros)
		}
	}
	com := filepath.Join(pasta, "par-com-to")
	sem := filepath.Join(pasta, "par-sem-to")
	nomes := listaDeArquivos(t, com)
	if len(nomes) != 2 {
		t.Fatalf("esperava duas imagens, saíram %v", nomes)
	}
	for _, nome := range nomes {
		// Os nomes derivam do Documento, que difere: compara por posição.
		semEquivalente := strings.Replace(nome, "par-com-to", "par-sem-to", 1)
		a := leArquivo(t, filepath.Join(com, nome))
		b := leArquivo(t, filepath.Join(sem, semEquivalente))
		if string(a) != string(b) {
			t.Errorf("a Ligação mudou o desenho de %s: %d bytes contra %d", nome, len(a), len(b))
		}
	}
}

// TestBoardEscreveUmArquivoSo afirma o contrato de saída do verbo: a Prancheta
// é um arquivo, o nome deriva do Documento, e só o caminho vai para o stdout.
func TestBoardEscreveUmArquivoSo(t *testing.T) {
	pasta := numaPastaTemporariaDe(t, "f8", "fluxo.yaml")
	saidaDoCaso := filepath.Join(pasta, "saida")
	codigo, stdout, erros := executa("board", "fluxo.yaml", "--out", saidaDoCaso)
	if codigo != 0 {
		t.Fatalf("código de saída = %d; stderr: %s", codigo, erros)
	}
	esperado := filepath.Join(saidaDoCaso, "fluxo.html")
	if stdout != esperado+"\n" {
		t.Errorf("stdout = %q, esperava o caminho da Prancheta %q", stdout, esperado)
	}
	if nomes := listaDeArquivos(t, saidaDoCaso); len(nomes) != 1 || nomes[0] != "fluxo.html" {
		t.Errorf("a Prancheta não é um arquivo só: %v", nomes)
	}
}

// TestPranchetaEDeterministica protege o mesmo que o WebP protege: rodar duas
// vezes o mesmo Documento tem de dar os mesmos bytes, para que o diff em Git
// seja confiável.
func TestPranchetaEDeterministica(t *testing.T) {
	pasta := numaPastaTemporariaDe(t, "f8", "fluxo.yaml")
	var saidas []string
	for _, vez := range []string{"a", "b"} {
		destino := filepath.Join(pasta, vez)
		if codigo, _, erros := executa("board", "fluxo.yaml", "--out", destino); codigo != 0 {
			t.Fatalf("board falhou: %s", erros)
		}
		saidas = append(saidas, string(leArquivo(t, filepath.Join(destino, "fluxo.html"))))
	}
	if saidas[0] != saidas[1] {
		t.Error("duas execuções de board produziram Pranchetas diferentes")
	}
}

// TestPranchetaTemUmCaminhoPorLigacao afirma que nenhuma Ligação declarada se
// perde no caminho da resolução até o SVG, e que nenhuma é inventada.
func TestPranchetaTemUmCaminhoPorLigacao(t *testing.T) {
	html := prancheta(t, "fluxo.yaml")
	linhas := regexp.MustCompile(`<path class="ligacao"[^>]*data-de="(\d+)" data-para="(\d+)"`).FindAllStringSubmatch(html, -1)
	// fluxo.yaml declara seis `to`: dois em login, dois em dashboard, um em
	// detalhe e um em recuperar.
	if len(linhas) != 6 {
		t.Fatalf("a Prancheta desenhou %d Ligações, esperava 6", len(linhas))
	}
}

// TestPranchetaNaoBuscaNadaDeFora afirma o que faz a Prancheta abrir por
// file://: nenhum script, folha de estilo ou imagem vem de outro arquivo nem
// da rede. O único endereço tolerado é o espaço de nomes do SVG, que é um
// identificador e não um pedido.
func TestPranchetaNaoBuscaNadaDeFora(t *testing.T) {
	html := prancheta(t, "fluxo.yaml")
	for _, proibido := range []string{"<script src", "<link ", "<img ", "@import", "url(http", "fetch(", "XMLHttpRequest"} {
		if strings.Contains(html, proibido) {
			t.Errorf("a Prancheta traz recurso de fora: %q", proibido)
		}
	}
	enderecos := regexp.MustCompile(`https?://[^"' ]+`).FindAllString(html, -1)
	for _, e := range enderecos {
		if e != "http://www.w3.org/2000/svg" {
			t.Errorf("a Prancheta cita um endereço externo: %s", e)
		}
	}
}

// TestPranchetaEscapaOTextoDoYAML protege contra o Documento que injeta markup
// pelo nome de um Frame ou pelo texto de uma Nota.
func TestPranchetaEscapaOTextoDoYAML(t *testing.T) {
	html := prancheta(t, "escapes.yaml")
	// O `<script>` do roteiro é da própria Prancheta: o que não pode aparecer
	// é o texto do YAML cru.
	if strings.Contains(html, "a & b <script>") {
		t.Error("o nome do Frame entrou na Prancheta sem escape")
	}
	if strings.Contains(html, `aspas " e &`) {
		t.Error("a Nota entrou na Prancheta sem escape")
	}
	if !strings.Contains(html, "a &amp; b &lt;script&gt;") {
		t.Error("o nome do Frame não aparece escapado")
	}
	if !strings.Contains(html, "aspas &#34; e &amp; &lt;tag&gt;") {
		t.Error("a Nota não aparece escapada")
	}
}

// TestPranchetaSemLigacaoAindaDispoeOsFrames: um Documento sem fluxo nenhum não
// tem grafo, e ainda assim tem de virar Prancheta — em grade, sem sobreposição.
func TestPranchetaSemLigacaoAindaDispoeOsFrames(t *testing.T) {
	html := prancheta(t, "sem-ligacao.yaml")
	if strings.Contains(html, `class="ligacao"`) {
		t.Error("Documento sem `to` desenhou Ligação")
	}
	caixas := caixasDosFrames(t, html)
	if len(caixas) != 3 {
		t.Fatalf("a Prancheta tem %d Frames, esperava 3", len(caixas))
	}
	conferaSemSobreposicao(t, caixas)
}

// TestCicloDeLigacoesNaoTravaOLayout: A→B→A é fluxo comum — todo botão "voltar"
// fecha um ciclo. O layout tem de terminar e devolver posições finitas.
func TestCicloDeLigacoesNaoTravaOLayout(t *testing.T) {
	html := prancheta(t, "ciclo.yaml")
	caixas := caixasDosFrames(t, html)
	if len(caixas) != 2 {
		t.Fatalf("a Prancheta tem %d Frames, esperava 2", len(caixas))
	}
	conferaSemSobreposicao(t, caixas)
	// A auto-Ligação de `b` não empurra `b` para depois de si mesmo: a entrada
	// continua sendo `a`, na primeira coluna.
	if caixas[0].x >= caixas[1].x {
		t.Errorf("o Frame de entrada não ficou à esquerda: %v", caixas)
	}
}

// TestPranchetaDemaisERecusadaAntesDeMontar: cada Elemento vira um nó do DOM, e
// um Documento acima do teto tem de ser recusado com mensagem, não entregue
// como uma Prancheta que trava o navegador.
func TestPranchetaDemaisERecusadaAntesDeMontar(t *testing.T) {
	pasta := t.TempDir()
	fixture := filepath.Join(pasta, "gigante.yaml")
	var b strings.Builder
	b.WriteString("frames:\n")
	// Cada Frame carrega o teto de Elementos por Frame; seis deles passam do
	// teto da Prancheta sem passar do teto de nenhum Frame.
	for i := 0; i < 6; i++ {
		b.WriteString("  - name: f")
		b.WriteString(string(rune('a' + i)))
		b.WriteString("\n    w: 320\n    h: 240\n    layers:\n      - name: base\n        elements:\n")
		b.WriteString("          - rect: {x: 0, y: 0, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 2, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 4, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 6, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 8, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 10, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 12, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 14, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 16, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
		b.WriteString("          - rect: {x: 0, y: 18, w: 1, h: 1}\n            repeat: {n: 1000, axis: x, gap: 0}\n")
	}
	if err := os.WriteFile(fixture, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("escrevendo a fixture: %v", err)
	}
	saida := filepath.Join(pasta, "saida")
	codigo, stdout, erros := executa("board", fixture, "--out", saida)
	if codigo != 1 {
		t.Fatalf("código de saída = %d, esperava 1; stderr: %s", codigo, erros)
	}
	if stdout != "" {
		t.Errorf("a recusa escreveu no stdout: %q", stdout)
	}
	if !strings.Contains(erros, "acima do limite") {
		t.Errorf("a mensagem não diz que passou do teto: %q", erros)
	}
	if _, err := os.Stat(saida); !os.IsNotExist(err) {
		t.Error("a recusa criou o diretório de saída antes de conferir o teto")
	}
}

// prancheta gera a Prancheta de uma fixture de f8 e devolve o HTML.
func prancheta(t *testing.T, fixture string) string {
	t.Helper()
	fixtures := []string{fixture}
	if fixture == "destino-em-componente.yaml" {
		fixtures = append(fixtures, "peca.yaml")
	}
	pasta := numaPastaTemporariaDe(t, "f8", fixtures...)
	saida := filepath.Join(pasta, "saida")
	codigo, stdout, erros := executa("board", fixture, "--out", saida)
	if codigo != 0 {
		t.Fatalf("board de %s falhou: %s", fixture, erros)
	}
	return string(leArquivo(t, strings.TrimSpace(stdout)))
}

type caixa struct{ x, y, l, a float64 }

// caixasDosFrames lê a posição e as dimensões de cada Frame direto do SVG, na
// ordem de declaração.
func caixasDosFrames(t *testing.T, html string) []caixa {
	t.Helper()
	re := regexp.MustCompile(`data-x="([-0-9.]+)" data-y="([-0-9.]+)" data-l="([0-9]+)" data-a="([0-9]+)"`)
	var caixas []caixa
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		caixas = append(caixas, caixa{
			x: numeroDoTeste(t, m[1]), y: numeroDoTeste(t, m[2]),
			l: numeroDoTeste(t, m[3]), a: numeroDoTeste(t, m[4]),
		})
	}
	return caixas
}

// conferaSemSobreposicao afirma que dois Frames nunca ocupam o mesmo espaço na
// Prancheta: telas empilhadas uma sobre a outra não se leem.
func conferaSemSobreposicao(t *testing.T, caixas []caixa) {
	t.Helper()
	for i := range caixas {
		for j := i + 1; j < len(caixas); j++ {
			a, b := caixas[i], caixas[j]
			if a.x < b.x+b.l && b.x < a.x+a.l && a.y < b.y+b.a && b.y < a.y+a.a {
				t.Errorf("os Frames %d e %d se sobrepõem: %+v e %+v", i, j, a, b)
			}
		}
	}
}

func numeroDoTeste(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("coordenada inválida no SVG: %q", s)
	}
	return v
}

func leArquivo(t *testing.T, caminho string) []byte {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo %s: %v", caminho, err)
	}
	return dados
}
