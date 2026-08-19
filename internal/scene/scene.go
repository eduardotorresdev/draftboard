// Package scene define o modelo resolvido de um Documento: a projeção achatada
// onde Instâncias, Slots e Repetições já foram materializados em Elementos com
// geometria absoluta, e onde cada Elemento já carrega sua Elevação e seu Tom.
//
// Este pacote é o contrato entre a resolução e a rasterização. Não contém
// lógica de parsing nem de desenho.
package scene

// Tom é o valor de cinza atribuído automaticamente a partir da Elevação.
// A escala segue a convenção de Tailwind: 100 é quase branco, 900 quase preto.
type Tom int

const (
	// TomFrame é o Tom fixo do fundo de todo Frame.
	TomFrame Tom = 100
	// TomChrome é o extremo escuro reservado ao plano de anotação: é o Tom
	// do balão da Nota. A escada de Elevação nunca o alcança, então nenhum
	// Elemento pode ser confundido com uma anotação.
	TomChrome Tom = 900
	// tomMax é o Tom mais escuro que a escada de Elevação pode atingir.
	tomMax Tom = 800
	// tomMin é o Tom mais claro da escada.
	tomMin Tom = 100
	// passo é o deslocamento em degraus da escala por nível de Elevação.
	passo Tom = 200
)

// TomDaElevacao devolve o Tom de um Elemento a partir da sua Elevação.
//
// A Elevação 0 é o próprio Frame. Cada nível desloca o Tom em dois degraus da
// escala; ao não haver espaço para mais um passo, a direção do deslocamento
// inverte. Isso garante que Superfícies adjacentes sempre difiram por dois
// degraus e que a escala nunca esgote, a qualquer profundidade.
func TomDaElevacao(elevacao int) Tom {
	if elevacao < 0 {
		elevacao = 0
	}
	t := TomFrame
	dir := Tom(1)
	for i := 0; i < elevacao; i++ {
		prox := t + dir*passo
		if prox > tomMax || prox < tomMin {
			dir = -dir
			prox = t + dir*passo
		}
		t = prox
	}
	return t
}

// cinzas mapeia cada Tom ao seu valor de cinza de 8 bits. Os valores são a
// luminância da escala gray do Tailwind.
var cinzas = map[Tom]uint8{
	100: 0xF3,
	200: 0xE5,
	300: 0xD1,
	400: 0x9C,
	500: 0x6B,
	600: 0x4B,
	700: 0x37,
	800: 0x1F,
	900: 0x11,
}

// Cinza devolve o valor de cinza de 8 bits de um Tom. Tom fora da escala é
// arredondado para dentro dela.
func (t Tom) Cinza() uint8 {
	if t < 100 {
		t = 100
	}
	if t > 900 {
		t = 900
	}
	if c, ok := cinzas[t]; ok {
		return c
	}
	return cinzas[Tom((int(t)/100)*100)]
}

// Forma é o tipo de um Elemento. Toda forma é sólida: não existe contorno.
type Forma int

const (
	// Retangulo é o Elemento de quatro lados.
	Retangulo Forma = iota
	// Circulo é o Elemento redondo. Nunca vira elipse.
	Circulo
	// Texto é o Rótulo desenhado dentro do Elemento que o carrega. É a única
	// Forma que carrega conteúdo textual no plano do desenho, e só a resolução
	// a produz — de um Controle do catálogo ou do `label` de um Retângulo:
	// não existe nó de Texto solto no YAML.
	Texto
)

// String devolve o nome da Forma no vocabulário do domínio. É um switch, e não
// um if com retorno padrão, porque um valor novo de Forma tem que quebrar aqui
// de forma visível em vez de se disfarçar de Retângulo na árvore do inspect.
func (f Forma) String() string {
	switch f {
	case Circulo:
		return "circulo"
	case Texto:
		return "texto"
	default:
		return "retangulo"
	}
}

// Alinhamento diz onde o Rótulo se apoia dentro da área que lhe foi dada.
// Existe porque a resolução não conhece métrica de fonte: ela entrega a área,
// e só o rasterizador sabe a largura real da string.
type Alinhamento int

const (
	// AoCentro centraliza o Rótulo nos dois eixos da área.
	AoCentro Alinhamento = iota
	// AEsquerda encosta o Rótulo na borda esquerda da área, centralizado só na
	// vertical.
	AEsquerda
)

// Elemento é uma forma posicionada num Frame, com geometria já resolvida em
// pixels absolutos relativos ao canto superior esquerdo do Frame.
type Elemento struct {
	// Caminho é o identificador estável do Elemento na árvore resolvida.
	// Quando o Elemento declara um id, o último segmento é esse id.
	Caminho string
	// Local é o caminho de chaves YAML do nó que originou o Elemento,
	// atravessando a cadeia de Componentes — o mesmo texto que o Aviso e o
	// Erro imprimem. Preenchido em todo Elemento.
	//
	// Não é único: um Controle materializa várias peças e um Retângulo
	// rotulado materializa dois Elementos, todos do mesmo nó e portanto do
	// mesmo Local. Quem identifica Elemento é o Caminho; quem quiser falar do
	// nó que o autor escreveu deduplica por Local.
	Local string
	// ID é o identificador declarado no YAML. Vazio quando não declarado.
	ID string
	// Forma é Retangulo ou Circulo.
	Forma Forma
	// X e Y ancoram sempre o canto superior esquerdo da bounding box,
	// inclusive para Circulo. Em pixels, relativos ao Frame.
	X, Y float64
	// L e A são largura e altura da bounding box em pixels. Para Circulo,
	// L == A == diâmetro.
	L, A float64
	// Arredondado liga os cantos arredondados de um Retangulo.
	Arredondado bool
	// Elevacao é a distância até o Frame, contada em Superfícies empilhadas.
	Elevacao int
	// Tom é derivado da Elevacao por TomDaElevacao.
	Tom Tom
	// Origem é o caminho relativo do Componente de onde o Elemento veio.
	// Vazio quando o Elemento foi escrito diretamente no Documento.
	Origem string
	// Nota é a anotação textual aninhada no Elemento. Vazia quando não há.
	Nota string
	// Destino é o nome do Frame para onde a Ligação deste Elemento aponta,
	// ou "" quando não há Ligação. Como a Nota, não faz parte do desenho: não
	// entra na Elevação e o rasterizador a ignora. Só a Prancheta a usa.
	Destino string
	// Rotulo é o texto do Rótulo. Diferente da Nota: o Rótulo vive no plano do
	// desenho e participa da Elevação; a Nota vive no plano de anotação e não
	// participa.
	//
	// Aparece em dois Elementos por Rótulo, e Rotulo != "" não implica
	// Forma == Texto: o Retângulo que declarou `label` também o carrega, e ali
	// ele existe só para o `inspect`. A área de referência do Rótulo — a que se
	// desenha, a que se recorta e a que um dia se mede — é sempre a do Elemento
	// de Forma Texto. Quem desenha e quem mede despacham por Forma, nunca por
	// Rotulo != "".
	Rotulo string
	// Controle é o nome de catálogo do Controle que materializou este Elemento,
	// preenchido tanto na cabeça quanto nos Elementos internos. Vazio quando o
	// Elemento não veio de um Controle.
	Controle string
	// Detalhe são os parâmetros do Controle já formatados para a árvore do
	// inspect, no formato "itens=3 ativo=2". Só a cabeça do Controle o carrega,
	// e fica vazio quando não há parâmetro que valha imprimir.
	Detalhe string
	// Interno marca as peças de dentro de um Controle, que existem no desenho
	// mas não na árvore: o Controle é fechado também na leitura. A cabeça do
	// Controle não é interna.
	Interno bool
	// Alinhamento posiciona o Rotulo dentro da área do Elemento. Só tem efeito
	// quando Forma é Texto.
	Alinhamento Alinhamento
}

// Camada é um grupo nomeado e ordenado de Elementos dentro de um Frame.
type Camada struct {
	Nome      string
	Elementos []Elemento
}

// Frame é um viewport de dimensões declaradas em pixels que contém Camadas.
type Frame struct {
	Nome string
	// L e A são as dimensões declaradas em pixels, antes do fator de escala.
	L, A    int
	Camadas []Camada
}

// Documento é a unidade que se renderiza: uma lista de Frames.
type Documento struct {
	// Nome deriva do nome do arquivo, sem diretório e sem extensão.
	Nome   string
	Frames []Frame
}

// Aviso é um problema tolerável: a renderização ocorre mesmo assim.
type Aviso struct {
	// Arquivo é o caminho do arquivo YAML onde o problema foi detectado.
	Arquivo string
	// Local é a localização dentro do arquivo, no formato de caminho de
	// chaves YAML. Ex.: "frames[0].layers[1].elements[3]".
	Local string
	// Msg descreve o problema em português, usando o vocabulário do domínio.
	Msg string
}

func (a Aviso) String() string {
	return "aviso: " + a.Arquivo + ": " + a.Local + ": " + a.Msg
}

// Erro é um problema que impede a resolução. Carrega arquivo e localização para
// que a mensagem aponte o ponto exato em cadeias de Componentes.
type Erro struct {
	// Arquivo é o caminho do arquivo YAML onde o problema foi detectado.
	Arquivo string
	// Local é o caminho de chaves YAML dentro do arquivo, atravessando a
	// cadeia de Componentes. Ex.: "frames[0].layers[0].elements[2] -> ./card.yaml: elements[1]".
	Local string
	// Msg descreve o problema em português, usando o vocabulário do domínio.
	Msg string
}

func (e *Erro) Error() string {
	if e.Local == "" {
		return e.Arquivo + ": " + e.Msg
	}
	return e.Arquivo + ": " + e.Local + ": " + e.Msg
}
