package resolve

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/schema"
)

// LimiteDeProfundidade é o número máximo de níveis de Componente que uma cadeia
// de Instâncias pode atravessar. Passar dele é erro: recursão acidental falha
// rápido em vez de consumir memória até o processo morrer.
const LimiteDeProfundidade = 16

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
			clones = no.Repeticao.N
			passo = tamanhoNoEixo(no, no.Repeticao.Eixo) + no.Repeticao.Intervalo
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

// materializa resolve um nó, já deslocado por dx e dy no espaço local pela
// Repetição que o clonou.
func (r *resolucao) materializa(no schema.No, esp espaco, dx, dy float64, caminho string, ctx contexto, dest *[]scene.Elemento) error {
	switch no.Tipo {
	case schema.TipoRetangulo:
		c := desloca(*no.Retangulo, dx, dy)
		x, y, l, a := esp.retangulo(c)
		r.acrescenta(dest, no, caminho, ctx, scene.Retangulo, x, y, l, a)
	case schema.TipoCirculo:
		d := *no.Circulo
		d.X, d.Y = d.X+dx, d.Y+dy
		x, y, l, a := esp.circulo(d)
		r.acrescenta(dest, no, caminho, ctx, scene.Circulo, x, y, l, a)
	case schema.TipoInstancia:
		return r.instancia(no, esp, desloca(*no.Caixa, dx, dy), caminho, ctx, dest)
	default:
		return r.slot(no, esp, desloca(*no.Caixa, dx, dy), caminho, ctx, dest)
	}
	return nil
}

// instancia expande uma Instância: carrega o Componente e achata seus nós no
// espaço local que a caixa da Instância abriu.
func (r *resolucao) instancia(no schema.No, esp espaco, caixa schema.Caixa, caminho string, ctx contexto, dest *[]scene.Elemento) error {
	comp, interno, err := r.carrega(no.Componente, ctx.prefixo+no.Local, ctx)
	if err != nil {
		return err
	}
	interno.preenchimentos = preenchimentosDaInstancia(no, ctx)
	x, y, l, a := esp.retangulo(caixa)
	return r.achata(comp.Elementos, espaco{X: x, Y: y, L: l, A: a}, caminho, interno, dest)
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
		comp, ctxDoComponente, err := r.carrega(p.componente, p.local, p.ctx)
		if err != nil {
			return err
		}
		return r.achata(comp.Elementos, interno, caminho, ctxDoComponente, dest)
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
	if _, err := os.Stat(caminho); err != nil {
		return nil, contexto{}, r.erro(local, "Componente não encontrado: %q", referencia)
	}

	comp, ok := r.componentes[chave]
	if !ok {
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
		origem:  r.relativoAoDocumento(caminho),
		dir:     filepath.Dir(caminho),
		pilha:   pilha,
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
// avisos que dependem só da geometria.
func (r *resolucao) acrescenta(dest *[]scene.Elemento, no schema.No, caminho string, ctx contexto, forma scene.Forma, x, y, l, a float64) {
	local := ctx.prefixo + no.Local
	if x < 0 || y < 0 || x+l > r.frameL || y+a > r.frameA {
		r.aviso(local, "Elemento fora do Frame: será recortado na borda")
	}
	if l <= 0 || a <= 0 {
		r.aviso(local, "Elemento de área zero: não aparecerá no desenho")
	}
	*dest = append(*dest, scene.Elemento{
		Caminho:     caminho,
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
	default:
		if eixo == "x" {
			return no.Caixa.L
		}
		return no.Caixa.A
	}
}

// segmento devolve o segmento de caminho de um nó: o nome do Slot, o id
// declarado, ou a posição do nó na sua lista.
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
