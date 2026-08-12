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
func TestSkillNaoDessincroniza(t *testing.T) {
	c := skill.Conteudo()

	flags := []string{"--out", "--scale", "--notes", "--layers", "--install"}
	for _, f := range flags {
		if !strings.Contains(c, f) {
			t.Errorf("skill não documenta a flag %s da CLI", f)
		}
	}

	discriminantes := []string{"rect", "circle", "use", "slot"}
	for _, d := range discriminantes {
		if !strings.Contains(c, d) {
			t.Errorf("skill não documenta a chave discriminante de nó %q", d)
		}
	}

	// Os três padrões das flags de render também precisam aparecer.
	for _, padrao := range []string{"margin", "float", "off"} {
		if !strings.Contains(c, padrao) {
			t.Errorf("skill não documenta o modo de Nota %q", padrao)
		}
	}
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
