// Package update consulta os Lançamentos publicados no GitHub e substitui o
// binário em execução pelo binário da Versão mais nova.
//
// # Por que a comparação de Versão é escrita à mão
//
// A biblioteca padrão não tem comparação semver: `go/version` só entende
// `go1.21.0`, e `golang.org/x/mod/semver` é dependência nova, proibida pelo §0
// do CONTRATO. A ordenação vive em Compara, com o cuidado de ordenar
// prerelease do jeito certo — ausência de prerelease sorteia ACIMA de
// presença. Inverter essa regra faria `v1.0.0-rc.1` parecer mais nova que
// `v1.0.0` e empurraria um downgrade, então é o teste de tabela que sustenta
// esta parte.
package update

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// versao, commit e data são injetados no link:
//
//	go build -ldflags "-X github.com/eduardotorresdev/draftboard/internal/update.versao=v1.4.0 ..."
//
// Num `go install` ou `go build` sem os -X, valem os padrões abaixo, e é
// exatamente isso que "dev" significa: binário sem informação de Versão.
var (
	versao = "dev"
	commit = "desconhecido"
	data   = "desconhecida"
)

// Info descreve o binário em execução.
type Info struct {
	// Versao é a tag do Lançamento, verbatim, ou "dev".
	Versao string
	// Commit é o sha curto, ou "desconhecido".
	Commit string
	// Data é a data do build em RFC 3339, ou "desconhecida".
	Data string
}

// Atual devolve a Info do binário em execução.
func Atual() Info {
	return Info{Versao: versao, Commit: commit, Data: data}
}

// ImprimeVersao escreve a Versão, o commit e a data no writer dado, uma por
// linha.
func ImprimeVersao(w io.Writer) error {
	i := Atual()
	_, err := fmt.Fprintf(w, "draftboard %s\ncommit: %s\ndata: %s\n", i.Versao, i.Commit, i.Data)
	if err != nil {
		return fmt.Errorf("imprimir versão: %w", err)
	}
	return nil
}

// Compara ordena duas Versões: -1 quando a é menor que b, 0 quando são
// equivalentes, +1 quando a é maior. ok é falso quando alguma das duas não é
// uma Versão reconhecível — "dev", "latest", "main" —, e nesse caso ordem não
// tem significado.
//
// A ordenação segue semver 2.0.0: o núcleo numérico primeiro, campo a campo, e
// só então o prerelease, onde a AUSÊNCIA de prerelease é maior que qualquer
// prerelease.
func Compara(a, b string) (ordem int, ok bool) {
	va, ok := interpreta(a)
	if !ok {
		return 0, false
	}
	vb, ok := interpreta(b)
	if !ok {
		return 0, false
	}
	return va.compara(vb), true
}

// numero é a Versão interpretada: núcleo numérico e identificadores de
// prerelease. O build metadata (depois de "+") não entra — semver o ignora na
// ordenação.
type numero struct {
	nucleo [3]int
	pre    []string
}

// interpreta lê `v?N[.N[.N]][-prerelease][+build]`. Campos ausentes no núcleo
// valem 0. Sem regexp: a gramática é pequena o bastante para ser lida à mão, e
// assim o erro de leitura é sempre local.
func interpreta(s string) (numero, bool) {
	var n numero
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return n, false
	}
	// Build metadata é descartado antes de qualquer outra coisa.
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	nucleo := s
	if i := strings.IndexByte(s, '-'); i >= 0 {
		nucleo = s[:i]
		pre := s[i+1:]
		if pre == "" {
			return n, false
		}
		n.pre = strings.Split(pre, ".")
		for _, id := range n.pre {
			if id == "" || !ehIdentificador(id) {
				return n, false
			}
		}
	}
	campos := strings.Split(nucleo, ".")
	if len(campos) > 3 {
		return n, false
	}
	for i, c := range campos {
		if c == "" || !ehNumerico(c) {
			return n, false
		}
		v, err := strconv.Atoi(c)
		if err != nil {
			return n, false
		}
		n.nucleo[i] = v
	}
	return n, true
}

// compara devolve -1, 0 ou +1 comparando n com o.
func (n numero) compara(o numero) int {
	for i := range n.nucleo {
		if d := sinal(n.nucleo[i] - o.nucleo[i]); d != 0 {
			return d
		}
	}
	return comparaPrerelease(n.pre, o.pre)
}

// comparaPrerelease implementa a regra de precedência de prerelease do semver:
// quem não tem prerelease é maior; identificadores são comparados da esquerda
// para a direita; numéricos comparam numericamente e ficam ABAIXO dos
// alfanuméricos; empate em todos os identificadores compartilhados dá vitória
// para a lista mais longa.
func comparaPrerelease(a, b []string) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		x, y := a[i], b[i]
		if x == y {
			continue
		}
		nx, ny := ehNumerico(x), ehNumerico(y)
		switch {
		case nx && ny:
			vx, _ := strconv.Atoi(x)
			vy, _ := strconv.Atoi(y)
			return sinal(vx - vy)
		case nx:
			return -1
		case ny:
			return 1
		default:
			return sinal(strings.Compare(x, y))
		}
	}
	return sinal(len(a) - len(b))
}

func ehNumerico(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func ehIdentificador(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		alfanumerico := c == '-' ||
			(c >= '0' && c <= '9') ||
			(c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z')
		if !alfanumerico {
			return false
		}
	}
	return s != ""
}

func sinal(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}
