package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// escreve grava um Documento numa pasta descartável e devolve o caminho.
func escreve(t *testing.T, nome, conteudo string) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatalf("gravando %s: %v", nome, err)
	}
	return caminho
}

func le(t *testing.T, caminho string) string {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo %s: %v", caminho, err)
	}
	return string(dados)
}

// cabecalho é o começo comum das fixtures: quatro linhas antes do primeiro nó,
// para que a conta de linha do YAML não seja trivialmente 1.
const cabecalho = `frames:
  - name: tela
    w: 100
    h: 100
    layers:
      - name: base
`

const localDoPrimeiro = "frames[0].layers[0].elements[0]"

// TestCirurgiaPreservaOArquivoAoRedor é a afirmação central do pacote: só os
// bytes do número trocam. Cada caso é uma forma de escrever YAML que um splice
// mal medido corromperia — e o arquivo do autor não tem cópia de segurança.
func TestCirurgiaPreservaOArquivoAoRedor(t *testing.T) {
	casos := []struct {
		nome     string
		corpo    string
		local    string
		para     float64
		de       float64
		esperado string
	}{
		{
			nome: "estilo de bloco",
			corpo: `        elements:
          - rect:
              x: 0
              y: 0
              w: 20
              h: 10
`,
			local: localDoPrimeiro, para: 47, de: 20,
			esperado: `        elements:
          - rect:
              x: 0
              y: 0
              w: 47
              h: 10
`,
		},
		{
			nome: "estilo de fluxo",
			corpo: `        elements:
          - rect: {x: 0, y: 0, w: 20, h: 10}
`,
			local: localDoPrimeiro, para: 47, de: 20,
			esperado: `        elements:
          - rect: {x: 0, y: 0, w: 47, h: 10}
`,
		},
		{
			nome: "comentário na mesma linha",
			corpo: `        elements:
          - rect: {x: 0, y: 0, w: 20, h: 10} # o bloco de resultados
`,
			local: localDoPrimeiro, para: 47, de: 20,
			esperado: `        elements:
          - rect: {x: 0, y: 0, w: 47, h: 10} # o bloco de resultados
`,
		},
		{
			// A coluna do YAML conta runas: medida em bytes, o corte cairia
			// cinco bytes adiante e partiria o `h` ao meio.
			nome: "Rótulo acentuado antes do w na mesma linha",
			corpo: `        elements:
          - {label: "Configurações às três", rect: {x: 0, y: 0, w: 20, h: 10}}
`,
			local: localDoPrimeiro, para: 47, de: 20,
			esperado: `        elements:
          - {label: "Configurações às três", rect: {x: 0, y: 0, w: 47, h: 10}}
`,
		},
		{
			nome: "expoente",
			corpo: `        elements:
          - rect: {x: 0, y: 0, w: 2e1, h: 10}
`,
			local: localDoPrimeiro, para: 47, de: 20,
			esperado: `        elements:
          - rect: {x: 0, y: 0, w: 47, h: 10}
`,
		},
		{
			nome: "Retângulo dentro do preenchimento de um Slot",
			corpo: `        elements:
          - use: ./cartao.yaml
            box: {x: 0, y: 0, w: 100, h: 100}
            slots:
              corpo.principal:
                elements:
                  - rect: {x: 0, y: 0, w: 20, h: 10}
`,
			local: "frames[0].layers[0].elements[0].slots.corpo.principal.elements[0]",
			para:  47, de: 20,
			esperado: `        elements:
          - use: ./cartao.yaml
            box: {x: 0, y: 0, w: 100, h: 100}
            slots:
              corpo.principal:
                elements:
                  - rect: {x: 0, y: 0, w: 47, h: 10}
`,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			caminho := escreve(t, "doc.yaml", cabecalho+c.corpo)
			a, err := Abre(caminho)
			if err != nil {
				t.Fatalf("abrindo: %v", err)
			}
			if ok, razao := a.Alargavel(c.local); !ok {
				t.Fatalf("Alargavel(%q) = false, %q; queria true", c.local, razao)
			}
			if err := a.Alarga(c.local, c.para); err != nil {
				t.Fatalf("alargando: %v", err)
			}
			trocas, err := a.Grava()
			if err != nil {
				t.Fatalf("gravando: %v", err)
			}
			if len(trocas) != 1 || trocas[0].De != c.de || trocas[0].Para != c.para {
				t.Errorf("trocas = %+v, queria uma de %v para %v", trocas, c.de, c.para)
			}
			if got := le(t, caminho); got != cabecalho+c.esperado {
				t.Errorf("arquivo gravado:\n%s\nqueria:\n%s", got, cabecalho+c.esperado)
			}
		})
	}
}

// TestDoisRetangulosNaMesmaLinha prova a ordem decrescente do splice: aplicada
// da esquerda para a direita, a primeira troca empurraria a segunda e o corte
// cairia no meio do `h`.
func TestDoisRetangulosNaMesmaLinha(t *testing.T) {
	// As duas larguras mudam de COMPRIMENTO em texto, e em direções opostas:
	// com larguras de mesmo tamanho o splice seria correto em qualquer ordem e
	// o caso não provaria nada.
	corpo := "        elements: [{rect: {x: 0, y: 0, w: 5, h: 10}}, {rect: {x: 50, y: 0, w: 20, h: 10}}]\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	if err := a.Alarga(localDoPrimeiro, 123); err != nil {
		t.Fatalf("alargando o primeiro: %v", err)
	}
	if err := a.Alarga("frames[0].layers[0].elements[1]", 7); err != nil {
		t.Fatalf("alargando o segundo: %v", err)
	}
	trocas, err := a.Grava()
	if err != nil {
		t.Fatalf("gravando: %v", err)
	}
	// A ordem devolvida é a de registro, não a de aplicação.
	if len(trocas) != 2 || trocas[0].Para != 123 || trocas[1].Para != 7 {
		t.Errorf("trocas = %+v, queria 123 e depois 7", trocas)
	}
	quer := cabecalho + "        elements: [{rect: {x: 0, y: 0, w: 123, h: 10}}, {rect: {x: 50, y: 0, w: 7, h: 10}}]\n"
	if got := le(t, caminho); got != quer {
		t.Errorf("arquivo gravado:\n%s\nqueria:\n%s", got, quer)
	}
}

// TestGravaSegueOSymlinkAteOAlvo protege o Documento apontado por link: escrever
// ao lado do link e renomear por cima dele trocaria o link por um arquivo comum
// e deixaria o Documento real intocado.
func TestGravaSegueOSymlinkAteOAlvo(t *testing.T) {
	pasta := t.TempDir()
	real := filepath.Join(pasta, "real.yaml")
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	if err := os.WriteFile(real, []byte(cabecalho+corpo), 0o644); err != nil {
		t.Fatalf("gravando o alvo: %v", err)
	}
	link := filepath.Join(pasta, "doc.yaml")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("sem symlink neste sistema: %v", err)
	}

	a, err := Abre(link)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	if err := a.Alarga(localDoPrimeiro, 47); err != nil {
		t.Fatalf("alargando: %v", err)
	}
	if _, err := a.Grava(); err != nil {
		t.Fatalf("gravando: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat do link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("o link virou arquivo comum: o Documento real ficaria para trás")
	}
	if got := le(t, real); !strings.Contains(got, "w: 47") {
		t.Errorf("o alvo do link não foi corrigido:\n%s", got)
	}
}

// TestGravaRecusaArquivoSomenteLeitura fecha o buraco do rename: renomear por
// cima só depende de escrever no diretório, então sem conferir a permissão do
// alvo a cirurgia trocaria um Documento que o autor marcou como intocável.
func TestGravaRecusaArquivoSomenteLeitura(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	if err := a.Alarga(localDoPrimeiro, 47); err != nil {
		t.Fatalf("alargando: %v", err)
	}
	if err := os.Chmod(caminho, 0o444); err != nil {
		t.Fatalf("marcando como somente-leitura: %v", err)
	}
	t.Cleanup(func() { os.Chmod(caminho, 0o644) })

	_, err = a.Grava()
	var domínio *scene.Erro
	if err == nil {
		t.Fatal("gravou num arquivo somente-leitura")
	}
	if !comoErroDoDominio(err, &domínio) {
		t.Fatalf("erro = %T (%v), queria *scene.Erro do domínio", err, err)
	}
	if !strings.Contains(domínio.Msg, "permissão") {
		t.Errorf("mensagem = %q, queria falar de permissão", domínio.Msg)
	}
	if got := le(t, caminho); !strings.Contains(got, "w: 20") {
		t.Errorf("o arquivo foi mexido:\n%s", got)
	}
}

// TestGravaRecusaArquivoMudadoNoDisco protege quem edita o Documento enquanto o
// comando roda: os deslocamentos foram medidos contra o buffer da leitura, e
// aplicá-los sobre outro conteúdo embaralharia o YAML em silêncio.
func TestGravaRecusaArquivoMudadoNoDisco(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	if err := a.Alarga(localDoPrimeiro, 47); err != nil {
		t.Fatalf("alargando: %v", err)
	}
	mudado := cabecalho + "        elements:\n          - rect: {x: 0, y: 0, w: 30, h: 10}\n\n"
	if err := os.WriteFile(caminho, []byte(mudado), 0o644); err != nil {
		t.Fatalf("mudando o arquivo: %v", err)
	}

	if _, err := a.Grava(); err == nil {
		t.Fatal("gravou por cima de um arquivo que mudou no disco")
	} else if !strings.Contains(err.Error(), "o arquivo mudou no disco desde a leitura") {
		t.Errorf("erro = %v, queria a recusa por mudança no disco", err)
	}
	if got := le(t, caminho); got != mudado {
		t.Errorf("o arquivo foi mexido:\n%s", got)
	}
}

// TestPunhadoVazioNaoTocaOArquivo: sem nada a consertar, `--fix` não pode nem
// mudar o mtime do Documento.
func TestPunhadoVazioNaoTocaOArquivo(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	antes, err := os.Stat(caminho)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	trocas, err := a.Grava()
	if err != nil || trocas != nil {
		t.Fatalf("Grava() = %v, %v; queria nada", trocas, err)
	}
	depois, err := os.Stat(caminho)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !depois.ModTime().Equal(antes.ModTime()) {
		t.Error("o arquivo foi reescrito sem haver troca nenhuma")
	}
}

// TestAlargaRecusaLarguraImpossivel: um `w` zero, negativo ou não finito nunca
// chega ao arquivo. É a última barreira antes da escrita, e ela é do pacote que
// escreve, não de quem calcula.
func TestAlargaRecusaLarguraImpossivel(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	for _, w := range []float64{0, -1, mais(), naoNumero()} {
		a, err := Abre(caminho)
		if err != nil {
			t.Fatalf("abrindo: %v", err)
		}
		if err := a.Alarga(localDoPrimeiro, w); err == nil {
			t.Errorf("Alarga(%v) aceitou uma largura impossível", w)
		}
		if trocas, err := a.Grava(); err != nil || trocas != nil {
			t.Errorf("Grava depois da recusa = %v, %v; queria nada", trocas, err)
		}
	}
}

// TestAlargavelClassificaCadaNo fixa o predicado que decide a categoria do
// diagnóstico. Cada linha aqui é um Erro a menos ou a mais na saída do comando.
func TestAlargavelClassificaCadaNo(t *testing.T) {
	doc := cabecalho + `        elements:
          - rect: {x: 0, y: 0, w: 20, h: 10}
          - rect: {x: 0, y: 20, h: 10}
          - rect: {x: 0, y: 40, w: 20, h: 10}
            repeat: {n: 3, axis: x, gap: 1}
          - circle: {x: 0, y: 60, d: 10}
          - use: ./cartao.yaml
            box: {x: 0, y: 0, w: 100, h: 100}
            slots:
              corpo:
                elements:
                  - rect: {x: 0, y: 0, w: 20, h: 10}
          - use: ./cartao.yaml
            box: {x: 0, y: 0, w: 100, h: 100}
            repeat: {n: 2, axis: y, gap: 1}
            slots:
              corpo:
                elements:
                  - rect: {x: 0, y: 0, w: 20, h: 10}
          - rect: {x: 0, y: 80, w: &largura 20, h: 10}
`
	caminho := escreve(t, "doc.yaml", doc)
	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}

	casos := []struct {
		nome  string
		local string
		ok    bool
		razao string
	}{
		{"Retângulo com w declarado", "frames[0].layers[0].elements[0]", true, ""},
		{"Retângulo sem w", "frames[0].layers[0].elements[1]", false, RazaoSemLargura},
		{"Retângulo repetido", "frames[0].layers[0].elements[2]", false, RazaoRepeticao},
		{"nó que não é Retângulo", "frames[0].layers[0].elements[3]", false, RazaoSemLargura},
		{"Retângulo no preenchimento de um Slot", "frames[0].layers[0].elements[4].slots.corpo.elements[0]", true, ""},
		{"Retângulo no preenchimento de uma Instância repetida", "frames[0].layers[0].elements[5].slots.corpo.elements[0]", false, RazaoRepeticao},
		{"w com âncora", "frames[0].layers[0].elements[6]", false, RazaoSemLargura},
		{"Local fora da faixa", "frames[0].layers[0].elements[99]", false, RazaoSemLargura},
		{"Local de outro Frame", "frames[9].layers[0].elements[0]", false, RazaoSemLargura},
		{"Local sem gramática", "frames[0].camadas[0]", false, RazaoSemLargura},
		{"Local de Componente", "frames[0].layers[0].elements[0] -> ./cartao.yaml: elements[0]", false, RazaoSemLargura},
		{"Local vazio", "", false, RazaoSemLargura},
		{"Slot que não existe", "frames[0].layers[0].elements[4].slots.rodape.elements[0]", false, RazaoSemLargura},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ok, razao := a.Alargavel(c.local)
			if ok != c.ok || razao != c.razao {
				t.Errorf("Alargavel(%q) = %v, %q; queria %v, %q", c.local, ok, razao, c.ok, c.razao)
			}
		})
	}
}

// comoErroDoDominio é errors.As sem importar errors só para isto.
func comoErroDoDominio(err error, alvo **scene.Erro) bool {
	e, ok := err.(*scene.Erro)
	if ok {
		*alvo = e
	}
	return ok
}

func mais() float64      { return 1 / zero() }
func naoNumero() float64 { return zero() / zero() }
func zero() float64      { return 0 }

// TestChaveRepetidaTrocaAQueODesenhoUsa casa a cirurgia com a decodificação.
//
// O schema monta o mapa iterando os pares, então um nó com dois `w` desenha com
// o ÚLTIMO. Trocando o primeiro, a largura desenhada não mudaria: o Aviso
// continuaria saindo e toda execução seguinte reescreveria o arquivo imprimindo
// `w 40 → 40`, sem nunca convergir.
func TestChaveRepetidaTrocaAQueODesenhoUsa(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10, w: 21}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	if ok, razao := a.Alargavel(localDoPrimeiro); !ok {
		t.Fatalf("Alargavel = false, %q; queria true", razao)
	}
	if err := a.Alarga(localDoPrimeiro, 47); err != nil {
		t.Fatalf("alargando: %v", err)
	}
	trocas, err := a.Grava()
	if err != nil {
		t.Fatalf("gravando: %v", err)
	}
	// `De` é o valor DECODIFICADO: 21, e não os 20 da primeira ocorrência.
	if len(trocas) != 1 || trocas[0].De != 21 {
		t.Errorf("trocas = %+v, queria uma de 21", trocas)
	}
	quer := cabecalho + "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10, w: 47}\n"
	if got := le(t, caminho); got != quer {
		t.Errorf("arquivo gravado:\n%s\nqueria:\n%s", got, quer)
	}
}

// TestGravaPreservaOModoDoDocumento: o temporário nasce 0600, e um Documento de
// grupo que virasse 0600 por ter sido corrigido tiraria o acesso de quem
// compartilha o repositório com o autor.
func TestGravaPreservaOModoDoDocumento(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	// 0640 é distinto do 0600 do temporário e do 0644 que a fixture nasce:
	// com 0600 o caso não distinguiria preservar de não preservar.
	const modo os.FileMode = 0o640
	if err := os.Chmod(caminho, modo); err != nil {
		t.Fatalf("marcando o modo: %v", err)
	}

	a, err := Abre(caminho)
	if err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	if err := a.Alarga(localDoPrimeiro, 47); err != nil {
		t.Fatalf("alargando: %v", err)
	}
	if _, err := a.Grava(); err != nil {
		t.Fatalf("gravando: %v", err)
	}

	info, err := os.Stat(caminho)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != modo {
		t.Errorf("modo depois do conserto = %04o, queria %04o", got, modo)
	}
	if !strings.Contains(le(t, caminho), "w: 47") {
		t.Error("o conserto não chegou ao arquivo: o caso não prova nada sobre o modo")
	}
}

// TestAbreAmostraOMtimeAntesDaLeitura fecha a janela em que a guarda de mtime
// não guarda nada: amostrado só DEPOIS da leitura, o mtime seria o da edição do
// autor, Grava aprovaria, e o buffer velho voltaria por cima — a edição
// desapareceria em silêncio.
func TestAbreAmostraOMtimeAntesDaLeitura(t *testing.T) {
	corpo := "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n"
	caminho := escreve(t, "doc.yaml", cabecalho+corpo)
	doAutor := cabecalho + "        elements:\n          - rect: {x: 0, y: 0, w: 20, h: 10}\n          - circle: {x: 0, y: 50, d: 10}\n"

	// O autor salva EXATAMENTE entre a leitura e o segundo stat.
	original := leArquivo
	leArquivo = func(nome string) ([]byte, error) {
		dados, err := original(nome)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(nome, []byte(doAutor), 0o644); err != nil {
			t.Fatalf("simulando o salvamento do autor: %v", err)
		}
		return dados, nil
	}
	t.Cleanup(func() { leArquivo = original })

	a, err := Abre(caminho)
	if err == nil {
		// Segue até o fim para mostrar o estrago que a guarda evita.
		if erroDeAlargar := a.Alarga(localDoPrimeiro, 47); erroDeAlargar == nil {
			a.Grava()
		}
		t.Fatalf("Abre aceitou o arquivo que mudou durante a leitura; Documento agora:\n%s", le(t, caminho))
	}
	if !strings.Contains(err.Error(), "o arquivo mudou no disco desde a leitura") {
		t.Errorf("erro = %v, queria a recusa por mudança no disco", err)
	}
	if got := le(t, caminho); got != doAutor {
		t.Errorf("a edição do autor foi revertida:\n%s", got)
	}
}
