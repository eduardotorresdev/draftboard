package schema

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/eduardotorresdev/draftboard/internal/controls"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Chaves válidas de cada nível. A ordem é a ordem de desempate da sugestão.
var (
	chavesDoDocumento     = []string{"frames"}
	chavesDoComponente    = []string{"elements"}
	chavesDoFrame         = []string{"name", "w", "h", "layers"}
	chavesDaCamada        = []string{"name", "elements"}
	chavesDoNo            = []string{"rect", "circle", "use", "slot", "control", "round", "id", "note", "to", "repeat", "box", "slots", "default", "label", "items", "active", "value"}
	chavesDaCaixa         = []string{"x", "y", "w", "h"}
	chavesDoDisco         = []string{"x", "y", "d"}
	chavesDaRepeticao     = []string{"n", "axis", "gap"}
	chavesDoPreenchimento = []string{"use", "elements"}

	discriminantes = []string{"rect", "circle", "use", "slot", "control"}

	// chavesDeControle são todos os campos que algum Controle do catálogo
	// aceita. Estão em chavesDoNo para que a sugestão de chave próxima os
	// conheça, e a permissão real é conferida contra o Controle declarado.
	chavesDeControle = []string{"label", "items", "active", "value"}

	// chavesSoDeControle são os campos que nenhum outro nó aceita. É uma lista
	// à parte de chavesDeControle porque `label` deixou de ser exclusivo do
	// Controle: o Retângulo também o aceita, e tem regra e mensagem próprias.
	chavesSoDeControle = []string{"items", "active", "value"}
)

// LeDocumento lê e decodifica o arquivo YAML no caminho dado como Documento. O
// tipo do arquivo é inferido pelo conteúdo: declarar `frames` faz dele um
// Documento. Um arquivo sem `frames` é um Componente, e é recusado aqui.
//
// O erro devolvido é sempre do tipo *scene.Erro.
func LeDocumento(caminho string) (*Documento, error) {
	l, raiz, err := abre(caminho)
	if err != nil {
		return nil, err
	}
	if !temChave(raiz, "frames") {
		return nil, l.erro("", "esperava um Documento, mas o arquivo não declara `frames`; Componente só pode ser usado por uma Instância")
	}
	l.emDocumento = true
	return l.documento(raiz, caminho)
}

// LeComponente lê e decodifica o arquivo YAML no caminho dado como Componente:
// um arquivo sem `frames`, escrito num espaço de coordenadas próprio de 0 a
// 100. Um arquivo com `frames` é um Documento, e é recusado aqui.
//
// O erro devolvido é sempre do tipo *scene.Erro.
func LeComponente(caminho string) (*Componente, error) {
	l, raiz, err := abre(caminho)
	if err != nil {
		return nil, err
	}
	if temChave(raiz, "frames") {
		return nil, l.erro("", "esperava um Componente, mas o arquivo declara `frames`; um Documento não pode ser instanciado")
	}
	return l.componente(raiz, caminho)
}

// abre lê o arquivo, decodifica o YAML e devolve o leitor já preparado junto do
// nó de mapa no topo do arquivo.
func abre(caminho string) (*leitor, *yaml.Node, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		msg := "não foi possível ler o arquivo"
		if errors.Is(err, fs.ErrNotExist) {
			msg = "arquivo não encontrado"
		} else if p := (*fs.PathError)(nil); errors.As(err, &p) {
			msg += ": " + p.Err.Error()
		}
		return nil, nil, &scene.Erro{Arquivo: caminho, Msg: msg}
	}

	l := &leitor{arquivo: caminho}

	var doc yaml.Node
	if err := yaml.Unmarshal(dados, &doc); err != nil {
		return nil, nil, &scene.Erro{Arquivo: caminho, Msg: "YAML inválido: " + err.Error()}
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return nil, nil, l.erro("", "arquivo vazio: declare `frames` para um Documento ou `elements` para um Componente")
	}
	raiz := doc.Content[0]
	if raiz.Kind != yaml.MappingNode {
		return nil, nil, l.erro("", "esperava um mapa no topo do arquivo, encontrou %s", nomeDoTipo(raiz))
	}
	return l, raiz, nil
}

// leitor carrega o estado comum da decodificação de um arquivo.
type leitor struct {
	arquivo string
	// emDocumento distingue um Documento de um Componente: a declaração de
	// Slot só é permitida em Componente.
	emDocumento bool
}

func (l *leitor) erro(local, formato string, args ...any) *scene.Erro {
	return &scene.Erro{Arquivo: l.arquivo, Local: local, Msg: fmt.Sprintf(formato, args...)}
}

// nomeDoDocumento deriva o nome do Documento do nome do arquivo, sem diretório
// e sem extensão.
func nomeDoDocumento(caminho string) string {
	base := filepath.Base(caminho)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (l *leitor) documento(raiz *yaml.Node, caminho string) (*Documento, error) {
	m, err := l.mapa(raiz, "", chavesDoDocumento)
	if err != nil {
		return nil, err
	}
	itens, err := l.sequencia(m.valores["frames"], "", "frames")
	if err != nil {
		return nil, err
	}
	if len(itens) == 0 {
		return nil, l.erro("frames", "Documento sem Frames: declare ao menos um Frame")
	}
	d := &Documento{Arquivo: caminho, Nome: nomeDoDocumento(caminho)}
	for i, item := range itens {
		f, err := l.frame(item, fmt.Sprintf("frames[%d]", i))
		if err != nil {
			return nil, err
		}
		d.Frames = append(d.Frames, f)
	}
	if err := l.validaDestinos(d); err != nil {
		return nil, err
	}
	return d, nil
}

// validaDestinos confere que toda Ligação aponta para um Frame declarado no
// mesmo Documento. Roda depois do laço de Frames porque um `to` pode apontar
// para frente: a tela de destino não precisa vir antes do gatilho no arquivo.
func (l *leitor) validaDestinos(d *Documento) error {
	nomes := make([]string, 0, len(d.Frames))
	declarados := make(map[string]bool, len(d.Frames))
	for _, f := range d.Frames {
		nomes = append(nomes, f.Nome)
		declarados[f.Nome] = true
	}
	var confere func(nos []No) error
	confere = func(nos []No) error {
		for _, no := range nos {
			if no.Destino != "" && !declarados[no.Destino] {
				msg := fmt.Sprintf("Ligação para Frame desconhecido %q", no.Destino)
				if s := sugestao(no.Destino, nomes); s != "" {
					msg += fmt.Sprintf("; você quis dizer %q?", s)
				}
				return l.erro(no.Local, "%s", msg)
			}
			if err := confere(no.Padrao); err != nil {
				return err
			}
			for _, nome := range no.OrdemDosPreenchimentos {
				if err := confere(no.Preenchimentos[nome].Elementos); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, f := range d.Frames {
		for _, c := range f.Camadas {
			if err := confere(c.Elementos); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *leitor) componente(raiz *yaml.Node, caminho string) (*Componente, error) {
	m, err := l.mapa(raiz, "", chavesDoComponente)
	if err != nil {
		return nil, err
	}
	c := &Componente{Arquivo: caminho}
	if m.valores["elements"] == nil {
		return c, nil
	}
	nos, err := l.nos(m.valores["elements"], "", "elements")
	if err != nil {
		return nil, err
	}
	c.Elementos = nos
	return c, nil
}

func (l *leitor) frame(n *yaml.Node, local string) (Frame, error) {
	m, err := l.mapa(n, local, chavesDoFrame)
	if err != nil {
		return Frame{}, err
	}
	f := Frame{Local: local}
	if f.Nome, err = l.texto(m, "name"); err != nil {
		return Frame{}, err
	}
	if f.L, err = l.dimensao(m, "w"); err != nil {
		return Frame{}, err
	}
	if f.A, err = l.dimensao(m, "h"); err != nil {
		return Frame{}, err
	}
	if m.valores["layers"] == nil {
		return f, nil
	}
	itens, err := l.sequencia(m.valores["layers"], local, "layers")
	if err != nil {
		return Frame{}, err
	}
	for i, item := range itens {
		c, err := l.camada(item, fmt.Sprintf("%s.layers[%d]", local, i))
		if err != nil {
			return Frame{}, err
		}
		f.Camadas = append(f.Camadas, c)
	}
	return f, nil
}

func (l *leitor) camada(n *yaml.Node, local string) (Camada, error) {
	m, err := l.mapa(n, local, chavesDaCamada)
	if err != nil {
		return Camada{}, err
	}
	c := Camada{Local: local}
	if c.Nome, err = l.texto(m, "name"); err != nil {
		return Camada{}, err
	}
	if m.valores["elements"] == nil {
		return c, nil
	}
	if c.Elementos, err = l.nos(m.valores["elements"], local, "elements"); err != nil {
		return Camada{}, err
	}
	return c, nil
}

// nos decodifica a sequência de nós de elemento guardada em campo.
func (l *leitor) nos(n *yaml.Node, local, campo string) ([]No, error) {
	itens, err := l.sequencia(n, local, campo)
	if err != nil {
		return nil, err
	}
	base := campo
	if local != "" {
		base = local + "." + campo
	}
	nos := make([]No, 0, len(itens))
	for i, item := range itens {
		no, err := l.no(item, fmt.Sprintf("%s[%d]", base, i))
		if err != nil {
			return nil, err
		}
		nos = append(nos, no)
	}
	return nos, nil
}

func (l *leitor) no(n *yaml.Node, local string) (No, error) {
	m, err := l.mapa(n, local, chavesDoNo)
	if err != nil {
		return No{}, err
	}

	var achadas []string
	for _, d := range discriminantes {
		if m.valores[d] != nil {
			achadas = append(achadas, `"`+d+`"`)
		}
	}
	switch len(achadas) {
	case 0:
		return No{}, l.erro(local, `nó de elemento sem chave discriminante; declare "rect", "circle", "use", "slot" ou "control"`)
	case 1:
	default:
		return No{}, l.erro(local, "mais de uma chave discriminante no mesmo nó de elemento: %s", strings.Join(achadas, ", "))
	}

	no := No{Local: local}
	switch {
	case m.valores["rect"] != nil:
		no.Tipo = TipoRetangulo
		c, err := l.caixa(m.valores["rect"], local+".rect")
		if err != nil {
			return No{}, err
		}
		no.Retangulo = &c
	case m.valores["circle"] != nil:
		no.Tipo = TipoCirculo
		d, err := l.disco(m.valores["circle"], local+".circle")
		if err != nil {
			return No{}, err
		}
		no.Circulo = &d
	case m.valores["use"] != nil:
		no.Tipo = TipoInstancia
		if no.Componente, err = l.texto(m, "use"); err != nil {
			return No{}, err
		}
		if no.Componente == "" {
			return No{}, l.erro(local, `Instância sem caminho de Componente em "use"`)
		}
	case m.valores["control"] != nil:
		no.Tipo = TipoControle
		if no.Controle, err = l.controle(m, local); err != nil {
			return No{}, err
		}
	default:
		no.Tipo = TipoSlot
		if l.emDocumento {
			return No{}, l.erro(local, "Slot só pode ser declarado em Componente, não em Documento")
		}
		if no.Slot, err = l.texto(m, "slot"); err != nil {
			return No{}, err
		}
		if no.Slot == "" {
			return No{}, l.erro(local, `Slot sem nome em "slot"`)
		}
	}

	if no.ID, err = l.texto(m, "id"); err != nil {
		return No{}, err
	}
	if no.Nota, err = l.texto(m, "note"); err != nil {
		return No{}, err
	}

	if m.valores["to"] != nil {
		// A Ligação aponta para um Frame, e Componente não conhece Frame: o
		// mesmo arquivo instanciado em dois Documentos apontaria para telas
		// diferentes, ou para nenhuma.
		if !l.emDocumento {
			return No{}, l.erro(local, "Ligação só pode ser declarada em Documento, não em Componente")
		}
		// A Ligação sai da borda de um Elemento, e Instância e Slot não
		// deixam nenhum Elemento seu no Documento resolvido: eles viram o
		// conteúdo que expandiram. Sem Elemento não há de onde a seta sair.
		if no.Tipo == TipoInstancia || no.Tipo == TipoSlot {
			return No{}, l.erro(local, `campo "to" só é permitido em Retângulo, Círculo ou Controle`)
		}
		// Repetir o gatilho não repete a Ligação: N setas idênticas saindo de
		// clones empilhados não comunicam nada, e escolher um clone seria
		// arbitrário.
		if m.valores["repeat"] != nil {
			return No{}, l.erro(local, `campo "to" não pode ser usado com "repeat"`)
		}
		if no.Destino, err = l.texto(m, "to"); err != nil {
			return No{}, err
		}
		if no.Destino == "" {
			return No{}, l.erro(local, `Ligação sem nome de Frame em "to"`)
		}
	}

	if m.valores["round"] != nil {
		if no.Tipo != TipoRetangulo {
			return No{}, l.erro(local, `campo "round" só é permitido em Retângulo`)
		}
		if no.Arredondado, err = l.booleano(m, "round"); err != nil {
			return No{}, err
		}
	}

	if m.valores["label"] != nil {
		// O Rótulo é do Retângulo e do Controle, e de mais ninguém: num
		// Círculo a faixa retangular no topo cairia fora da forma, e nem a
		// Instância nem o Slot deixam Elemento seu no Documento resolvido para
		// carregar o texto. O Controle lê o seu em l.controle, contra o
		// catálogo; aqui só o Retângulo tem o que ler.
		if no.Tipo != TipoRetangulo && no.Tipo != TipoControle {
			return No{}, l.erro(local, `campo "label" só é permitido em Retângulo ou Controle`)
		}
		if no.Tipo == TipoRetangulo {
			if no.Rotulo, err = l.texto(m, "label"); err != nil {
				return No{}, err
			}
		}
	}

	precisaDeCaixa := no.Tipo == TipoInstancia || no.Tipo == TipoSlot || no.Tipo == TipoControle
	if m.valores["box"] != nil {
		if !precisaDeCaixa {
			return No{}, l.erro(local, `campo "box" só é permitido em Instância, Slot ou Controle`)
		}
		c, err := l.caixa(m.valores["box"], local+".box")
		if err != nil {
			return No{}, err
		}
		no.Caixa = &c
	} else if precisaDeCaixa {
		return No{}, l.erro(local, `%s exige o campo "box"`, no.Tipo)
	}

	if m.valores["slots"] != nil {
		if no.Tipo != TipoInstancia {
			return No{}, l.erro(local, `campo "slots" só é permitido em Instância`)
		}
		if err := l.preenchimentos(m.valores["slots"], local, &no); err != nil {
			return No{}, err
		}
	}

	if m.valores["default"] != nil {
		if no.Tipo != TipoSlot {
			return No{}, l.erro(local, `campo "default" só é permitido em Slot`)
		}
		if no.Padrao, err = l.nos(m.valores["default"], local, "default"); err != nil {
			return No{}, err
		}
	}

	if no.Tipo != TipoControle {
		for _, campo := range chavesSoDeControle {
			if m.valores[campo] != nil {
				return No{}, l.erro(local, `campo %q só é permitido em Controle`, campo)
			}
		}
	}

	if m.valores["repeat"] != nil {
		r, err := l.repeticao(m.valores["repeat"], local+".repeat")
		if err != nil {
			return No{}, err
		}
		no.Repeticao = &r
	}

	return no, nil
}

func (l *leitor) preenchimentos(n *yaml.Node, local string, no *No) error {
	if n.Kind != yaml.MappingNode {
		return l.erro(local, `campo "slots" espera mapa, encontrou %s`, nomeDoTipo(n))
	}
	no.Preenchimentos = make(map[string]Preenchimento, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		nome := n.Content[i].Value
		alvo := fmt.Sprintf("%s.slots.%s", local, nome)
		m, err := l.mapa(n.Content[i+1], alvo, chavesDoPreenchimento)
		if err != nil {
			return err
		}
		p := Preenchimento{Local: alvo}
		temUse := m.valores["use"] != nil
		temElementos := m.valores["elements"] != nil
		switch {
		case temUse && temElementos:
			return l.erro(alvo, `preenchimento de Slot declara "use" e "elements" ao mesmo tempo`)
		case temUse:
			if p.Componente, err = l.texto(m, "use"); err != nil {
				return err
			}
		case temElementos:
			if p.Elementos, err = l.nos(m.valores["elements"], alvo, "elements"); err != nil {
				return err
			}
		default:
			return l.erro(alvo, `preenchimento de Slot exige "use" ou "elements"`)
		}
		no.Preenchimentos[nome] = p
		no.OrdemDosPreenchimentos = append(no.OrdemDosPreenchimentos, nome)
	}
	return nil
}

func (l *leitor) caixa(n *yaml.Node, local string) (Caixa, error) {
	m, err := l.mapa(n, local, chavesDaCaixa)
	if err != nil {
		return Caixa{}, err
	}
	var c Caixa
	if c.X, err = l.numero(m, "x"); err != nil {
		return Caixa{}, err
	}
	if c.Y, err = l.numero(m, "y"); err != nil {
		return Caixa{}, err
	}
	if c.L, err = l.numero(m, "w"); err != nil {
		return Caixa{}, err
	}
	if c.A, err = l.numero(m, "h"); err != nil {
		return Caixa{}, err
	}
	return c, nil
}

func (l *leitor) disco(n *yaml.Node, local string) (Disco, error) {
	m, err := l.mapa(n, local, chavesDoDisco)
	if err != nil {
		return Disco{}, err
	}
	var d Disco
	if d.X, err = l.numero(m, "x"); err != nil {
		return Disco{}, err
	}
	if d.Y, err = l.numero(m, "y"); err != nil {
		return Disco{}, err
	}
	if d.D, err = l.numero(m, "d"); err != nil {
		return Disco{}, err
	}
	return d, nil
}

func (l *leitor) repeticao(n *yaml.Node, local string) (Repeticao, error) {
	m, err := l.mapa(n, local, chavesDaRepeticao)
	if err != nil {
		return Repeticao{}, err
	}
	var r Repeticao
	quantos, err := l.numero(m, "n")
	if err != nil {
		return Repeticao{}, err
	}
	// A quantidade fica em float64: o teto de clones é da resolução, e a
	// conversão para int de um valor que estoura o int64 depende de
	// plataforma. Fracionário é recusado em vez de arredondado, para que a
	// mensagem de erro nunca cite um número que o usuário não escreveu.
	if quantos != math.Trunc(quantos) {
		return Repeticao{}, l.erro(local, `campo "n" da Repetição espera um número inteiro, encontrou %s`, formataNumero(quantos))
	}
	r.N = quantos
	if r.N < 1 {
		return Repeticao{}, l.erro(local, `campo "n" da Repetição deve ser no mínimo 1`)
	}
	if r.Eixo, err = l.texto(m, "axis"); err != nil {
		return Repeticao{}, err
	}
	if r.Eixo != "x" && r.Eixo != "y" {
		return Repeticao{}, l.erro(local, `campo "axis" da Repetição espera "x" ou "y"`)
	}
	if r.Intervalo, err = l.numero(m, "gap"); err != nil {
		return Repeticao{}, err
	}
	return r, nil
}

// mapaLido é um mapa YAML já validado contra as chaves permitidas do seu nível.
type mapaLido struct {
	local   string
	valores map[string]*yaml.Node
}

func (l *leitor) mapa(n *yaml.Node, local string, validas []string) (*mapaLido, error) {
	if n.Kind != yaml.MappingNode {
		return nil, l.erro(local, "esperava um mapa, encontrou %s", nomeDoTipo(n))
	}
	m := &mapaLido{local: local, valores: make(map[string]*yaml.Node, len(n.Content)/2)}
	for i := 0; i+1 < len(n.Content); i += 2 {
		chave := n.Content[i].Value
		if !contem(validas, chave) {
			if s := sugestao(chave, validas); s != "" {
				return nil, l.erro(local, "campo desconhecido %q; você quis dizer %q?", chave, s)
			}
			return nil, l.erro(local, "campo desconhecido %q", chave)
		}
		m.valores[chave] = n.Content[i+1]
	}
	return m, nil
}

// texto lê um campo de texto opcional. Campo ausente ou nulo devolve "".
func (l *leitor) texto(m *mapaLido, campo string) (string, error) {
	n := m.valores[campo]
	if n == nil || n.Tag == "!!null" {
		return "", nil
	}
	if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
		return "", l.erro(m.local, "campo %q espera texto, encontrou %s", campo, nomeDoTipo(n))
	}
	return n.Value, nil
}

// numero lê um campo numérico opcional. Campo ausente ou nulo devolve 0.
func (l *leitor) numero(m *mapaLido, campo string) (float64, error) {
	n := m.valores[campo]
	if n == nil || n.Tag == "!!null" {
		return 0, nil
	}
	if n.Kind != yaml.ScalarNode || (n.Tag != "!!int" && n.Tag != "!!float") {
		return 0, l.erro(m.local, "campo %q espera número, encontrou %s", campo, nomeDoTipo(n))
	}
	var v float64
	if err := n.Decode(&v); err != nil {
		return 0, l.erro(m.local, "campo %q espera número, encontrou %q", campo, n.Value)
	}
	// Infinito e NaN atravessariam a conversão para pixels como geometria
	// absurda: saturam o inteiro e escapam das comparações de área zero.
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, l.erro(m.local, "campo %q espera número finito, encontrou %q", campo, n.Value)
	}
	return v, nil
}

// booleano lê um campo booleano opcional. Campo ausente ou nulo devolve false.
func (l *leitor) booleano(m *mapaLido, campo string) (bool, error) {
	n := m.valores[campo]
	if n == nil || n.Tag == "!!null" {
		return false, nil
	}
	if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" {
		return false, l.erro(m.local, "campo %q espera booleano, encontrou %s", campo, nomeDoTipo(n))
	}
	var v bool
	if err := n.Decode(&v); err != nil {
		return false, l.erro(m.local, "campo %q espera booleano, encontrou %q", campo, n.Value)
	}
	return v, nil
}

// limiteDeDimensao é a maior dimensão de Frame aceita em pixels. O valor não
// tem nada de especial além de caber com folga num int em qualquer plataforma:
// qualquer Frame perto disso já é recusado pelo teto de área da CLI. O que
// importa é existir um teto — sem ele, `w: 1e30` satura o int64 e o Frame passa
// a valer 9223372036854775807 px, com código de saída 0.
const limiteDeDimensao = 1 << 30

// dimensao lê uma dimensão de Frame em pixels: obrigatória, maior que zero e
// dentro do que um int representa. Campo ausente é lido como zero e cai na
// mensagem de obrigatoriedade.
func (l *leitor) dimensao(m *mapaLido, campo string) (int, error) {
	v, err := l.numero(m, campo)
	if err != nil {
		return 0, err
	}
	// A comparação é feita em float64, antes da conversão: converter
	// primeiro satura no extremo do int64 e esconde o problema.
	px := math.Round(v)
	if px <= 0 {
		return 0, l.erro(m.local, "campo %q é obrigatório e deve ser maior que zero", campo)
	}
	if px > limiteDeDimensao {
		return 0, l.erro(m.local, "campo %q passa do máximo de %d pixels, encontrou %s",
			campo, limiteDeDimensao, formataNumero(v))
	}
	return int(px), nil
}

// formataNumero escreve um número como o usuário reconhece, sem notação
// científica inventada nem casas decimais que ele não digitou.
func formataNumero(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// sequencia lê a sequência guardada em campo. Campo ausente devolve nil.
func (l *leitor) sequencia(n *yaml.Node, local, campo string) ([]*yaml.Node, error) {
	if n == nil || n.Tag == "!!null" {
		return nil, nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, l.erro(local, "campo %q espera sequência, encontrou %s", campo, nomeDoTipo(n))
	}
	return n.Content, nil
}

func temChave(n *yaml.Node, chave string) bool {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == chave {
			return true
		}
	}
	return false
}

func contem(vs []string, v string) bool {
	for _, x := range vs {
		if x == v {
			return true
		}
	}
	return false
}

// nomeDoTipo devolve, em português, o tipo YAML de um nó, para mensagens de
// erro de tipo inválido.
func nomeDoTipo(n *yaml.Node) string {
	switch n.Kind {
	case yaml.SequenceNode:
		return "sequência"
	case yaml.MappingNode:
		return "mapa"
	case yaml.AliasNode:
		return "âncora"
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!int", "!!float":
			return "número"
		case "!!bool":
			return "booleano"
		case "!!null":
			return "nulo"
		}
		return "texto"
	}
	return "valor desconhecido"
}

// controle lê o nó de um Controle: resolve o nome contra o catálogo embutido,
// aceita só os campos daquele Controle e devolve os parâmetros já com os
// padrões preenchidos e validados.
//
// A permissão por campo é conferida aqui, e não pela tabela de chaves do nó,
// porque `label` é legítimo num `button` e ilegítimo num `slider`: é o Controle
// declarado que decide, não o nível.
func (l *leitor) controle(m *mapaLido, local string) (*controls.Parametros, error) {
	nome, err := l.texto(m, "control")
	if err != nil {
		return nil, err
	}
	if nome == "" {
		return nil, l.erro(local, `Controle sem nome em "control"`)
	}
	def, ok := controls.Definido(nome)
	if !ok {
		if s := sugestao(nome, controls.Nomes()); s != "" {
			return nil, l.erro(local, "nome de Controle desconhecido %q; você quis dizer %q?", nome, s)
		}
		return nil, l.erro(local, "nome de Controle desconhecido %q", nome)
	}

	p := controls.Parametros{
		Nome:    nome,
		Quantos: controls.Ausente,
		Ativo:   controls.Ausente,
		Valor:   controls.Ausente,
	}
	for _, campo := range chavesDeControle {
		if m.valores[campo] == nil {
			continue
		}
		if !aceita(def.Chaves, campo) {
			return nil, l.erro(local, "campo %q não é permitido no Controle %q", campo, nome)
		}
		switch campo {
		case "label":
			if p.Rotulo, err = l.texto(m, "label"); err != nil {
				return nil, err
			}
		case "items":
			if err := l.itens(m.valores["items"], m, local, &p); err != nil {
				return nil, err
			}
		case "active":
			if p.Ativo, err = l.inteiro(m, "active", 0, controls.LimiteDeItens); err != nil {
				return nil, err
			}
		case "value":
			if p.Valor, err = l.fracao(m, "value"); err != nil {
				return nil, err
			}
		}
	}

	def.Padroes(&p)
	if msg := def.Valida(p); msg != "" {
		return nil, l.erro(local, "%s", msg)
	}
	return &p, nil
}

// itens lê o campo "items", que aceita duas formas: um número, que gera itens
// sem texto, ou uma lista de rótulos, que gera itens com Rótulo. As duas formas
// existem para o autor escolher quanto token gastar descrevendo o Controle.
func (l *leitor) itens(n *yaml.Node, m *mapaLido, local string, p *controls.Parametros) error {
	switch n.Kind {
	case yaml.SequenceNode:
		p.Itens = make([]string, 0, len(n.Content))
		for i, filho := range n.Content {
			if filho.Kind != yaml.ScalarNode {
				return l.erro(fmt.Sprintf("%s.items[%d]", local, i),
					"item do Controle espera texto, encontrou %s", nomeDoTipo(filho))
			}
			p.Itens = append(p.Itens, filho.Value)
		}
		p.Quantos = len(p.Itens)
		return nil
	case yaml.ScalarNode:
		quantos, err := l.inteiro(m, "items", 0, controls.LimiteDeItens)
		if err != nil {
			return err
		}
		p.Quantos = quantos
		return nil
	default:
		return l.erro(local, `campo "items" espera número ou sequência, encontrou %s`, nomeDoTipo(n))
	}
}

// inteiro lê um campo numérico que só aceita inteiro dentro de uma faixa.
func (l *leitor) inteiro(m *mapaLido, campo string, minimo, maximo int) (int, error) {
	v, err := l.numero(m, campo)
	if err != nil {
		return 0, err
	}
	if v != float64(int(v)) || v < float64(minimo) || v > float64(maximo) {
		return 0, l.erro(m.local, "campo %q do Controle deve ser um inteiro entre %d e %d, encontrou %s",
			campo, minimo, maximo, strconv.FormatFloat(v, 'g', -1, 64))
	}
	return int(v), nil
}

// fracao lê um campo numérico de 0 a 100.
func (l *leitor) fracao(m *mapaLido, campo string) (float64, error) {
	v, err := l.numero(m, campo)
	if err != nil {
		return 0, err
	}
	if v < 0 || v > 100 {
		return 0, l.erro(m.local, "campo %q do Controle deve estar entre 0 e 100, encontrou %s",
			campo, strconv.FormatFloat(v, 'g', -1, 64))
	}
	return v, nil
}

// aceita diz se o campo está entre as chaves de um Controle.
func aceita(chaves []string, campo string) bool {
	for _, c := range chaves {
		if c == campo {
			return true
		}
	}
	return false
}
