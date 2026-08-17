package skill_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardotorresdev/draftboard/internal/skill"
)

func TestConteudoTrazAsSecoesEssenciais(t *testing.T) {
	c := skill.Conteudo()
	if strings.TrimSpace(c) == "" {
		t.Fatal("Conteudo() devolveu texto vazio")
	}
	secoes := []string{
		"name: draftboard",
		"description:",
		"frames:",
		"elements:",
		"draftboard render",
		"draftboard inspect",
		"draftboard validate",
		"draftboard skill",
		"draftboard version",
		"draftboard update",
		"Elevação",
		"Tom",
		"Componente",
		"Slot",
		"Repetição",
		"Nota",
		"documento <nome>",
		".webp",
		"erro: ",
		"aviso: ",
	}
	for _, s := range secoes {
		if !strings.Contains(c, s) {
			t.Errorf("skill não menciona %q", s)
		}
	}
}

func TestConteudoAbreComFrontmatterDeSkill(t *testing.T) {
	c := skill.Conteudo()
	if !strings.HasPrefix(c, "---\n") {
		t.Fatalf("skill não começa com frontmatter YAML; começo = %q", primeiraLinha(c))
	}
	fim := strings.Index(c[4:], "\n---\n")
	if fim < 0 {
		t.Fatal("frontmatter YAML não é fechado por uma linha ---")
	}
	frontmatter := c[4 : 4+fim]
	if !strings.Contains(frontmatter, "name: draftboard") {
		t.Errorf("frontmatter sem `name: draftboard`; frontmatter = %q", frontmatter)
	}
	if !strings.Contains(frontmatter, "description:") {
		t.Errorf("frontmatter sem `description:`; frontmatter = %q", frontmatter)
	}
}

// TestSkillNaoDessincroniza protege contra o esquecimento mais provável: mudar
// a CLI ou o schema e não atualizar a skill.
//
// As asserções são de LINHA INTEIRA, não de substring. Substring nua não pega
// nada aqui: "slot" casa dentro de "slots", "--layers" casa no bloco de uso da
// CLI, e apagar a linha da tabela que documenta qualquer um dos dois passaria
// despercebido. Cada linha abaixo é o único lugar da skill que carrega aquele
// fato; mexer no fato quebra o teste.
func TestSkillNaoDessincroniza(t *testing.T) {
	c := skill.Conteudo()

	linhas := map[string]string{
		"tabela do nó use (box obrigatório)":  "| `use: \"./comp.yaml\"` | caminho relativo | `box` (**obrigatório**), `slots`, `id`, `note`, `repeat` |",
		"tabela do nó slot (box obrigatório)": "| `slot: \"nome\"` | nome do Slot | `box` (**obrigatório**), `default`, `id`, `note`, `repeat` |",
		"lista de chaves discriminantes":      "Exatamente **uma** chave discriminante por nó: `rect`, `circle`, `use`, `slot` ou `control`.",
		"regra de slot só em Componente":      "- `slot` só é válido dentro de Componente — nunca em Documento.",
		"nó slot no exemplo de Componente":    "  - slot: \"body\"",
		"tabela da flag --out":                "| `--out DIR` | `render` | `.` | diretório de saída |",
		"tabela da flag --scale":              "| `--scale N` | `render` | `1` | multiplicador float > 0 de toda a imagem |",
		"tabela da flag --notes":              "| `--notes MODO` | `render` | `margin` | `margin` (Notas no Chrome), `float` (sobre o Frame), `off` (sem Notas) |",
		"tabela da flag --layers":             "| `--layers` | `render` | desligado | uma imagem por Camada, cumulativa (a Camada e todas abaixo) |",
		"tabela da flag --install":            "| `--install [DIR]` | `skill` | `~/.claude/skills` | grava a skill em `<DIR>/draftboard/SKILL.md` |",
		"tabela da flag --sync":               "| `--sync [DIR]` | `skill` | `~/.claude/skills` | regrava a skill só se o conteúdo mudou, perguntando antes |",
		"tabela da flag --check":              "| `--check` | `update` | desligado | só reporta se há versão nova; não escreve nada |",
		"tabela da flag --yes":                "| `--yes` | `skill`, `update` | desligado | responde `s` a qualquer pergunta |",
		"tabela da flag --no":                 "| `--no` | `skill`, `update` | desligado | responde `n` a qualquer pergunta |",
		"nome de saída sem --layers":          "| sem `--layers` | `<doc>-<frame>.webp` |",
		"nome de saída com --layers":          "| com `--layers` | `<doc>-<frame>-<nn>-<camada>.webp`, `nn` de dois dígitos a partir de `01` |",
		"fórmula do Círculo no Frame":         "Círculo:  L = A = d/100*FL      (largura nos dois eixos — nunca vira elipse)",
		"fórmula do Círculo no espaço local":  "Círculo:  L = A = d/100*bl",
		"linha de Elemento do inspect":        "      <caminho> <retangulo|circulo> <X>,<Y> <L>x<A> tom=<T> elev=<E>[ round][ de=<componente>][ controle=<nome> <parâmetros>]",
		"tabela do nó control":                "| `control: nome` | nome do catálogo | `box` (**obrigatório**), `id`, `note`, `repeat`, e os campos do Controle |",
		"regra de Controle fechado":           "- `control` é fechado: não aceita `slots`, `default` nem `round`, e não recebe conteúdo.",
		"catálogo do Controle button":         "| `button` | `label` | sem `label`, o rótulo vira barra cinza |",
		"catálogo do Controle input":          "| `input` | `label` | sem `label`, o rótulo vira barra cinza |",
		"catálogo do Controle tabs":           "| `tabs` | `items`, `active` | `items: 3`, `active: 1` |",
		"catálogo do Controle slider":         "| `slider` | `value` | `value: 50` |",
		"catálogo do Controle checkbox":       "| `checkbox` | `label`, `active` | `active: 0` (desmarcado) |",
		"catálogo do Controle radio":          "| `radio` | `items`, `active` | `items: 3`, `active: 1` |",
		"catálogo do Controle toggle":         "| `toggle` | `active` | `active: 0` (desligado) |",
		"catálogo do Controle accordion":      "| `accordion` | `items`, `active` | `items: 3`, `active: 1`; a seção ativa abre |",
		"catálogo do Controle dropdown":       "| `dropdown` | `label` | sem `label`, o rótulo vira barra cinza |",
		"catálogo do Controle avatar":         "| `avatar` | `label` | sem `label`, é só o disco |",
		"catálogo do Controle badge":          "| `badge` | `label` | sem `label`, o rótulo vira barra cinza |",
		"catálogo do Controle progress":       "| `progress` | `value` | `value: 50` |",
		"regra dos dois estados":              "- Em `checkbox` e `toggle`, `active` só aceita 0 ou 1: eles têm dois estados, não lista.",
		"regra das duas formas de items":      "- `items` aceita **número** (itens sem texto) ou **lista de rótulos** (itens com texto).",
		"regra da fonte derivada":             "- O tamanho da fonte do Rótulo é derivado da altura da área — **não existe campo de",
	}
	for fato, linha := range linhas {
		if !contemLinha(c, linha) {
			t.Errorf("skill perdeu a %s; linha esperada:\n%s", fato, linha)
		}
	}

	// O nó rect é o único que aceita `round`, e circle o único com `d`.
	for fato, linha := range map[string]string{
		"tabela do nó rect":   "| `rect: {x, y, w, h}` | geometria em % | `round`, `id`, `note`, `repeat` |",
		"tabela do nó circle": "| `circle: {x, y, d}` | geometria em % | `id`, `note`, `repeat` |",
	} {
		if !contemLinha(c, linha) {
			t.Errorf("skill perdeu a %s; linha esperada:\n%s", fato, linha)
		}
	}
}

// TestSkillDocumentaAsRegrasDeElevacao cobre CONTRACT.md §3: sem o desempate e
// sem a regra da bounding box declarada, um agente não consegue prever o Tom.
func TestSkillDocumentaAsRegrasDeElevacao(t *testing.T) {
	c := skill.Conteudo()

	regras := map[string]string{
		"piso por Camada":            "- Cada Camada `i` parte de um piso: `base[0] = 0` e `base[i] = max(base[i-1], maiorElevacaoNaCamada[i-1]) + 1`.",
		"desempate do pai":           "- O pai de um Elemento é o **último Elemento já pintado** (em qualquer Camada ≤ `i`) cuja bounding box contém a do Elemento, com contenção inclusiva. Sem pai, `elevacaoDoPai = base[i]`.",
		"fórmula da Elevação":        "- `Elevacao = max(elevacaoDoPai, base[i]) + 1` — pode subir mais de um degrau de uma vez quando o pai está numa Camada inferior.",
		"bounding box declarada":     "- Elemento recortado pela borda do Frame entra na contenção com a bounding box **declarada**, não com a recortada.",
		"Nota fora da Elevação":      "- `note` é a Nota: texto anexado ao Elemento, fora do desenho. A Nota **não participa da Elevação e não aparece no export por Camada** (`--layers`); some inteira com `--notes off`.",
		"ausência de cor declarada":  "**Não existe declaração de cor.** O Tom (cinza de 100, quase branco, a 900, quase preto)",
		"ordem de pintura declarada": "- Ordem de pintura: Camadas na ordem declarada, Elementos na ordem declarada dentro da Camada.",
	}
	for fato, linha := range regras {
		if !contemLinha(c, linha) {
			t.Errorf("skill perdeu a regra de %s; linha esperada:\n%s", fato, linha)
		}
	}
}

// contemLinha reporta se o texto tem uma linha exatamente igual à esperada,
// desconsiderando espaço em branco no fim da linha.
func contemLinha(texto, esperada string) bool {
	esperada = strings.TrimRight(esperada, " \t")
	for _, l := range strings.Split(texto, "\n") {
		if strings.TrimRight(l, " \t\r") == esperada {
			return true
		}
	}
	return false
}

func TestImprimeEscreveExatamenteOConteudo(t *testing.T) {
	var buf bytes.Buffer
	if err := skill.Imprime(&buf); err != nil {
		t.Fatalf("Imprime devolveu erro: %v", err)
	}
	if buf.String() != skill.Conteudo() {
		t.Errorf("Imprime escreveu %d bytes, Conteudo() tem %d bytes", buf.Len(), len(skill.Conteudo()))
	}
}

func TestImprimePropagaErroDoWriter(t *testing.T) {
	if err := skill.Imprime(writerQueFalha{}); err == nil {
		t.Fatal("Imprime devolveu nil para um writer que sempre falha")
	}
}

type writerQueFalha struct{}

func (writerQueFalha) Write([]byte) (int, error) {
	return 0, os.ErrClosed
}

func TestInstalaGravaNoSubdiretorioDraftboard(t *testing.T) {
	dir := t.TempDir()

	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}

	esperado := filepath.Join(dir, "draftboard", "SKILL.md")
	if caminho != esperado {
		t.Errorf("Instala devolveu %q, esperado %q", caminho, esperado)
	}
	gravado, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler o arquivo instalado: %v", err)
	}
	if string(gravado) != skill.Conteudo() {
		t.Error("arquivo instalado não bate com Conteudo()")
	}
}

func TestInstalaCriaDiretoriosIntermediarios(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "casa", ".claude", "skills")

	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}
	if _, err := os.Stat(caminho); err != nil {
		t.Fatalf("arquivo não existe no caminho devolvido: %v", err)
	}
}

func TestInstalaEIdempotente(t *testing.T) {
	dir := t.TempDir()

	primeiro, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("primeira instalação devolveu erro: %v", err)
	}
	segundo, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("segunda instalação devolveu erro: %v", err)
	}
	if primeiro != segundo {
		t.Errorf("caminhos divergem entre execuções: %q e %q", primeiro, segundo)
	}
	gravado, err := os.ReadFile(segundo)
	if err != nil {
		t.Fatalf("ler o arquivo reinstalado: %v", err)
	}
	if string(gravado) != skill.Conteudo() {
		t.Error("arquivo reinstalado não bate com Conteudo()")
	}
}

// TestInstalaSemDiretorioUsaClaudeSkills fixa o destino padrão. É o único
// caminho que escreve na HOME do operador, então o valor exato importa.
func TestInstalaSemDiretorioUsaClaudeSkills(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa) // Windows

	caminho, err := skill.Instala("")
	if err != nil {
		t.Fatalf("Instala(\"\") devolveu erro: %v", err)
	}

	esperado := filepath.Join(casa, ".claude", "skills", "draftboard", "SKILL.md")
	if caminho != esperado {
		t.Errorf("Instala(\"\") gravou em %q, esperado %q", caminho, esperado)
	}
	gravado, err := os.ReadFile(esperado)
	if err != nil {
		t.Fatalf("ler o arquivo instalado no destino padrão: %v", err)
	}
	if string(gravado) != skill.Conteudo() {
		t.Error("arquivo instalado no destino padrão não bate com Conteudo()")
	}
}

// TestInstalaNaoSegueLinkSimbolicoNoDestino: um SKILL.md preexistente apontando
// para fora do destino não pode virar uma escrita no alvo do link.
func TestInstalaNaoSegueLinkSimbolicoNoDestino(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "skills")
	if err := os.MkdirAll(filepath.Join(dir, "draftboard"), 0o755); err != nil {
		t.Fatalf("preparar destino: %v", err)
	}
	alvo := filepath.Join(base, "vitima.txt")
	const intocado = "conteudo que nao pode ser sobrescrito"
	if err := os.WriteFile(alvo, []byte(intocado), 0o644); err != nil {
		t.Fatalf("preparar vítima: %v", err)
	}
	armadilha := filepath.Join(dir, "draftboard", "SKILL.md")
	if err := os.Symlink(alvo, armadilha); err != nil {
		t.Skipf("ambiente não permite criar link simbólico: %v", err)
	}

	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}

	sobrevivente, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("ler a vítima: %v", err)
	}
	if string(sobrevivente) != intocado {
		t.Errorf("Instala seguiu o link e sobrescreveu %s", alvo)
	}
	info, err := os.Lstat(caminho)
	if err != nil {
		t.Fatalf("lstat do arquivo instalado: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("o arquivo instalado continua sendo um link simbólico")
	}
	gravado, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler o arquivo instalado: %v", err)
	}
	if string(gravado) != skill.Conteudo() {
		t.Error("arquivo instalado não bate com Conteudo()")
	}
}

// TestInstalaSegueLinkSimbolicoNoDiretorio fixa uma decisão, não um acidente:
// um link simbólico em <dir>/draftboard É seguido. Apontar o diretório de
// skills para outro lugar é arranjo legítimo de operador, e nesse caso gravar
// no alvo é o comportamento esperado. Contrasta de propósito com
// TestInstalaNaoSegueLinkSimbolicoNoDestino, onde o link está no arquivo final
// e NÃO é seguido.
func TestInstalaSegueLinkSimbolicoNoDiretorio(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "skills")
	fora := filepath.Join(base, "skills-de-verdade")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("preparar dir: %v", err)
	}
	if err := os.MkdirAll(fora, 0o755); err != nil {
		t.Fatalf("preparar alvo: %v", err)
	}
	if err := os.Symlink(fora, filepath.Join(dir, "draftboard")); err != nil {
		t.Skipf("ambiente não permite criar link simbólico: %v", err)
	}

	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}

	if caminho != filepath.Join(dir, "draftboard", "SKILL.md") {
		t.Errorf("Instala devolveu %q, esperado o caminho pelo link", caminho)
	}
	// A skill tem de aparecer no alvo do link, que é o ponto da decisão.
	gravado, err := os.ReadFile(filepath.Join(fora, "SKILL.md"))
	if err != nil {
		t.Fatalf("ler o arquivo no alvo do link: %v", err)
	}
	if string(gravado) != skill.Conteudo() {
		t.Error("arquivo no alvo do link não bate com Conteudo()")
	}
}

// TestInstalaPreservaInstalacaoAnteriorQuandoFalha: a gravação é atômica, então
// um erro no meio não pode deixar o operador sem SKILL.md nem com um truncado.
func TestInstalaPreservaInstalacaoAnteriorQuandoFalha(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignora permissões de diretório")
	}
	dir := t.TempDir()

	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("instalação inicial devolveu erro: %v", err)
	}
	destino := filepath.Dir(caminho)

	// Diretório somente-leitura: o temporário não pode ser criado.
	if err := os.Chmod(destino, 0o500); err != nil {
		t.Fatalf("restringir permissões: %v", err)
	}
	t.Cleanup(func() { os.Chmod(destino, 0o755) })

	if _, err := skill.Instala(dir); err == nil {
		t.Fatal("Instala devolveu nil num diretório somente-leitura")
	}

	sobrevivente, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("a instalação anterior sumiu: %v", err)
	}
	if string(sobrevivente) != skill.Conteudo() {
		t.Error("a instalação anterior foi corrompida por uma instalação que falhou")
	}
}

// TestInstalaNaoDeixaTemporarioParaTras: o arquivo temporário da gravação
// atômica não pode virar lixo no diretório de skills.
func TestInstalaNaoDeixaTemporarioParaTras(t *testing.T) {
	dir := t.TempDir()

	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}

	entradas, err := os.ReadDir(filepath.Dir(caminho))
	if err != nil {
		t.Fatalf("listar o destino: %v", err)
	}
	if len(entradas) != 1 || entradas[0].Name() != "SKILL.md" {
		var nomes []string
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Errorf("destino tem %v, esperado só [SKILL.md]", nomes)
	}
}

func TestInstalaFalhaQuandoODestinoNaoPodeSerCriado(t *testing.T) {
	// Um arquivo comum no lugar do diretório impede o MkdirAll.
	base := t.TempDir()
	obstaculo := filepath.Join(base, "skills")
	if err := os.WriteFile(obstaculo, []byte("nao sou diretorio"), 0o644); err != nil {
		t.Fatalf("preparar obstáculo: %v", err)
	}

	if _, err := skill.Instala(obstaculo); err == nil {
		t.Fatal("Instala devolveu nil apesar de o destino ser um arquivo comum")
	}
}

func primeiraLinha(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestEstaSincronizadaReconheceInstalacaoIgual cobre o caso em que `skill
// --sync` não tem nada a fazer, que é o caminho normal de um update.
func TestEstaSincronizadaReconheceInstalacaoIgual(t *testing.T) {
	dir := t.TempDir()
	if _, err := skill.Instala(dir); err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}
	igual, caminho, err := skill.EstaSincronizada(dir)
	if err != nil {
		t.Fatalf("EstaSincronizada devolveu erro: %v", err)
	}
	if !igual {
		t.Errorf("skill recém-instalada foi reportada como dessincronizada")
	}
	if esperado := filepath.Join(dir, "draftboard", "SKILL.md"); caminho != esperado {
		t.Errorf("caminho = %q, esperado %q", caminho, esperado)
	}
}

func TestEstaSincronizadaReconheceInstalacaoDiferente(t *testing.T) {
	dir := t.TempDir()
	caminho, err := skill.Instala(dir)
	if err != nil {
		t.Fatalf("Instala devolveu erro: %v", err)
	}
	if err := os.WriteFile(caminho, []byte("skill de outra versão\n"), 0o644); err != nil {
		t.Fatalf("não foi possível alterar a skill instalada: %v", err)
	}
	igual, _, err := skill.EstaSincronizada(dir)
	if err != nil {
		t.Fatalf("EstaSincronizada devolveu erro: %v", err)
	}
	if igual {
		t.Errorf("skill alterada foi reportada como sincronizada")
	}
}

// TestEstaSincronizadaTrataDestinoAusenteComoDessincronizado: instalar é o que
// resolve tanto o arquivo diferente quanto o arquivo ausente, então ausência
// não é erro.
func TestEstaSincronizadaTrataDestinoAusenteComoDessincronizado(t *testing.T) {
	igual, caminho, err := skill.EstaSincronizada(t.TempDir())
	if err != nil {
		t.Fatalf("EstaSincronizada devolveu erro: %v", err)
	}
	if igual {
		t.Errorf("destino ausente foi reportado como sincronizado")
	}
	if caminho == "" {
		t.Errorf("EstaSincronizada não devolveu o caminho conferido")
	}
}
