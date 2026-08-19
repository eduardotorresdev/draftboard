package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eduardotorresdev/draftboard/internal/board"
	"github.com/eduardotorresdev/draftboard/internal/diag"
	"github.com/eduardotorresdev/draftboard/internal/fix"
	"github.com/eduardotorresdev/draftboard/internal/inspect"
	"github.com/eduardotorresdev/draftboard/internal/notes"
	"github.com/eduardotorresdev/draftboard/internal/render"
	"github.com/eduardotorresdev/draftboard/internal/resolve"
	"github.com/eduardotorresdev/draftboard/internal/scene"
	"github.com/eduardotorresdev/draftboard/internal/skill"
	"github.com/eduardotorresdev/draftboard/internal/update"
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
	codigo := diagnostica(stderr, o.arquivo, doc)
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
	return codigo
}

// escreveFrame gera as imagens de um Frame e devolve os caminhos escritos.
func escreveFrame(o opcoes, documento string, indice int, f scene.Frame) ([]string, error) {
	var caminhos []string
	// As Notas não aparecem no export por Camada, então ele também não
	// planeja anotação nenhuma.
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
	// O teto de área vem antes de planejar a anotação. As margens do plano
	// entravam na conta e obrigavam a inverter a ordem, mas o balão é preso
	// dentro do Frame desde que o modo margem foi aposentado: Margens() é
	// sempre 0, e a tela é a do Frame com ou sem Notas. Planejar primeiro só
	// gastava — com `--scale 9000`, 282 MB de residente para chegar ao mesmo
	// erro de teto que `cabeNaTela` dá de graça.
	if err := cabeNaTela(o, indice, f, 0, 0, 0, 0); err != nil {
		return nil, err
	}
	// Sem `--notes` não há plano: o *Plano nulo atravessa os dois métodos
	// abaixo sem desenhar nem pedir margem.
	var plano *notes.Plano
	if o.notas {
		plano = notes.Planeja(f, o.escala)
	}
	t, d, b, e := plano.Margens()
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
// passaria de render.LimiteDeArea. A área é a do Frame mais as margens, com o
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

// comandoBoard resolve o Documento e escreve a Prancheta: um arquivo HTML só,
// com todos os Frames e as Ligações entre eles. Só o caminho escrito vai para o
// stdout, como no render.
func comandoBoard(args []string, stdout, stderr io.Writer) int {
	o, err := interpretaBoard(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	doc, avisos, err := resolve.Arquivo(o.arquivo)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	codigo := diagnostica(stderr, o.arquivo, doc)
	if err := cabeNoNavegador(o.arquivo, doc); err != nil {
		return imprimeErro(stderr, err)
	}
	if err := os.MkdirAll(o.saida, 0o755); err != nil {
		fmt.Fprintf(stderr, "erro: não foi possível criar o diretório de saída %q\n", o.saida)
		return 1
	}
	caminho := filepath.Join(o.saida, slug(doc.Nome)+".html")
	if err := escrevePrancheta(caminho, doc); err != nil {
		return imprimeErro(stderr, err)
	}
	fmt.Fprintln(stdout, caminho)
	return codigo
}

// cabeNoNavegador recusa, antes de montar qualquer HTML, o Documento cuja
// Prancheta teria mais nós do que um navegador aguenta. Cada Elemento vira um
// nó do DOM, e o teto é da Prancheta inteira — não de um Frame.
func cabeNoNavegador(arquivo string, doc *scene.Documento) error {
	n := board.Elementos(doc)
	if n <= board.LimiteDeElementos {
		return nil
	}
	return &scene.Erro{
		Arquivo: arquivo,
		Msg: fmt.Sprintf(
			"a Prancheta teria %d Elementos, acima do limite de %d; separe o fluxo em mais de um Documento",
			n, board.LimiteDeElementos),
	}
}

func escrevePrancheta(caminho string, doc *scene.Documento) error {
	arquivo, err := os.Create(caminho)
	if err != nil {
		return &scene.Erro{Arquivo: caminho, Msg: "não foi possível criar a Prancheta"}
	}
	if err := board.Escreve(arquivo, doc); err != nil {
		arquivo.Close()
		return &scene.Erro{Arquivo: caminho, Msg: "não foi possível escrever a Prancheta: " + err.Error()}
	}
	if err := arquivo.Close(); err != nil {
		return &scene.Erro{Arquivo: caminho, Msg: "não foi possível fechar a Prancheta"}
	}
	return nil
}

// comandoInspect imprime a árvore resolvida no stdout. Nada é escrito em disco,
// a menos que `--fix` peça o conserto.
func comandoInspect(args []string, stdout, stderr io.Writer) int {
	o, err := interpretaInspect(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	if o.corrige {
		return consertaEInspeciona(o.arquivo, stdout, stderr)
	}
	doc, avisos, err := resolve.Arquivo(o.arquivo)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	codigo := diagnostica(stderr, o.arquivo, doc)
	if err := inspect.Arvore(stdout, doc); err != nil {
		return imprimeErro(stderr, err)
	}
	return codigo
}

// consertaEInspeciona alarga o `w` de cada Retângulo cujo Rótulo não cabe e
// imprime a árvore JÁ CORRIGIDA. As duas coisas numa chamada só porque imprimir
// a árvore velha diria ao agente que o conserto não aconteceu.
//
// A ordem no stderr é: as linhas de troca primeiro, e depois tudo o que a
// segunda resolução tem a dizer — os Avisos dela e o diagnóstico. Alargar um
// Retângulo encostado na borda direita produz um Aviso novo de "fora do Frame",
// e escondê-lo faria o `--fix` mostrar menos que o `inspect` puro.
func consertaEInspeciona(arquivo string, stdout, stderr io.Writer) int {
	doc, avisos, err := resolve.Arquivo(arquivo)
	if err != nil {
		imprimeAvisos(stderr, avisos)
		return imprimeErro(stderr, err)
	}
	arq, erroDeLeitura := fix.Abre(arquivo)
	var consertos []diag.Alargamento
	if erroDeLeitura == nil {
		consertos = diag.Alargamentos(doc, arq.Alargavel)
	}
	if len(consertos) == 0 {
		// Sem nada a consertar, `--fix` não escreve no arquivo e se comporta
		// como o `inspect` puro: um Documento cujo único diagnóstico é Erro
		// não pode perder nem o mtime por ter sido inspecionado.
		imprimeAvisos(stderr, avisos)
		codigo := diagnostica(stderr, arquivo, doc)
		if err := inspect.Arvore(stdout, doc); err != nil {
			return imprimeErro(stderr, err)
		}
		return codigo
	}
	for _, c := range consertos {
		if err := arq.Alarga(c.Local, c.W); err != nil {
			return imprimeErro(stderr, err)
		}
	}
	trocas, err := arq.Grava()
	if err != nil {
		return imprimeErro(stderr, err)
	}
	for _, t := range trocas {
		fmt.Fprintf(stderr, "%s: w %s → %s\n", t.Local, formataLargura(t.De), formataLargura(t.Para))
	}

	doc, avisos, err = resolve.Arquivo(arquivo)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	codigo := diagnostica(stderr, arquivo, doc)
	if err := inspect.Arvore(stdout, doc); err != nil {
		return imprimeErro(stderr, err)
	}
	return codigo
}

// diagnostica mede o Documento já resolvido e imprime o que não cabe. Devolve o
// código de saída: 1 quando sobrou Erro.
//
// O Erro daqui NÃO aborta o comando, e essa é a diferença de natureza que o
// vocabulário já registra: os Erros antigos impedem de saber o que desenhar,
// este descreve um desenho que já existe e está errado. O comando escreve o que
// tinha para escrever e só o código de saída muda.
func diagnostica(stderr io.Writer, arquivo string, doc *scene.Documento) int {
	avisos, erros := diag.Confere(arquivo, doc, predicadoDeAlargamento(arquivo))
	imprimeAvisos(stderr, avisos)
	for _, e := range erros {
		fmt.Fprintln(stderr, "erro: "+e.Error())
	}
	if len(erros) > 0 {
		return 1
	}
	return 0
}

// predicadoDeAlargamento abre o YAML cru só para responder se a máquina
// consegue alargar cada nó sozinha. Arquivo que não abre devolve nil: o
// diagnóstico continua, e tudo que não couber vira Erro — dizer "use w: 47"
// sobre um arquivo que não se consegue ler seria prometer um conserto que
// ninguém vai aplicar.
func predicadoDeAlargamento(arquivo string) func(string) (bool, string) {
	a, err := fix.Abre(arquivo)
	if err != nil {
		return nil
	}
	return a.Alargavel
}

// formataLargura escreve a largura como o autor a reconhece no arquivo: sem
// notação científica inventada e sem casas decimais que ele não digitou.
func formataLargura(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

// comandoValidate resolve o Documento sem produzir saída: nada no stdout em
// caso de sucesso, avisos no stderr, código 1 só quando há erro.
func comandoValidate(args []string, stderr io.Writer) int {
	caminho, err := interpretaArquivo(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	doc, avisos, err := resolve.Arquivo(caminho)
	imprimeAvisos(stderr, avisos)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	return diagnostica(stderr, caminho, doc)
}

// comandoSkill imprime a skill embutida, ou a grava e imprime o caminho
// escrito.
//
// Sem opção, imprime no stdout. Com `--install`, grava sempre. Com `--sync`,
// grava só quando a skill embutida difere da instalada, e pergunta antes — é o
// verbo que `update` chama no binário novo depois de trocar o binário, porque
// só o binário novo carrega a skill nova.
func comandoSkill(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	o, err := interpretaSkill(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	switch {
	case o.sincronizar:
		return sincronizaSkill(o, stdin, stdout, stderr)
	case o.instalar:
		caminho, err := skill.Instala(o.destino)
		if err != nil {
			return imprimeErro(stderr, err)
		}
		fmt.Fprintln(stdout, caminho)
		return 0
	default:
		if err := skill.Imprime(stdout); err != nil {
			return imprimeErro(stderr, err)
		}
		return 0
	}
}

// sincronizaSkill regrava a skill instalada só quando ela difere da embutida.
//
// Já sincronizada não imprime nada e sai 0, para que um `update` de rotina
// caiba em duas linhas. Quando a entrada não é um terminal, não pergunta e não
// grava: escrever em ~/.claude numa invocação canalizada violaria o padrão de
// só reportar, e errar quebraria o caminho dirigido por agente, que é o
// consumidor primário desta CLI.
func sincronizaSkill(o opcoesSkill, stdin io.Reader, stdout, stderr io.Writer) int {
	igual, caminho, err := skill.EstaSincronizada(o.destino)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	if igual {
		return 0
	}
	switch o.resposta {
	case sempreNao:
		return 0
	case sempreSim:
	default:
		if !ehTerminal(stdin) {
			imprimeAviso(stderr, fmt.Sprintf(
				"a skill embutida mudou; rode %q para atualizar (a entrada não é um terminal, nada foi gravado)",
				"draftboard skill --install"))
			return 0
		}
		fmt.Fprintln(stderr, "a skill embutida mudou nesta versão.")
		fmt.Fprintf(stderr, "reinstalar em %s? [s/N] ", caminho)
		if !confirma(stdin) {
			return 0
		}
	}
	escrito, err := skill.Instala(o.destino)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	fmt.Fprintln(stdout, escrito)
	return 0
}

// comandoVersion imprime a Versão, o commit e a data do build no stdout.
func comandoVersion(stdout, stderr io.Writer) int {
	if err := update.ImprimeVersao(stdout); err != nil {
		return imprimeErro(stderr, err)
	}
	return 0
}

// comandoUpdate troca o binário em execução pelo do último Lançamento.
//
// O stdout carrega só o que foi escrito — o caminho do binário substituído, e
// depois o da skill, se ela foi regravada —, do mesmo jeito que `render`
// imprime só os caminhos das imagens. Status, avisos e a pergunta vão para
// stderr.
//
// `--check` sai 0 sempre que a CONSULTA funcionou: a resposta é a linha única
// de stdout, que é o que um script grepa. Um código 2 para "há versão nova"
// quebraria o contrato de 0/1 do §7 do CONTRATO.
func comandoUpdate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	o, err := interpretaUpdate(args)
	if err != nil {
		return usoInvalido(stderr, err)
	}
	opcoes := update.Opcoes{}
	// Seam de teste: aponta a consulta para um servidor local. Documentado no
	// CONTRATO, deliberadamente fora da skill.
	if u := os.Getenv("DRAFTBOARD_LANCAMENTOS_URL"); u != "" {
		opcoes.BaseURL = u
	}
	atual := update.Atual()

	lancamento, maisNova, err := update.Verifica(opcoes)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	ordem, comparavel := update.Compara(atual.Versao, lancamento.Versao)
	switch {
	case !comparavel:
		imprimeAviso(stderr, fmt.Sprintf(
			"versão atual %q (binário construído sem informação de versão); não é possível comparar com %s",
			atual.Versao, lancamento.Versao))
	case ordem > 0:
		imprimeAviso(stderr, fmt.Sprintf(
			"a versão atual %s é mais nova que o último lançamento %s",
			atual.Versao, lancamento.Versao))
	}
	if !maisNova {
		fmt.Fprintf(stdout, "já na versão mais recente: %s\n", atual.Versao)
		return 0
	}
	if o.conferir {
		fmt.Fprintf(stdout, "atualização disponível: %s (atual: %s)\n", lancamento.Versao, atual.Versao)
		return 0
	}

	caminho, err := update.Aplica(opcoes, lancamento, stderr)
	if err != nil {
		return imprimeErro(stderr, err)
	}
	fmt.Fprintln(stdout, caminho)
	sincronizaComOBinarioNovo(caminho, o.resposta, stdin, stdout, stderr)
	return 0
}

// sincronizaComOBinarioNovo pede ao binário RECÉM-INSTALADO que confira a
// skill.
//
// A delegação não é rodeio: a skill é embutida com go:embed, então o processo
// em execução — o binário antigo — não tem como saber qual é a skill da versão
// nova. Quem compara tem de ser o binário novo. O terminal é repassado inteiro
// para que a pergunta seja feita no tty do usuário.
//
// Falha do filho não derruba o update: o usuário pediu troca de binário e
// teve. O aviso diz como completar o resto à mão.
func sincronizaComOBinarioNovo(binario string, r resposta, stdin io.Reader, stdout, stderr io.Writer) {
	args := []string{"skill", "--sync"}
	switch r {
	case sempreSim:
		args = append(args, "--yes")
	case sempreNao:
		args = append(args, "--no")
	}
	cmd := exec.Command(binario, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	if err := cmd.Run(); err != nil {
		imprimeAviso(stderr, fmt.Sprintf(
			"binário atualizado, mas a sincronização da skill falhou; rode %q",
			"draftboard skill --sync"))
	}
}

// ehTerminal reporta se r é um dispositivo de caractere — o mais perto de uma
// detecção de terminal que a biblioteca padrão permite, já que não há isatty na
// stdlib e dependência nova é proibida.
//
// Pipe e arquivo comum não são dispositivo de caractere, então o resultado é o
// certo nos testes e nos scripts. /dev/null é falso positivo conhecido e
// inofensivo: ler dele dá EOF na hora, o que confirma() lê como "não".
func ehTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// confirma lê uma linha e reporta se ela é um sim. Linha vazia, EOF e qualquer
// outra resposta são "não" — o padrão é sempre o que não escreve em disco.
func confirma(stdin io.Reader) bool {
	linha, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && linha == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(linha)) {
	case "s", "sim", "y", "yes":
		return true
	default:
		return false
	}
}
