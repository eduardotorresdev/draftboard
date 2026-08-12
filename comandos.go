package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eduardotorresdev/draftboard/internal/inspect"
	"github.com/eduardotorresdev/draftboard/internal/notes"
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/resolve"
	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/skill"
)

// comandoRender resolve o Documento e escreve uma imagem por Frame, ou uma por
// Camada quando o export por Camada está ligado. Só os caminhos escritos vão
// para o stdout, na ordem de geração.
func comandoRender(args []string, stdout, stderr io.Writer) int {
	o, err := interpretaRender(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	doc, avisos, err := resolve.Arquivo(o.arquivo)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	if err := os.MkdirAll(o.saida, 0o755); err != nil {
		// Erro de linha de comando não tem arquivo nem localização: não usa
		// o formato de `scene.Erro`.
		fmt.Fprintf(stderr, "erro: não foi possível criar o diretório de saída %q\n", o.saida)
		return 1
	}
	for i, f := range doc.Frames {
		caminhos, err := escreveFrame(o, doc.Nome, i, f)
		for _, c := range caminhos {
			fmt.Fprintln(stdout, c)
		}
		if err != nil {
			return imprimeErro(stderr, err)
		}
	}
	return 0
}

// escreveFrame gera as imagens de um Frame e devolve os caminhos escritos.
func escreveFrame(o opcoes, documento string, indice int, f scene.Frame) ([]string, error) {
	var caminhos []string
	// As Notas não aparecem no export por Camada, então ele também não
	// reserva Chrome ao redor do Frame.
	if o.camadas {
		if err := cabeNaTela(o, indice, f, 0, 0, 0, 0); err != nil {
			return nil, err
		}
		for i, c := range f.Camadas {
			tela := render.DesenhaFrame(f, o.escala, 0, 0, 0, 0, i)
			nome := fmt.Sprintf("%s-%s-%02d-%s.webp", slug(documento), slug(f.Nome), i+1, slug(c.Nome))
			caminho := filepath.Join(o.saida, nome)
			if err := escreveWebP(caminho, tela); err != nil {
				return caminhos, err
			}
			caminhos = append(caminhos, caminho)
		}
		return caminhos, nil
	}
	// O Chrome entra na conta do teto de área, então o teste vem depois de
	// planejar as Notas, com as margens que serão realmente usadas.
	plano := notes.Planeja(f, o.notas, o.escala)
	t, d, b, e := plano.Margens()
	if err := cabeNaTela(o, indice, f, t, d, b, e); err != nil {
		return nil, err
	}
	tela := render.DesenhaFrame(f, o.escala, t, d, b, e, -1)
	plano.Desenha(tela)
	nome := fmt.Sprintf("%s-%s.webp", slug(documento), slug(f.Nome))
	caminho := filepath.Join(o.saida, nome)
	if err := escreveWebP(caminho, tela); err != nil {
		return caminhos, err
	}
	return append(caminhos, caminho), nil
}

// cabeNaTela recusa, antes de qualquer alocação, o Frame cuja tela de saída
// passaria de render.LimiteDeArea. A área é a do Frame mais o Chrome, com o
// fator de escala aplicado nos dois eixos.
func cabeNaTela(o opcoes, indice int, f scene.Frame, margemT, margemD, margemB, margemE float64) error {
	largura := margemE + float64(f.L) + margemD
	altura := margemT + float64(f.A) + margemB
	area := largura * altura * o.escala * o.escala
	if area <= render.LimiteDeArea {
		return nil
	}
	return &scene.Erro{
		Arquivo: o.arquivo,
		Local:   fmt.Sprintf("frames[%d]", indice),
		Msg: fmt.Sprintf(
			"a tela de saída teria %.0f px com --scale %s, acima do limite de %d px; reduza a escala ou as dimensões do Frame",
			area, strconv.FormatFloat(o.escala, 'f', -1, 64), render.LimiteDeArea),
	}
}

func escreveWebP(caminho string, tela *render.Canvas) error {
	arquivo, err := os.Create(caminho)
	if err != nil {
		return &scene.Erro{Arquivo: caminho, Msg: "não foi possível criar a imagem"}
	}
	if err := tela.CodificaWebP(arquivo); err != nil {
		arquivo.Close()
		return &scene.Erro{Arquivo: caminho, Msg: "não foi possível codificar a imagem: " + err.Error()}
	}
	if err := arquivo.Close(); err != nil {
		return &scene.Erro{Arquivo: caminho, Msg: "não foi possível fechar a imagem"}
	}
	return nil
}

// comandoInspect imprime a árvore resolvida no stdout. Nada é escrito em disco.
func comandoInspect(args []string, stdout, stderr io.Writer) int {
	caminho, err := interpretaArquivo(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	doc, avisos, err := resolve.Arquivo(caminho)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	if err := inspect.Arvore(stdout, doc); err != nil {
		return imprimeErro(stderr, err)
	}
	return 0
}

// comandoValidate resolve o Documento sem produzir saída: nada no stdout em
// caso de sucesso, avisos no stderr, código 1 só quando há erro.
func comandoValidate(args []string, stderr io.Writer) int {
	caminho, err := interpretaArquivo(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	_, avisos, err := resolve.Arquivo(caminho)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	return 0
}

// comandoSkill imprime a skill embutida, ou a instala e imprime o caminho
// escrito.
func comandoSkill(args []string, stdout, stderr io.Writer) int {
	instalar := false
	destino := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		nome, valor, temValor := strings.Cut(a, "=")
		switch nome {
		case "--install", "-install":
			instalar = true
			switch {
			case temValor:
				destino = valor
			case i+1 < len(args) && !ehOpcao(args[i+1]):
				i++
				destino = args[i]
			}
		default:
			return usoInvalido(stderr, fmt.Errorf("argumento inesperado %q", a))
		}
	}
	if !instalar {
		if err := skill.Imprime(stdout); err != nil {
			return imprimeErro(stderr, err)
		}
		return 0
	}
	caminho, err := skill.Instala(destino)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	fmt.Fprintln(stdout, caminho)
	return 0
}
