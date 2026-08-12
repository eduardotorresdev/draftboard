package resolve

import (
	"fmt"

	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/schema"
)

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

// achata converte uma lista de nós declarados em Elementos com geometria
// absoluta, acrescentando-os a dest na ordem de pintura.
//
// prefixo é o caminho já acumulado na árvore; origem é o caminho relativo do
// Componente de onde os nós vieram, vazio quando são inline no Documento.
func (r *resolucao) achata(nos []schema.No, esp espaco, prefixo, origem string, dest *[]scene.Elemento) error {
	for i, no := range nos {
		caminho := junta(prefixo, segmento(no, i))
		if no.Repeticao != nil {
			return naoImplementado(r.arquivo, no.Local, "Repetição ainda não implementada")
		}
		switch no.Tipo {
		case schema.TipoRetangulo:
			x, y, l, a := esp.retangulo(*no.Retangulo)
			r.acrescenta(dest, no, caminho, origem, scene.Retangulo, x, y, l, a)
		case schema.TipoCirculo:
			x, y, l, a := esp.circulo(*no.Circulo)
			r.acrescenta(dest, no, caminho, origem, scene.Circulo, x, y, l, a)
		case schema.TipoInstancia:
			return naoImplementado(r.arquivo, no.Local, "Instância ainda não implementada")
		default:
			return naoImplementado(r.arquivo, no.Local, "Slot ainda não implementado")
		}
	}
	return nil
}

// acrescenta materializa um Elemento com geometria já absoluta e emite os
// avisos que dependem só da geometria.
func (r *resolucao) acrescenta(dest *[]scene.Elemento, no schema.No, caminho, origem string, forma scene.Forma, x, y, l, a float64) {
	if x < 0 || y < 0 || x+l > r.frameL || y+a > r.frameA {
		r.aviso(no.Local, "Elemento fora do Frame: será recortado na borda")
	}
	if l <= 0 || a <= 0 {
		r.aviso(no.Local, "Elemento de área zero: não aparecerá no desenho")
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
		Origem:      origem,
		Nota:        no.Nota,
	})
}

// segmento devolve o segmento de caminho de um nó: o id declarado, ou a
// posição do nó na sua lista.
func segmento(no schema.No, i int) string {
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

// naoImplementado marca um nó que o schema já aceita mas que o achatamento
// ainda não materializa.
func naoImplementado(arquivo, local, msg string) error {
	return &scene.Erro{Arquivo: arquivo, Local: local, Msg: msg}
}
