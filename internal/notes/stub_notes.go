// Package notes calcula o plano de anotação de um Frame. Este arquivo é um
// esqueleto de compilação escrito por F1 para fiar a CLI; F4 o substitui pela
// implementação real. Nada aqui desenha.
package notes

import (
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/scene"
)

// Modo é como as Notas são posicionadas na renderização.
type Modo int

const (
	// Margem posiciona as Notas no Chrome ao redor do Frame. É o padrão da
	// CLI.
	Margem Modo = iota
	// Flutuante posiciona as Notas sobre o desenho, perto da âncora.
	Flutuante
	// Desligado remove as Notas inteiras da renderização.
	Desligado
)

// Plano é o layout das Notas de um Frame, calculado sem desenhar.
type Plano struct{}

// Planeja resolve a posição de todas as Notas do Frame. escala é o fator da
// CLI.
func Planeja(f scene.Frame, m Modo, escala float64) *Plano { return &Plano{} }

// Margens devolve o Chrome necessário em px do espaço do Frame. No modo
// Flutuante e Desligado devolve 0,0,0,0.
func (p *Plano) Margens() (t, d, b, e float64) { return 0, 0, 0, 0 }

// Desenha pinta Notas e linhas de chamada sobre um Canvas já criado com essas
// margens. No modo Desligado não faz nada.
func (p *Plano) Desenha(c *render.Canvas) {}
