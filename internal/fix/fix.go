// Package fix troca, no YAML do autor, o valor de `w` de um Retângulo, sem
// reserializar o Documento.
//
// A cirurgia é de bytes, e não de árvore: reserializar perderia comentários,
// ordem das chaves e estilo de bloco. O autor não pediu para reformatar o
// arquivo dele — pediu um número maior.
//
// A posição no arquivo é conhecimento deste pacote, e de mais nenhum. Mantê-la
// fora de `schema.No` é o que impede o resto do sistema de começar a raciocinar
// sobre linhas de arquivo.
package fix

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// As razões pelas quais a máquina não consegue alargar um nó sozinha. São
// literais do contrato, e viajam até a mensagem que o autor lê: quem recebe um
// Erro precisa saber o que fazer com ele.
const (
	// RazaoSemLargura cobre também o nó que o Local não endereça e o `w` que
	// não é um escalar simples. Do ponto de vista do autor é a mesma coisa:
	// não há, naquele nó, um número solto que a máquina possa trocar.
	RazaoSemLargura = `o Retângulo não declara "w"`
	// RazaoRepeticao vale para o nó repetido e para qualquer um dentro de um
	// nó repetido: o `w` é o passo da Repetição.
	RazaoRepeticao = "o Retângulo está dentro de uma Repetição, e alargá-lo reposiciona os clones"
)

// Troca é uma largura trocada: o Local do nó, o valor que estava declarado e o
// que passou a estar. `De` é o valor decodificado, e não o texto original — um
// `w: 2e1` é reportado como 20, que é o número com que o autor de fato desenha.
type Troca struct {
	Local    string
	De, Para float64
}

// Arquivo é o YAML cru já lido e indexado, com as trocas ainda por gravar.
//
// É um punhado com estado, e não uma lista de correções aplicada de uma vez,
// porque a mesma leitura serve ao predicado de corrigibilidade e à cirurgia:
// parsear duas vezes abriria a janela para o arquivo mudar entre uma e outra.
type Arquivo struct {
	caminho string
	dados   []byte
	// inicios é o deslocamento em bytes do começo de cada linha, indexado a
	// partir de zero para a linha 1.
	inicios []int
	raiz    *yaml.Node
	tamanho int64
	mtime   time.Time
	trocas  []troca
}

// troca é uma Troca já resolvida na árvore: guarda o nó do valor para que a
// posição só seja convertida em deslocamento na hora de gravar.
type troca struct {
	Troca
	valor *yaml.Node
}

// leArquivo é a leitura do Documento, isolada num ponto de costura. A guarda de
// mtime só se prova com o arquivo mudando ENTRE a leitura e o segundo stat, e
// não há como provocar essa janela de fora sem depender de tempo de relógio.
var leArquivo = os.ReadFile

// Abre lê o arquivo YAML cru e indexa os nós endereçáveis.
//
// O tamanho e o mtime são amostrados ANTES da leitura e conferidos depois. É a
// guarda que Grava usa para saber se o Documento mudou no disco, e amostrá-la
// só depois de ler a inutilizaria: o editor do autor que salvasse entre a
// leitura e o stat faria Grava comparar com o mtime pós-edição, aprovar, e
// gravar o buffer velho por cima — a edição do autor desapareceria em silêncio.
func Abre(arquivo string) (*Arquivo, error) {
	antes, err := os.Stat(arquivo)
	if err != nil {
		return nil, erroDeArquivo(arquivo, err, "não foi possível ler o Documento")
	}
	dados, err := leArquivo(arquivo)
	if err != nil {
		return nil, erroDeArquivo(arquivo, err, "não foi possível ler o Documento")
	}
	depois, err := os.Stat(arquivo)
	if err != nil {
		return nil, erroDeArquivo(arquivo, err, "não foi possível ler o Documento")
	}
	if depois.Size() != antes.Size() || !depois.ModTime().Equal(antes.ModTime()) {
		return nil, &scene.Erro{Arquivo: arquivo,
			Msg: "o arquivo mudou no disco desde a leitura; rode o comando de novo"}
	}
	info := antes
	var doc yaml.Node
	if err := yaml.Unmarshal(dados, &doc); err != nil {
		return nil, &scene.Erro{Arquivo: arquivo, Msg: "YAML inválido: " + err.Error()}
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, &scene.Erro{Arquivo: arquivo, Msg: "esperava um mapa no topo do arquivo"}
	}
	return &Arquivo{
		caminho: arquivo,
		dados:   dados,
		inicios: iniciosDeLinha(dados),
		raiz:    doc.Content[0],
		tamanho: info.Size(),
		mtime:   info.ModTime(),
	}, nil
}

// Alargavel diz se o Local endereça um `rect` com `w` alargável pela máquina, e
// a razão quando não.
func (a *Arquivo) Alargavel(local string) (bool, string) {
	_, _, razao := a.larguraDe(local)
	return razao == "", razao
}

// larguraDe resolve o Local até o nó do valor de `w`, devolvendo também o
// número já decodificado. A razão vazia é o único sinal de sucesso.
func (a *Arquivo) larguraDe(local string) (*yaml.Node, float64, string) {
	no, repetido, ok := a.resolve(local)
	if !ok {
		return nil, 0, RazaoSemLargura
	}
	retangulo, ok := filho(no, "rect")
	if !ok {
		return nil, 0, RazaoSemLargura
	}
	if repetido {
		return nil, 0, RazaoRepeticao
	}
	valor, ok := filho(retangulo, "w")
	if !ok || !escalarSimples(valor) {
		return nil, 0, RazaoSemLargura
	}
	var v float64
	if err := valor.Decode(&v); err != nil || !finito(v) {
		return nil, 0, RazaoSemLargura
	}
	return valor, v, ""
}

// Alarga registra a troca do `w` de um nó. Nada é escrito até Grava.
func (a *Arquivo) Alarga(local string, w float64) error {
	if !finito(w) || w <= 0 {
		return &scene.Erro{Arquivo: a.caminho, Local: local,
			Msg: fmt.Sprintf("largura inválida para alargar o Retângulo: %s", formata(w))}
	}
	valor, de, razao := a.larguraDe(local)
	if razao != "" {
		return &scene.Erro{Arquivo: a.caminho, Local: local,
			Msg: "não foi possível alargar o Retângulo: " + razao}
	}
	a.trocas = append(a.trocas, troca{Troca: Troca{Local: local, De: de, Para: w}, valor: valor})
	return nil
}

// Grava escreve todas as trocas de uma vez e devolve o que trocou, na ordem em
// que foram registradas.
//
// O arquivo inteiro é montado em memória e trocado por os.Rename: truncar em
// cima do original deixaria o Documento do autor pela metade se o processo
// morresse no meio da escrita.
func (a *Arquivo) Grava() ([]Troca, error) {
	if len(a.trocas) == 0 {
		return nil, nil
	}
	info, err := os.Stat(a.caminho)
	if err != nil {
		return nil, erroDeArquivo(a.caminho, err, "não foi possível gravar o Documento")
	}
	// Reler não bastaria: os deslocamentos foram medidos contra o buffer da
	// leitura, e aplicá-los num arquivo que mudou embaralharia o YAML.
	if info.Size() != a.tamanho || !info.ModTime().Equal(a.mtime) {
		return nil, &scene.Erro{Arquivo: a.caminho,
			Msg: "o arquivo mudou no disco desde a leitura; rode o comando de novo"}
	}

	novo, err := a.buffer()
	if err != nil {
		return nil, err
	}
	// O symlink é resolvido até o alvo real: escrever ao lado do link e
	// renomear por cima dele trocaria o link por um arquivo comum.
	alvo := a.caminho
	if real, err := filepath.EvalSymlinks(alvo); err == nil {
		alvo = real
	}
	if err := escreveNoLugar(alvo, novo, info.Mode().Perm()); err != nil {
		return nil, err
	}

	feitas := make([]Troca, 0, len(a.trocas))
	for _, t := range a.trocas {
		feitas = append(feitas, t.Troca)
	}
	a.trocas = nil
	return feitas, nil
}

// buffer monta o arquivo inteiro com as trocas aplicadas.
//
// Os deslocamentos são todos medidos contra o buffer ORIGINAL e aplicados em
// ordem decrescente de posição: um `20` que vira `47` empurraria em um byte
// tudo o que viesse depois dele na mesma linha, e a troca seguinte cairia no
// lugar errado.
func (a *Arquivo) buffer() ([]byte, error) {
	type corte struct {
		inicio, fim int
		texto       string
	}
	cortes := make([]corte, 0, len(a.trocas))
	for _, t := range a.trocas {
		inicio, ok := a.deslocamento(t.valor.Line, t.valor.Column)
		if !ok || inicio+len(t.valor.Value) > len(a.dados) {
			return nil, &scene.Erro{Arquivo: a.caminho, Local: t.Local,
				Msg: "não foi possível localizar a largura declarada no arquivo"}
		}
		cortes = append(cortes, corte{inicio, inicio + len(t.valor.Value), formata(t.Para)})
	}
	sort.Slice(cortes, func(i, j int) bool { return cortes[i].inicio > cortes[j].inicio })

	novo := make([]byte, len(a.dados))
	copy(novo, a.dados)
	for _, c := range cortes {
		novo = append(novo[:c.inicio], append([]byte(c.texto), novo[c.fim:]...)...)
	}
	return novo, nil
}

// escreveNoLugar grava o conteúdo num temporário do MESMO diretório e o
// renomeia por cima do alvo: o rename é atômico dentro de um sistema de
// arquivos, e um temporário em /tmp poderia estar em outro.
func escreveNoLugar(alvo string, dados []byte, modo fs.FileMode) error {
	// A permissão do alvo é conferida antes de qualquer escrita: o rename só
	// depende de poder escrever no DIRETÓRIO, e sem esta conferência um
	// Documento marcado como somente-leitura seria trocado assim mesmo.
	if f, err := os.OpenFile(alvo, os.O_WRONLY, 0); err != nil {
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	} else if err := f.Close(); err != nil {
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	dir := filepath.Dir(alvo)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(alvo)+".*")
	if err != nil {
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	nome := tmp.Name()
	defer os.Remove(nome)
	if _, err := tmp.Write(dados); err != nil {
		tmp.Close()
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	// O conteúdo é sincronizado ANTES do rename: o rename é atômico para o
	// processo, não para a máquina. Sem isto, uma queda de energia logo depois
	// de o comando devolver 0 deixaria o Documento do autor com zero byte —
	// exatamente o que a escrita em temporário existe para evitar.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	if err := tmp.Close(); err != nil {
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	// O modo do original é copiado antes do rename: o temporário nasce 0600 e
	// o Documento do autor não pode mudar de permissão por ter sido corrigido.
	if err := os.Chmod(nome, modo); err != nil {
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	if err := os.Rename(nome, alvo); err != nil {
		return erroDeArquivo(alvo, err, "não foi possível gravar o Documento")
	}
	return nil
}

// erroDeArquivo traduz o erro cru do sistema operacional para o formato de erro
// do domínio: a CLI imprime `erro: <arquivo>: <msg>`, e um *fs.PathError cru
// ali dentro repetiria o caminho e falaria inglês.
func erroDeArquivo(arquivo string, err error, msg string) error {
	switch {
	case errors.Is(err, fs.ErrPermission):
		msg += ": permissão negada"
	case errors.Is(err, fs.ErrNotExist):
		msg = "arquivo não encontrado"
	}
	return &scene.Erro{Arquivo: arquivo, Msg: msg}
}

// deslocamento converte a posição de um nó do YAML em deslocamento de bytes.
//
// A coluna do yaml.v3 conta RUNAS, não bytes: num projeto inteiro em português,
// um `label` acentuado na mesma linha do `w` deslocaria o corte e corromperia o
// arquivo do autor.
func (a *Arquivo) deslocamento(linha, coluna int) (int, bool) {
	if linha < 1 || linha > len(a.inicios) || coluna < 1 {
		return 0, false
	}
	off := a.inicios[linha-1]
	for c := 1; c < coluna; c++ {
		if off >= len(a.dados) || a.dados[off] == '\n' {
			return 0, false
		}
		_, n := utf8.DecodeRune(a.dados[off:])
		off += n
	}
	return off, true
}

func iniciosDeLinha(dados []byte) []int {
	inicios := []int{0}
	for i, b := range dados {
		if b == '\n' {
			inicios = append(inicios, i+1)
		}
	}
	return inicios
}

// resolve percorre o Local até o nó do elemento que ele endereça, e reporta se
// algum nó do caminho — inclusive o próprio — está sob uma Repetição.
//
// A gramática aceita é fechada:
//
//	frames[i] .layers[i] .elements[i] .default[i] .slots.<nome>
//
// Local de Componente carrega " -> " e nunca chega aqui: quem o diagnostica já
// o classificou como Erro. Local que não case com a gramática devolve false, e
// nunca entra em pânico: é o predicado que decide a categoria do diagnóstico, e
// um pânico ali derrubaria o comando inteiro por causa de um nó só.
func (a *Arquivo) resolve(local string) (*yaml.Node, bool, bool) {
	atual := a.raiz
	repetido := false
	resto := local
	for resto != "" {
		if strings.HasPrefix(resto, "slots.") {
			mapa, ok := filho(atual, "slots")
			if !ok {
				return nil, false, false
			}
			valor, r, ok := slotDeNomeMaisLongo(mapa, resto[len("slots."):])
			if !ok {
				return nil, false, false
			}
			atual, resto = valor, r
			continue
		}
		abre := strings.IndexByte(resto, '[')
		fecha := strings.IndexByte(resto, ']')
		if abre <= 0 || fecha < abre {
			return nil, false, false
		}
		chave := resto[:abre]
		if !sequenciaConhecida(chave) {
			return nil, false, false
		}
		i, err := strconv.Atoi(resto[abre+1 : fecha])
		if err != nil || i < 0 {
			return nil, false, false
		}
		seq, ok := filho(atual, chave)
		if !ok || seq.Kind != yaml.SequenceNode || i >= len(seq.Content) {
			return nil, false, false
		}
		atual = seq.Content[i]
		if _, tem := filho(atual, "repeat"); tem {
			repetido = true
		}
		resto = resto[fecha+1:]
		if resto != "" {
			if resto[0] != '.' {
				return nil, false, false
			}
			resto = resto[1:]
		}
	}
	return atual, repetido, true
}

// sequenciaConhecida lista as chaves de sequência que a gramática do Local
// atravessa. Fechada de propósito: uma chave desconhecida é Local que não
// endereça nada, e não um nó a adivinhar.
func sequenciaConhecida(chave string) bool {
	switch chave {
	case "frames", "layers", "elements", "default":
		return true
	}
	return false
}

// slotDeNomeMaisLongo casa o nome de Slot mais longo que de fato exista entre
// as chaves do mapa, e devolve o que sobrou do Local.
//
// O nome mais longo, e não o primeiro ponto, porque nada proíbe um Slot de se
// chamar "corpo.principal": pelo primeiro ponto, o Local dele endereçaria um
// Slot chamado "corpo" que talvez nem exista.
func slotDeNomeMaisLongo(mapa *yaml.Node, resto string) (*yaml.Node, string, bool) {
	if mapa.Kind != yaml.MappingNode {
		return nil, "", false
	}
	melhor := -1
	var valor *yaml.Node
	for i := 0; i+1 < len(mapa.Content); i += 2 {
		nome := mapa.Content[i].Value
		if len(nome) <= melhor || !strings.HasPrefix(resto, nome) {
			continue
		}
		sobra := resto[len(nome):]
		if sobra != "" && sobra[0] != '.' {
			continue
		}
		melhor, valor = len(nome), mapa.Content[i+1]
	}
	if valor == nil {
		return nil, "", false
	}
	return valor, strings.TrimPrefix(resto[melhor:], "."), true
}

// filho devolve o valor de uma chave de um nó de mapa.
//
// Varre até o FIM e devolve a ÚLTIMA ocorrência, porque é essa a semântica da
// decodificação: schema monta um mapa iterando os pares, e a chave repetida
// sobrescreve a anterior. Devolvendo a primeira, a cirurgia trocaria um `w` que
// o desenho ignora — o Aviso continuaria saindo, e toda execução seguinte
// reescreveria o arquivo imprimindo `w 40 → 40`, sem nunca convergir.
func filho(no *yaml.Node, chave string) (*yaml.Node, bool) {
	if no == nil || no.Kind != yaml.MappingNode {
		return nil, false
	}
	var valor *yaml.Node
	for i := 0; i+1 < len(no.Content); i += 2 {
		if no.Content[i].Value == chave {
			valor = no.Content[i+1]
		}
	}
	return valor, valor != nil
}

// escalarSimples reporta se o nó é um número escrito solto no arquivo — o único
// caso em que trocar os bytes do escalar troca aquela largura e mais nada.
//
// Âncora e apelido ficam de fora porque o mesmo valor pode estar sendo lido em
// outro ponto do Documento, e alargar um Retângulo alargaria os outros junto.
func escalarSimples(no *yaml.Node) bool {
	return no != nil && no.Kind == yaml.ScalarNode && no.Style == 0 &&
		no.Anchor == "" && (no.Tag == "!!int" || no.Tag == "!!float")
}

func finito(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// formata escreve o número como o autor o reconhece, sem casas decimais que ele
// não digitou.
func formata(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
