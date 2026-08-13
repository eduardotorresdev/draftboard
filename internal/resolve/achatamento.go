package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/schema"
)

// LimiteDeProfundidade é o número máximo de níveis de Componente que uma cadeia
// de Instâncias pode atravessar. Passar dele é erro: recursão acidental falha
// rápido em vez de consumir memória até o processo morrer.
const LimiteDeProfundidade = 16

// LimiteDeClones é o número máximo de clones de uma única Repetição.
const LimiteDeClones = 1_000

// LimiteDeElementos é o número máximo de Elementos materializados num Frame.
// Repetições encadeadas por Componentes multiplicam: oito Componentes com
// `repeat: {n: 10}` cada, dentro do limite de profundidade, materializariam 10⁸
// Elementos a partir de um punhado de bytes de YAML.
//
// O orçamento é debitado por clone de nó, antes de o nó ser resolvido, e não
// no nascimento do Elemento. Todo Elemento é o clone de exatamente um nó, então
// o teto continua valendo ao pé da letra para Elementos; e as Instâncias e os
// Slots, que expandem sem materializar nada por conta própria, também pagam.
// Sem isso, uma cadeia de Repetições sobre um Componente vazio atravessa o teto
// sem nunca encostar nele: não estoura a memória, simplesmente não termina.
const LimiteDeElementos = 10_000

// LimiteDeAvisos é o número máximo de avisos guardados num Documento. Acima
// dele os avisos são contados e omitidos: um `repeat` de dez bytes gera dez mil
// avisos por Frame, e a lista inteira na memória é o mesmo problema em outra
// roupa.
const LimiteDeAvisos = 1_000

// espaco é o retângulo em pixels do Frame onde um espaço de coordenadas local
// de 0 a 100 está projetado. O espaço do Frame é o próprio Frame na origem; o
// espaço de uma Instância ou de um Slot é a caixa que a recebe.
type espaco struct{ X, Y, L, A float64 }

// retangulo projeta uma caixa declarada em porcentagem para pixels absolutos.
func (e espaco) retangulo(c schema.Caixa) (x, y, l, a float64) {
	return e.X + c.X/100*e.L, e.Y + c.Y/100*e.A, c.L / 100 * e.L, c.A / 100 * e.A
}

// circulo projeta um Círculo declarado em porcentagem para pixels absolutos. O
// diâmetro é sempre porcentagem da largura do espaço, nos dois eixos, para que
// o Círculo nunca vire elipse.
func (e espaco) circulo(d schema.Disco) (x, y, l, a float64) {
	tamanho := d.D / 100 * e.L
	return e.X + d.X/100*e.L, e.Y + d.Y/100*e.A, tamanho, tamanho
}

// contexto descreve o arquivo onde um conjunto de nós declarados foi escrito.
// Ele viaja junto dos nós ao longo da cadeia de Instâncias, e é o que permite
// que a mensagem de erro aponte o arquivo certo, que `use` seja resolvido
// relativo a quem referencia, e que o ciclo seja detectado.
type contexto struct {
	// prefixo é o começo do caminho de chaves YAML dos nós deste arquivo,
	// já atravessando a cadeia de Componentes. Vazio no Documento raiz;
	// senão termina em ": ", como em
	// "frames[0].layers[0].elements[2] -> ./card.yaml: ".
	prefixo string
	// origem é o caminho do Componente de onde os nós vieram, relativo ao
	// diretório do Documento. Vazio quando os nós são inline no Documento.
	origem string
	// dir é o diretório contra o qual os caminhos de `use` deste arquivo são
	// resolvidos: sempre o diretório do próprio arquivo.
	dir string
	// pilha são os caminhos absolutos dos Componentes em expansão, do mais
	// externo ao mais interno. É contra ela que o ciclo é detectado.
	pilha []string
	// preenchimentos são os Slots entregues pela Instância que trouxe estes
	// nós, indexados pelo nome do Slot. Nulo no Documento raiz.
	preenchimentos map[string]preenchimento
}

// profundidade é o número de níveis de Componente já atravessados.
func (c contexto) profundidade() int { return len(c.pilha) }

// preenchimento é o conteúdo entregue a um Slot, amarrado ao contexto do
// arquivo que o declarou — não ao do Componente que declara o Slot.
type preenchimento struct {
	// componente é o caminho declarado do Componente, ou "" quando o
	// preenchimento é inline.
	componente string
	// nos são os nós inline, ou nil quando o preenchimento é um Componente.
	nos []schema.No
	// local é o caminho de chaves YAML do preenchimento, dentro do arquivo
	// que o declarou.
	local string
	// ctx é o contexto de quem declarou o preenchimento.
	ctx contexto
}

// achata converte uma lista de nós declarados em Elementos com geometria
// absoluta, acrescentando-os a dest na ordem de pintura. Instâncias, Slots e
// Repetições são materializados aqui: o resultado é uma lista plana onde a fase
// de Elevação não distingue o que foi escrito à mão do que veio de Componente.
//
// prefixo é o caminho já acumulado na árvore resolvida; ctx descreve o arquivo
// onde os nós foram escritos.
func (r *resolucao) achata(nos []schema.No, esp espaco, prefixo string, ctx contexto, dest *[]scene.Elemento) error {
	for i, no := range nos {
		caminho := junta(prefixo, segmento(no, i))
		clones, passo := 1, 0.0
		if no.Repeticao != nil {
			quantos, err := r.clones(no, ctx)
			if err != nil {
				return err
			}
			clones = quantos
			passo = tamanhoNoEixo(no, no.Repeticao.Eixo) + no.Repeticao.Intervalo
		}
		// O orçamento do Frame é debitado aqui, antes de resolver o nó:
		// é o único ponto por onde passa toda tentativa de materializar,
		// inclusive a que não chega a criar Elemento nenhum.
		if err := r.debita(clones, ctx.prefixo+no.Local); err != nil {
			return err
		}
		for c := 0; c < clones; c++ {
			var dx, dy float64
			caminhoDoClone := caminho
			if no.Repeticao != nil {
				// O deslocamento é em unidades do espaço local,
				// antes da conversão para pixels.
				if no.Repeticao.Eixo == "x" {
					dx = float64(c) * passo
				} else {
					dy = float64(c) * passo
				}
				caminhoDoClone = fmt.Sprintf("%s#%d", caminho, c)
			}
			if err := r.materializa(no, esp, dx, dy, caminhoDoClone, ctx, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

// debita cobra do orçamento do Frame a quantidade de clones que um nó vai
// tentar materializar, e recusa o Documento quando o orçamento acaba.
func (r *resolucao) debita(quantos int, local string) error {
	if r.materializados+quantos > LimiteDeElementos {
		return r.erro(local,
			"o Frame %q passou do teto de %d Elementos materializados: reduza a Repetição ou a cadeia de Componentes",
			r.frameNome, LimiteDeElementos)
	}
	r.materializados += quantos
	return nil
}

// clones devolve a quantidade de clones de uma Repetição, recusando o valor
// acima do teto. O teto existe porque Repetições encadeadas por Componentes
// multiplicam: sem ele, poucos bytes de YAML materializam Elementos sem fim.
func (r *resolucao) clones(no schema.No, ctx contexto) (int, error) {
	n := no.Repeticao.N
	if n > LimiteDeClones {
		return 0, r.erro(ctx.prefixo+no.Local+".repeat",
			`campo "n" da Repetição deve estar entre 1 e %d, encontrou %s`,
			LimiteDeClones, strconv.FormatFloat(n, 'g', -1, 64))
	}
	return int(n), nil
}

// materializa resolve um nó, já deslocado por dx e dy no espaço local pela
// Repetição que o clonou.
func (r *resolucao) materializa(no schema.No, esp espaco, dx, dy float64, caminho string, ctx contexto, dest *[]scene.Elemento) error {
	switch no.Tipo {
	case schema.TipoRetangulo:
		c := desloca(*no.Retangulo, dx, dy)
		x, y, l, a := esp.retangulo(c)
		r.acrescenta(dest, no, caminho, ctx, scene.Retangulo, x, y, l, a)
		return nil
	case schema.TipoCirculo:
		d := *no.Circulo
		d.X, d.Y = d.X+dx, d.Y+dy
		x, y, l, a := esp.circulo(d)
		r.acrescenta(dest, no, caminho, ctx, scene.Circulo, x, y, l, a)
		return nil
	case schema.TipoSlot:
		return r.slot(no, esp, desloca(*no.Caixa, dx, dy), caminho, ctx, dest)
	default:
		return r.instancia(no, esp, desloca(*no.Caixa, dx, dy), caminho, ctx, dest)
	}
}

// instancia expande uma Instância: carrega o Componente e achata seus nós no
// espaço local que a caixa da Instância abriu.
func (r *resolucao) instancia(no schema.No, esp espaco, caixa schema.Caixa, caminho string, ctx contexto, dest *[]scene.Elemento) error {
	x, y, l, a := esp.retangulo(caixa)
	return r.expande(no.Componente, ctx.prefixo+no.Local, ctx,
		preenchimentosDaInstancia(no, ctx),
		espaco{X: x, Y: y, L: l, A: a}, caminho, dest)
}

// expande carrega o Componente referenciado e achata seus nós no espaço dado.
// É o caminho comum da Instância e do Slot preenchido por Componente.
func (r *resolucao) expande(referencia, local string, ctx contexto, preenchimentos map[string]preenchimento, esp espaco, caminho string, dest *[]scene.Elemento) error {
	comp, interno, err := r.carrega(referencia, local, ctx)
	if err != nil {
		return err
	}
	interno.preenchimentos = preenchimentos
	return r.achata(comp.Elementos, esp, caminho, interno, dest)
}

// preenchimentosDaInstancia amarra cada Slot preenchido pela Instância ao
// contexto do arquivo que escreveu o preenchimento.
func preenchimentosDaInstancia(no schema.No, ctx contexto) map[string]preenchimento {
	if len(no.OrdemDosPreenchimentos) == 0 {
		return nil
	}
	m := make(map[string]preenchimento, len(no.OrdemDosPreenchimentos))
	for _, nome := range no.OrdemDosPreenchimentos {
		p := no.Preenchimentos[nome]
		m[nome] = preenchimento{
			componente: p.Componente,
			nos:        p.Elementos,
			local:      ctx.prefixo + p.Local,
			ctx:        ctx,
		}
	}
	return m
}

// slot resolve a declaração de um Slot. A região do Slot vira um novo espaço
// local de 0 a 100, preenchido pelo conteúdo entregue por quem instanciou o
// Componente, pelo conteúdo padrão, ou — quando não há nem um nem outro — por
// uma Superfície vazia, com aviso.
func (r *resolucao) slot(no schema.No, esp espaco, caixa schema.Caixa, caminho string, ctx contexto, dest *[]scene.Elemento) error {
	x, y, l, a := esp.retangulo(caixa)
	interno := espaco{X: x, Y: y, L: l, A: a}

	if p, ok := ctx.preenchimentos[no.Slot]; ok {
		if p.componente == "" {
			return r.achata(p.nos, interno, caminho, p.ctx, dest)
		}
		// O preenchimento é resolvido contra o contexto de quem o
		// escreveu, não contra o do Componente que declara o Slot.
		return r.expande(p.componente, p.local, p.ctx, nil, interno, caminho, dest)
	}

	if no.Padrao != nil {
		return r.achata(no.Padrao, interno, caminho, ctx, dest)
	}

	r.aviso(ctx.prefixo+no.Local, fmt.Sprintf(
		"Slot %q%s sem preenchimento e sem conteúdo padrão: renderiza uma Superfície vazia",
		no.Slot, declaradoEm(ctx)))
	r.acrescenta(dest, no, caminho, ctx, scene.Retangulo, x, y, l, a)
	return nil
}

func declaradoEm(ctx contexto) string {
	if ctx.origem == "" {
		return ""
	}
	return " do Componente " + ctx.origem
}

// carrega resolve a referência a um Componente feita em local, devolvendo o
// Componente decodificado e o contexto dos nós dele. A referência é sempre
// relativa ao arquivo que a escreveu, nunca ao diretório de trabalho.
func (r *resolucao) carrega(referencia, local string, ctx contexto) (*schema.Componente, contexto, error) {
	caminho := referencia
	if !filepath.IsAbs(caminho) {
		caminho = filepath.Join(ctx.dir, caminho)
	}
	chave, err := filepath.Abs(caminho)
	if err != nil {
		chave = filepath.Clean(caminho)
	}

	for _, emExpansao := range ctx.pilha {
		if emExpansao == chave {
			return nil, contexto{}, r.erro(local,
				"ciclo de referência entre Componentes: %q já está sendo expandido nesta cadeia de Instâncias", referencia)
		}
	}
	if ctx.profundidade()+1 > LimiteDeProfundidade {
		return nil, contexto{}, r.erro(local,
			"aninhamento de Componentes acima do limite de %d níveis em %q", LimiteDeProfundidade, referencia)
	}
	// O Componente já lido não volta ao disco: numa Repetição de Instância a
	// mesma referência é resolvida a cada clone.
	comp, ok := r.componentes[chave]
	if !ok {
		info, err := os.Stat(caminho)
		switch {
		case err != nil:
			return nil, contexto{}, r.erro(local, "Componente não encontrado: %q", referencia)
		case info.IsDir():
			return nil, contexto{}, r.erro(local, "Componente é um diretório, não um arquivo: %q", referencia)
		}
		lido, err := schema.LeComponente(caminho)
		if err != nil {
			return nil, contexto{}, r.reposiciona(err, local, referencia)
		}
		comp = lido
		r.componentes[chave] = comp
	}

	pilha := make([]string, 0, len(ctx.pilha)+1)
	pilha = append(pilha, ctx.pilha...)
	pilha = append(pilha, chave)
	return comp, contexto{
		prefixo: local + " -> " + referencia + ": ",
		// A Origem é medida sobre o caminho absoluto: `de=` promete um
		// caminho relativo ao Documento, e a referência pode ter sido
		// escrita em qualquer forma.
		origem: r.relativoAoDocumento(chave),
		dir:    filepath.Dir(caminho),
		pilha:  pilha,
	}, nil
}

// relativoAoDocumento devolve o caminho de um Componente relativo ao diretório
// do Documento raiz, que é o que o `inspect` imprime em `de=`.
func (r *resolucao) relativoAoDocumento(caminho string) string {
	rel, err := filepath.Rel(r.dirDoDocumento, caminho)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(caminho))
	}
	return filepath.ToSlash(rel)
}

// reposiciona traz um erro de decodificação de Componente para o Documento
// raiz, encadeando a localização até o ponto exato dentro do Componente.
func (r *resolucao) reposiciona(err error, local, referencia string) error {
	e, ok := err.(*scene.Erro)
	if !ok {
		return &scene.Erro{Arquivo: r.arquivo, Local: local, Msg: err.Error()}
	}
	encadeado := local + " -> " + referencia
	if e.Local != "" {
		encadeado += ": " + e.Local
	}
	return &scene.Erro{Arquivo: r.arquivo, Local: encadeado, Msg: e.Msg}
}

// acrescenta materializa um Elemento com geometria já absoluta e emite os
// avisos que dependem só da geometria. É o único ponto onde um Elemento nasce,
// e onde o caminho é desambiguado. O orçamento do Frame já foi debitado pelo
// clone que trouxe o nó até aqui.
func (r *resolucao) acrescenta(dest *[]scene.Elemento, no schema.No, caminho string, ctx contexto, forma scene.Forma, x, y, l, a float64) {
	local := ctx.prefixo + no.Local
	if x < 0 || y < 0 || x+l > r.frameL || y+a > r.frameA {
		r.aviso(local, "Elemento fora do Frame: será recortado na borda")
	}
	if l <= 0 || a <= 0 {
		r.aviso(local, "Elemento de área zero: não aparecerá no desenho")
	}
	*dest = append(*dest, scene.Elemento{
		Caminho:     r.caminhoUnico(caminho),
		ID:          no.ID,
		Forma:       forma,
		X:           x,
		Y:           y,
		L:           l,
		A:           a,
		Arredondado: no.Arredondado,
		Origem:      ctx.origem,
		Nota:        no.Nota,
	})
}

// caminhoUnico garante que dois Elementos do mesmo Frame nunca compartilhem o
// <caminho>: as duas regras de segmento podem colidir, como um Elemento com
// `id: x` e um Slot chamado `x` no mesmo espaço. A repetição do caminho ganha o
// sufixo ~2, ~3, ... na ordem de pintura, de forma determinística.
func (r *resolucao) caminhoUnico(base string) string {
	if !r.caminhos[base] {
		r.caminhos[base] = true
		return base
	}
	for n := 2; ; n++ {
		c := fmt.Sprintf("%s~%d", base, n)
		if !r.caminhos[c] {
			r.caminhos[c] = true
			return c
		}
	}
}

// desloca move uma caixa no espaço local, como faz a Repetição antes de
// converter para pixels.
func desloca(c schema.Caixa, dx, dy float64) schema.Caixa {
	c.X, c.Y = c.X+dx, c.Y+dy
	return c
}

// tamanhoNoEixo é a extensão do nó no eixo da Repetição, em unidades do espaço
// local: `w`/`h` do Retângulo, `d` do Círculo, ou `box.w`/`box.h` da Instância.
func tamanhoNoEixo(no schema.No, eixo string) float64 {
	switch no.Tipo {
	case schema.TipoCirculo:
		return no.Circulo.D
	case schema.TipoRetangulo:
		if eixo == "x" {
			return no.Retangulo.L
		}
		return no.Retangulo.A
	case schema.TipoInstancia, schema.TipoSlot:
		if eixo == "x" {
			return no.Caixa.L
		}
		return no.Caixa.A
	default:
		return 0
	}
}

// segmento devolve o segmento de caminho de um nó: o nome do Slot, o id
// declarado, ou a posição do nó na sua lista. O nome do Slot vem antes do id
// porque é ele que o contrato usa nos exemplos de caminho; a colisão que isso
// pode criar é desfeita por caminhoUnico.
func segmento(no schema.No, i int) string {
	if no.Tipo == schema.TipoSlot {
		return no.Slot
	}
	if no.ID != "" {
		return no.ID
	}
	return fmt.Sprintf("e%d", i)
}

func junta(prefixo, seg string) string {
	if prefixo == "" {
		return seg
	}
	return prefixo + "/" + seg
}
