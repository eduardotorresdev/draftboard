// Package diag mede o Documento já resolvido e diz o que não cabe.
//
// O diagnóstico nasce aqui, e não em internal/resolve, porque medir um Rótulo
// exige a fonte: a resolução é o único lugar do sistema que calcula geometria
// sem depender do freetype, e inverter isso custaria o pacote inteiro.
//
// A categoria de cada achado é decidida pela CORRIGIBILIDADE, e não pela
// gravidade: o que a máquina conserta sozinha é Aviso, o que exige juízo do
// autor é Erro. Quem responde se a máquina consegue consertar é um predicado
// que chega de fora — assim este pacote não conhece internal/fix e pode ser
// medido com um predicado de mentira.
package diag

import (
	"fmt"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/eduardotorresdev/draftboard/internal/notes"
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/resolve"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// As razões que este pacote conhece por conta própria. As outras chegam pelo
// predicado, que é quem lê o YAML cru.
const (
	razaoComponente = "o Retângulo vem de um Componente, e alargá-lo lá muda todas as Instâncias"
	razaoEspacoZero = "o espaço do Retângulo tem largura zero"
)

// Alargamento é o conserto que faz um Rótulo caber: o Local do nó que o
// declarou e o `w` que ele passa a precisar.
type Alargamento struct {
	Local string
	W     float64
}

// Confere mede o Documento já resolvido e devolve o que não cabe, separado por
// corrigibilidade. alargavel responde se a máquina consegue consertar o nó
// sozinha, e por que não quando não consegue; nil significa que nada é
// alargável e tudo que não couber é Erro.
func Confere(arquivo string, doc *scene.Documento, alargavel func(local string) (bool, string)) ([]scene.Aviso, []*scene.Erro) {
	var avisos []scene.Aviso
	var erros []*scene.Erro
	for _, a := range confere(doc, alargavel) {
		if a.razao == "" {
			avisos = append(avisos, scene.Aviso{Arquivo: arquivo, Local: a.local, Msg: a.msg()})
			continue
		}
		erros = append(erros, &scene.Erro{Arquivo: arquivo, Local: a.local, Msg: a.msg()})
	}
	return avisos, erros
}

// Alargamentos devolve os consertos correspondentes aos Avisos que Confere
// emitiria: um por nó do YAML, na ordem em que foram medidos.
func Alargamentos(doc *scene.Documento, alargavel func(local string) (bool, string)) []Alargamento {
	var consertos []Alargamento
	for _, a := range confere(doc, alargavel) {
		if a.razao != "" || a.w <= 0 {
			continue
		}
		consertos = append(consertos, Alargamento{Local: a.local, W: a.w})
	}
	return consertos
}

// achado é um problema medido, antes de virar Aviso ou Erro. A razão vazia é o
// que faz dele um Aviso.
type achado struct {
	local string
	razao string
	// rotulo, precisa e tem descrevem o Rótulo que não coube; w é a largura
	// sugerida em porcentagem do espaço do nó.
	rotulo         string
	precisa, tem   float64
	w              float64
	runas, limite  int
	ehNotaComprida bool
}

// msg escreve a mensagem que o autor lê. As três formas são literais do
// contrato: nenhuma diz "caixa" nem "texto", que são proibições do vocabulário.
func (a achado) msg() string {
	switch {
	case a.ehNotaComprida:
		return fmt.Sprintf("a Nota tem %d runas, acima do limite de %d; encurte-a", a.runas, a.limite)
	case a.razao == "":
		return fmt.Sprintf("o Rótulo %q não cabe no Retângulo: precisa de %.0f px e tem %.0f; use w: %s",
			a.rotulo, a.precisa, a.tem, strconv.FormatFloat(a.w, 'g', -1, 64))
	default:
		return fmt.Sprintf("o Rótulo %q não cabe no Retângulo: precisa de %.0f px e tem %.0f; %s",
			a.rotulo, a.precisa, a.tem, a.razao)
	}
}

// confere percorre o Documento uma vez e devolve todos os achados, na ordem de
// pintura.
//
// A régua é render.LarguraDoDiagnostico, que mede sem hinting e sem escala: o
// diagnóstico é do Documento, e não da invocação — `--scale` não muda o que
// cabe, e o avanço ideal da fonte é justamente o que não depende dela.
func confere(doc *scene.Documento, alargavel func(local string) (bool, string)) []achado {
	if alargavel == nil {
		alargavel = semPredicado
	}
	var achados []achado
	for _, f := range doc.Frames {
		donos := donosDoFrame(f)
		for _, c := range f.Camadas {
			for _, e := range c.Elementos {
				if n := utf8.RuneCountInString(e.Nota); n > notes.LimiteDaNota {
					achados = append(achados, achado{
						local: e.Local, razao: razaoSemConserto,
						ehNotaComprida: true, runas: n, limite: notes.LimiteDaNota,
					})
				}
				// Os Elementos medidos são os Rótulos de Retângulo, e só eles:
				// o Rótulo de Controle tem a caixa escolhida pelo catálogo, e o
				// autor não tem no YAML um `w` de Retângulo para alargar.
				if e.Forma != scene.Texto || e.Controle != "" {
					continue
				}
				if a, achou := mede(e, donos, alargavel); achou {
					achados = append(achados, a)
				}
			}
		}
	}
	return deduplica(achados)
}

// deduplica reduz os achados a um por nó do YAML.
//
// A deduplicação é por Local, e não por Caminho, porque é do nó que o autor
// escreveu que o diagnóstico fala: uma Repetição de dez clones repetiria a
// mesma frase dez vezes, e o `--fix` tentaria dez cirurgias no mesmo `w` — a
// segunda medindo os bytes que a primeira já moveu. Empatando, vence quem
// precisa de mais largura: a menor deixaria o Rótulo cortado do mesmo jeito.
func deduplica(achados []achado) []achado {
	unicos := make([]achado, 0, len(achados))
	onde := map[chave]int{}
	for _, a := range achados {
		c := chave{local: a.local, nota: a.ehNotaComprida}
		i, visto := onde[c]
		if !visto {
			onde[c] = len(unicos)
			unicos = append(unicos, a)
			continue
		}
		if a.precisa > unicos[i].precisa {
			unicos[i] = a
		}
	}
	return unicos
}

// chave separa os dois diagnósticos que um mesmo nó pode receber: um Rótulo que
// não cabe e uma Nota comprida são problemas diferentes do mesmo Local.
type chave struct {
	local string
	nota  bool
}

// razaoSemConserto marca o achado que não é sobre largura nenhuma. Não chega a
// aparecer na mensagem: ela só serve para classificá-lo como Erro.
const razaoSemConserto = "-"

// mede diz se o Rótulo cabe na área que a resolução lhe deu e, quando não cabe,
// monta o achado já classificado.
func mede(e scene.Elemento, donos map[string]scene.Elemento, alargavel func(string) (bool, string)) (achado, bool) {
	largura := render.LarguraDoDiagnostico(e.Rotulo, render.TamanhoDoRotulo(e.A))
	if largura <= e.L {
		return achado{}, false
	}
	dono, temDono := donos[donoDoRotulo(e.Caminho)]
	if !temDono {
		// Inalcançável: o Rótulo é sempre emitido com o caminho do Retângulo
		// que o carrega mais um segmento fixo. Fica como silêncio, e não como
		// pânico, porque um par que não se acha é problema nosso, não do autor.
		return achado{}, false
	}

	a := achado{
		local:   e.Local,
		rotulo:  e.Rotulo,
		precisa: largura,
		tem:     e.L,
	}
	switch {
	case dono.Origem != "":
		a.razao = razaoComponente
	case !finito(dono.Espaco.L) || dono.Espaco.L <= 0:
		a.razao = razaoEspacoZero
	default:
		if ok, razao := alargavel(e.Local); !ok {
			a.razao = razao
		}
	}
	if a.razao != "" {
		return a, true
	}
	// A largura necessária é a do RETÂNGULO, não a da área do Rótulo: a
	// resolução já descontou o respiro de cada ponta, e devolvê-lo aqui é o que
	// impede o `w` sugerido de sair curto e não consertar nada.
	//
	// O respiro volta POR VALOR, e não pela diferença `dono.L - e.L`: a área do
	// Rótulo satura em zero, então num Retângulo mais estreito que o dobro do
	// respiro a diferença vale a largura inteira do Retângulo, o `w` sugerido
	// sai curto, e a mesma chamada que conserta já pede outro conserto.
	necessarioNoRetangulo := largura + 2*resolve.RespiroDoRotulo
	// Arredondado para cima: um `w` que arredonda para baixo continua cortando,
	// e um Aviso que sugere um conserto que não conserta é pior que nenhum.
	a.w = math.Ceil(100 * necessarioNoRetangulo / dono.Espaco.L)
	return a, true
}

// donoDoRotulo devolve o Caminho do Retângulo que carrega o Rótulo de caminho
// dado. O sufixo vem de resolve, para que não existam duas grafias dele.
func donoDoRotulo(caminho string) string {
	sufixo := "/" + resolve.SegmentoDoRotulo
	if len(caminho) <= len(sufixo) || caminho[len(caminho)-len(sufixo):] != sufixo {
		return ""
	}
	return caminho[:len(caminho)-len(sufixo)]
}

// donosDoFrame indexa os Elementos do Frame pelo Caminho, que é o que
// identifica Elemento — o Local não serve, porque um nó materializa vários.
func donosDoFrame(f scene.Frame) map[string]scene.Elemento {
	m := map[string]scene.Elemento{}
	for _, c := range f.Camadas {
		for _, e := range c.Elementos {
			m[e.Caminho] = e
		}
	}
	return m
}

// semPredicado é o predicado de quem não passou nenhum: sem alguém que leia o
// YAML cru, não há `w` que a máquina saiba trocar, e tudo que não couber é Erro.
func semPredicado(string) (bool, string) { return false, `o Retângulo não declara "w"` }

func finito(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
