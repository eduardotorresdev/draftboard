package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/eduardotorresdev/draftboard/internal/notes"
)

// opcoes são as opções de `render` já interpretadas.
type opcoes struct {
	// arquivo é o caminho do Documento.
	arquivo string
	// saida é o diretório onde as imagens são escritas.
	saida string
	// escala multiplica as dimensões declaradas do Frame.
	escala float64
	// notas é o modo de exibição das Notas.
	notas notes.Modo
	// camadas liga o export por Camada, cumulativo.
	camadas bool
}

// interpretaRender lê as opções de `render`. O arquivo do Documento pode vir
// antes ou depois das opções.
func interpretaRender(args []string) (opcoes, error) {
	o := opcoes{saida: ".", escala: 1, notas: notes.Margem}
	var posicionais []string
	args = expandeIguais(args)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !ehOpcao(a) {
			posicionais = append(posicionais, a)
			continue
		}
		switch a {
		case "--out", "-out":
			v, err := valorDe(args, &i, a)
			if err != nil {
				return o, err
			}
			if v == "" {
				return o, fmt.Errorf("opção %q espera um diretório, encontrou valor vazio", a)
			}
			o.saida = v
		case "--scale", "-scale":
			v, err := valorDe(args, &i, a)
			if err != nil {
				return o, err
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f <= 0 {
				return o, fmt.Errorf("opção %q espera um número maior que zero, encontrou %q", a, v)
			}
			o.escala = f
		case "--notes", "-notes":
			v, err := valorDe(args, &i, a)
			if err != nil {
				return o, err
			}
			switch v {
			case "margin":
				o.notas = notes.Margem
			case "float":
				o.notas = notes.Flutuante
			case "off":
				o.notas = notes.Desligado
			default:
				return o, fmt.Errorf("opção %q espera margin, float ou off, encontrou %q", a, v)
			}
		case "--layers", "-layers":
			o.camadas = true
		default:
			return o, fmt.Errorf("opção desconhecida %q", a)
		}
	}
	arquivo, err := unicoArquivo(posicionais)
	if err != nil {
		return o, err
	}
	o.arquivo = arquivo
	return o, nil
}

// opcoesBoard são as opções do verbo `board`. A Prancheta não tem escala nem
// modo de Nota: ela é vetorial, o zoom é do navegador, e a Nota aparece na
// inspeção em vez de num plano fixo do desenho.
type opcoesBoard struct {
	arquivo string
	// saida é o diretório onde a Prancheta é escrita.
	saida string
}

func interpretaBoard(args []string) (opcoesBoard, error) {
	o := opcoesBoard{saida: "."}
	var posicionais []string
	args = expandeIguais(args)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !ehOpcao(a) {
			posicionais = append(posicionais, a)
			continue
		}
		switch a {
		case "--out", "-out":
			v, err := valorDe(args, &i, a)
			if err != nil {
				return o, err
			}
			if v == "" {
				return o, fmt.Errorf("opção %q espera um diretório, encontrou valor vazio", a)
			}
			o.saida = v
		default:
			return o, fmt.Errorf("opção desconhecida %q", a)
		}
	}
	arquivo, err := unicoArquivo(posicionais)
	if err != nil {
		return o, err
	}
	o.arquivo = arquivo
	return o, nil
}

// interpretaArquivo lê os argumentos de um verbo que só recebe o caminho do
// Documento.
func interpretaArquivo(args []string) (string, error) {
	var posicionais []string
	for _, a := range expandeIguais(args) {
		if ehOpcao(a) {
			return "", fmt.Errorf("opção desconhecida %q", a)
		}
		posicionais = append(posicionais, a)
	}
	return unicoArquivo(posicionais)
}

func unicoArquivo(posicionais []string) (string, error) {
	switch len(posicionais) {
	case 0:
		return "", errors.New("informe o caminho do Documento")
	case 1:
		return posicionais[0], nil
	default:
		return "", fmt.Errorf("argumento em excesso: %q", posicionais[1])
	}
}

func ehOpcao(a string) bool {
	return len(a) > 1 && strings.HasPrefix(a, "-")
}

// expandeIguais quebra `--opcao=valor` em dois argumentos, para que a leitura
// aceite as duas formas.
func expandeIguais(args []string) []string {
	saida := make([]string, 0, len(args))
	for _, a := range args {
		if nome, valor, achou := strings.Cut(a, "="); achou && ehOpcao(a) {
			saida = append(saida, nome, valor)
			continue
		}
		saida = append(saida, a)
	}
	return saida
}

// valorDe consome o argumento seguinte como valor da opção.
func valorDe(args []string, i *int, opcao string) (string, error) {
	if *i+1 >= len(args) {
		return "", fmt.Errorf("opção %q exige um valor", opcao)
	}
	*i++
	return args[*i], nil
}

// slug reduz um componente de nome de arquivo: minúsculas, cada sequência de
// caracteres fora de [a-z0-9] vira um "-" único, "-" das pontas removidos.
func slug(s string) string {
	var b strings.Builder
	pendente := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendente && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendente = false
			b.WriteRune(r)
			continue
		}
		pendente = true
	}
	return b.String()
}

// resposta é a resposta pré-definida a uma pergunta interativa: `--yes` e
// `--no` existem para quem roda a CLI num script, onde não há ninguém para
// responder.
type resposta int

const (
	// perguntar é o padrão: pergunta quando a entrada é um terminal.
	perguntar resposta = iota
	sempreSim
	sempreNao
)

// opcoesSkill são as opções de `skill` já interpretadas.
type opcoesSkill struct {
	// instalar grava a skill sem conferir o que já está instalado.
	instalar bool
	// sincronizar grava a skill só quando o conteúdo mudou, perguntando antes.
	sincronizar bool
	// destino é o diretório de skills; vazio significa o padrão.
	destino string
	// resposta é a resposta pré-definida à pergunta de `--sync`.
	resposta resposta
}

// interpretaSkill lê as opções de `skill`. `--install` e `--sync` aceitam um
// diretório opcional, que pode vir como `--flag=DIR` ou como argumento
// seguinte.
func interpretaSkill(args []string) (opcoesSkill, error) {
	var o opcoesSkill
	sim, nao := false, false
	args = expandeIguais(args)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--install", "-install":
			o.instalar = true
			o.destino = valorOpcional(args, &i, o.destino)
		case "--sync", "-sync":
			o.sincronizar = true
			o.destino = valorOpcional(args, &i, o.destino)
		case "--yes", "-yes":
			sim = true
		case "--no", "-no":
			nao = true
		default:
			return o, fmt.Errorf("argumento inesperado %q", a)
		}
	}
	if o.instalar && o.sincronizar {
		return o, errors.New("não combine --install com --sync: --install sempre grava, --sync só quando a skill mudou")
	}
	r, err := respostaDe(sim, nao)
	if err != nil {
		return o, err
	}
	o.resposta = r
	return o, nil
}

// opcoesUpdate são as opções de `update` já interpretadas.
type opcoesUpdate struct {
	// conferir só reporta se há Versão nova, sem escrever nada.
	conferir bool
	// resposta é a resposta pré-definida à pergunta da skill.
	resposta resposta
}

// interpretaUpdate lê as opções de `update`.
func interpretaUpdate(args []string) (opcoesUpdate, error) {
	var o opcoesUpdate
	sim, nao := false, false
	for _, a := range expandeIguais(args) {
		switch a {
		case "--check", "-check":
			o.conferir = true
		case "--yes", "-yes":
			sim = true
		case "--no", "-no":
			nao = true
		default:
			return o, fmt.Errorf("argumento inesperado %q", a)
		}
	}
	r, err := respostaDe(sim, nao)
	if err != nil {
		return o, err
	}
	o.resposta = r
	return o, nil
}

func respostaDe(sim, nao bool) (resposta, error) {
	switch {
	case sim && nao:
		return perguntar, errors.New("não combine --yes com --no")
	case sim:
		return sempreSim, nil
	case nao:
		return sempreNao, nil
	default:
		return perguntar, nil
	}
}

// valorOpcional consome o argumento seguinte como valor da opção, mas só
// quando ele existe e não é outra opção. Sem valor, devolve o atual.
func valorOpcional(args []string, i *int, atual string) string {
	if *i+1 < len(args) && !ehOpcao(args[*i+1]) {
		*i++
		return args[*i]
	}
	return atual
}
