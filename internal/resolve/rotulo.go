package resolve

import (
	"math"

	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/schema"
)

// alturaDoRotulo é a altura, em px do espaço do Frame, da caixa que a resolução
// reserva ao Rótulo de um Retângulo.
//
// É fixa, e não uma fração da altura do Retângulo, porque o tamanho da fonte já
// é uma fração da caixa: numa região de 400 px de altura, a fração daria uma
// fonte de 180 px, que é mockup, não wireframe.
const alturaDoRotulo = 28.0

// respiroDoRotulo é a folga em cada ponta horizontal, em px do espaço do Frame,
// entre a borda do Retângulo e a área do Rótulo.
//
// É geometria, e é entregue já descontada da área do Elemento de Texto: se
// morasse no desenhista, a Prancheta e o WebP teriam cada um a sua regra de
// afastamento, e o mesmo Documento sairia diferente nos dois.
const respiroDoRotulo = 6.0

// SegmentoDoRotulo é o segmento fixo que o Rótulo acrescenta ao caminho do
// Retângulo. Fixo de propósito: sem ele, caminhoUnico desambiguaria o Rótulo
// com um sufixo ~2 e o painel de inspeção da Prancheta mostraria um Elemento
// fantasma ao lado do Retângulo que o carrega.
//
// É exportado para que o diagnóstico ache o Retângulo dono pelo Caminho sem
// escrever uma segunda grafia do sufixo: duas grafias que divergissem fariam o
// diagnóstico medir o Rótulo contra a largura errada.
const SegmentoDoRotulo = "rotulo"

// rotuloDoRetangulo materializa o Rótulo declarado num nó `rect`, logo depois
// do Retângulo que o carrega e na mesma Camada.
//
// A geometria entregue aqui é a do próprio Retângulo, e é provisória: a posição
// final depende da contenção, que só se conhece com o Frame inteiro achatado.
// Quem a corrige é posicionaRotulos.
//
// **Invariante de adjacência**: o Rótulo é sempre o Elemento imediatamente
// seguinte ao seu Retângulo. É por ela que posicionaRotulos identifica o dono.
// Imediatamente depois, e não no fim da Camada, porque a Superfície do Rótulo
// tem que ser o Retângulo que o carrega: é dele que vêm a Elevação e o Tom que
// dão contraste ao texto. Empilhado no fim, um filho qualquer viraria a
// Superfície.
//
// A invariante vale só até atribuiElevacao: depois dela, sobeRotulos leva cada
// Rótulo para o fim da Camada, para que os filhos não o apaguem da imagem.
func (r *resolucao) rotuloDoRetangulo(no schema.No, caminho string, ctx contexto, esp espaco, dest *[]scene.Elemento, x, y, l, a float64) error {
	if no.Rotulo == "" {
		return nil
	}
	local := ctx.prefixo + no.Local
	// O Rótulo é uma materialização a mais e paga o orçamento do Frame, como
	// as peças de um Controle pagam: sem isso, `rect` com `label` dentro de uma
	// Repetição materializa o dobro de Elementos sob o mesmo teto.
	if err := r.debita(1, local); err != nil {
		return err
	}
	// Sem aviso geométrico próprio: a geometria do Rótulo é derivada, o autor
	// não a declarou, e um segundo aviso no mesmo Local não teria como ser
	// lido como coisa diferente do aviso do Retângulo.
	// O espaço é o do Retângulo que carrega o Rótulo: quem sugere um `w` maior
	// tem que desfazer a projeção contra a mesma base que o dono usou.
	r.emite(dest, esp, scene.Elemento{
		Caminho: caminhoDaPeca(caminho, SegmentoDoRotulo),
		Forma:   scene.Texto,
		X:       x,
		Y:       y,
		L:       l,
		A:       a,
		Origem:  ctx.origem,
		Rotulo:  no.Rotulo,
		Local:   local,
		Interno: true,
	})
	return nil
}

// posicionaRotulos corrige a geometria e o Alinhamento de cada Rótulo de
// Retângulo do Frame a partir da contenção, e roda antes de atribuiElevacao:
// ela precisa do Frame inteiro já achatado, e a Elevação precisa da geometria
// final.
//
// Retângulo que contém outro Elemento apoia o Rótulo numa faixa no topo, à
// esquerda, fora do caminho dos filhos; Retângulo vazio o centraliza na
// vertical. Não existe chave para forçar: ganhar um filho move o Rótulo
// sozinho, como ganhar uma Superfície já muda o Tom sozinho.
func posicionaRotulos(camadas []scene.Camada) {
	// A varredura olha todos os Elementos do Frame, de todas as Camadas, como
	// a Elevação já faz: um filho declarado numa Camada acima continua sendo
	// um filho.
	var todos []*scene.Elemento
	for i := range camadas {
		for j := range camadas[i].Elementos {
			todos = append(todos, &camadas[i].Elementos[j])
		}
	}
	for k, e := range todos {
		if !ehRotuloDeRetangulo(*e) {
			continue
		}
		// Invariante de adjacência, estabelecida por rotuloDoRetangulo: o
		// Elemento anterior é o Retângulo dono deste Rótulo.
		if k == 0 {
			continue
		}
		posiciona(e, todos[k-1], todos)
	}
}

// ehRotuloDeRetangulo diz se o Elemento é o Rótulo materializado por um `label`
// de `rect`.
//
// O despacho é por Forma, e nunca por `Rotulo != ""`: o Retângulo também carrega
// o texto na cabeça, e quem varresse por ele acharia dois Elementos por Rótulo.
// O Rótulo de Controle fica de fora porque a caixa dele é escolha do catálogo, e
// a pintura dele já está resolvida na ordem das peças do Controle.
func ehRotuloDeRetangulo(e scene.Elemento) bool {
	return e.Forma == scene.Texto && e.Controle == ""
}

// sobeRotulos move cada Rótulo de Retângulo para o fim da sua Camada,
// preservando a ordem relativa entre eles. Roda depois de atribuiElevacao, e é
// a segunda metade do arranjo que dá as duas coisas ao mesmo tempo.
//
// Emitido junto do dono, o Rótulo ganha dele a Elevação e o Tom — mas é pintado
// antes dos filhos, e uma barra de cabeçalho apoiada no mesmo topo o apaga da
// imagem em silêncio, com o `inspect` ainda dizendo que ele está lá. Movido só
// depois da Elevação, o texto é pintado por cima de tudo que a Camada desenha
// sem perder o contraste que veio do Retângulo. Uma Camada posterior continua
// cobrindo: isso é a Elevação funcionando, não defeito.
func sobeRotulos(camadas []scene.Camada) {
	for i := range camadas {
		elementos := camadas[i].Elementos
		var rotulos []scene.Elemento
		restantes := elementos[:0]
		for _, e := range elementos {
			if ehRotuloDeRetangulo(e) {
				rotulos = append(rotulos, e)
				continue
			}
			restantes = append(restantes, e)
		}
		camadas[i].Elementos = append(restantes, rotulos...)
	}
}

// posiciona escreve no Rótulo a caixa que ele ocupa dentro do seu Retângulo.
func posiciona(rotulo, retangulo *scene.Elemento, todos []*scene.Elemento) {
	// A altura satura na do Retângulo: uma faixa de 28 px dentro de um bloco
	// de 10 px transbordaria por baixo dele e apagaria o vizinho.
	a := math.Min(alturaDoRotulo, retangulo.A)

	rotulo.X = retangulo.X + respiroDoRotulo
	rotulo.L = math.Max(0, retangulo.L-2*respiroDoRotulo)
	rotulo.A = a
	if temFilho(retangulo, todos) {
		rotulo.Y = retangulo.Y
		rotulo.Alinhamento = scene.AEsquerda
		return
	}
	rotulo.Y = retangulo.Y + (retangulo.A-a)/2
	rotulo.Alinhamento = scene.AoCentro
}

// temFilho diz se o Retângulo contém geometricamente algum outro Elemento do
// Frame, pela mesma relação que decide a Superfície de cada Elemento.
func temFilho(retangulo *scene.Elemento, todos []*scene.Elemento) bool {
	for _, outro := range todos {
		// A exclusão é por identidade de ponteiro, e não por geometria:
		// contemGeometricamente é inclusiva nas quatro bordas, então todo
		// Retângulo contém a si mesmo. Sem esta linha, nenhum Retângulo
		// rotulado seria vazio e o caso centrado nunca aconteceria.
		if outro == retangulo {
			continue
		}
		// O Elemento de Texto não conta como filho: senão o próprio Rótulo
		// faria todo Retângulo rotulado parecer cheio.
		if outro.Forma == scene.Texto {
			continue
		}
		if contemGeometricamente(retangulo, outro) {
			return true
		}
	}
	return false
}
