// Package schema decodifica arquivos YAML de Documento e de Componente na
// árvore tal como foi declarada, antes de qualquer resolução de geometria.
//
// A decodificação é estrita: toda chave desconhecida, em qualquer nível, é
// erro, acompanhada da sugestão da chave válida mais próxima. Todo erro carrega
// o arquivo e a localização em caminho de chaves YAML.
package schema

import "github.com/eduardotorresdev/draftboard/internal/controls"

// Documento é um arquivo YAML que declara Frames. É a unidade que se renderiza.
type Documento struct {
	// Arquivo é o caminho do arquivo de onde o Documento foi lido.
	Arquivo string
	// Nome deriva do nome do arquivo, sem diretório e sem extensão.
	Nome   string
	Frames []Frame
}

// Componente é um arquivo YAML escrito num espaço de coordenadas próprio de
// 0 a 100, sem Frames, reutilizável em qualquer tamanho.
type Componente struct {
	// Arquivo é o caminho do arquivo de onde o Componente foi lido.
	Arquivo   string
	Elementos []No
}

// Frame é um viewport de dimensões declaradas em pixels que contém Camadas.
type Frame struct {
	Nome string
	// L e A são as dimensões declaradas em pixels.
	L, A    int
	Camadas []Camada
	// Local é o caminho de chaves YAML deste Frame no arquivo.
	Local string
}

// Camada é um grupo nomeado e ordenado de nós dentro de um Frame.
type Camada struct {
	Nome      string
	Elementos []No
	// Local é o caminho de chaves YAML desta Camada no arquivo.
	Local string
}

// Tipo é a chave discriminante de um nó da árvore declarada.
type Tipo int

const (
	// TipoRetangulo é o nó `rect`.
	TipoRetangulo Tipo = iota
	// TipoCirculo é o nó `circle`.
	TipoCirculo
	// TipoInstancia é o nó `use`: o uso de um Componente.
	TipoInstancia
	// TipoSlot é o nó `slot`: a declaração de um Slot num Componente.
	TipoSlot
	// TipoControle é o nó `control`: o uso de um Controle do catálogo embutido.
	TipoControle
)

func (t Tipo) String() string {
	switch t {
	case TipoCirculo:
		return "Círculo"
	case TipoInstancia:
		return "Instância"
	case TipoSlot:
		return "Slot"
	case TipoControle:
		return "Controle"
	default:
		return "Retângulo"
	}
}

// Caixa é um retângulo no espaço local de quem o declara, em porcentagem do
// eixo correspondente.
type Caixa struct{ X, Y, L, A float64 }

// Disco é um Círculo no espaço local de quem o declara. D é o diâmetro em
// porcentagem da largura do espaço, usado nos dois eixos.
type Disco struct{ X, Y, D float64 }

// Repeticao é a clonagem de um nó ao longo de um eixo, com espaçamento fixo.
type Repeticao struct {
	// N é a quantidade de clones declarada, já arredondada para inteiro. É
	// float64 porque o teto de clones é da resolução: converter para int
	// antes dela faria um valor que estoura o int64 saturar num extremo
	// diferente conforme a plataforma, e a mesma entrada seria aceita numa
	// e recusada noutra.
	N float64
	// Eixo é "x" ou "y".
	Eixo string
	// Intervalo é o espaçamento entre clones, em porcentagem do eixo do
	// espaço local.
	Intervalo float64
}

// Preenchimento é o conteúdo entregue a um Slot por quem instancia o
// Componente: ou outro Componente, ou nós escritos inline.
type Preenchimento struct {
	// Componente é o caminho relativo do Componente, ou "" quando o
	// preenchimento é inline.
	Componente string
	// Elementos são os nós inline, ou nil quando o preenchimento é um
	// Componente.
	Elementos []No
	// Local é o caminho de chaves YAML deste preenchimento no arquivo.
	Local string
}

// No é um nó de elemento da árvore declarada: um Retângulo, um Círculo, uma
// Instância ou a declaração de um Slot.
type No struct {
	Tipo Tipo
	// Local é o caminho de chaves YAML deste nó no arquivo.
	Local string
	// ID é o identificador declarado, ou "" quando ausente.
	ID string
	// Nota é a anotação textual aninhada, ou "" quando ausente.
	Nota string
	// Rotulo é o texto declarado em `label` num nó TipoRetangulo, ou "" quando
	// ausente. O Controle guarda o seu em Controle.Rotulo: lá o campo é
	// validado contra o catálogo, e nem todo Controle o aceita.
	Rotulo string
	// Destino é o nome do Frame para onde a Ligação aponta, ou "" quando o
	// nó não declara Ligação.
	Destino string
	// Retangulo é a geometria de um nó TipoRetangulo.
	Retangulo *Caixa
	// Circulo é a geometria de um nó TipoCirculo.
	Circulo *Disco
	// Arredondado liga os cantos arredondados de um Retângulo.
	Arredondado bool
	// Componente é o caminho relativo do Componente de um nó TipoInstancia.
	Componente string
	// Slot é o nome do Slot de um nó TipoSlot.
	Slot string
	// Caixa é a caixa no espaço do pai de um nó TipoInstancia ou TipoSlot.
	Caixa *Caixa
	// Repeticao é a Repetição declarada, ou nil quando ausente.
	Repeticao *Repeticao
	// Preenchimentos são os Slots preenchidos por uma Instância, indexados
	// pelo nome do Slot.
	Preenchimentos map[string]Preenchimento
	// OrdemDosPreenchimentos guarda os nomes de Preenchimentos na ordem
	// declarada, para que a resolução seja determinística.
	OrdemDosPreenchimentos []string
	// Padrao é o conteúdo padrão de um nó TipoSlot, ou nil quando ausente.
	Padrao []No
	// Controle são os parâmetros do Controle quando Tipo é TipoControle, já
	// validados contra o catálogo e com os padrões preenchidos.
	Controle *controls.Parametros
}
