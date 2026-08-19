# Achados aceitos — F10

Placar: 3 blockers (5 cada) + 6 quickwins (2 cada) = **27 → retificar**.
Um achado foi **recusado** (ver o fim).

Os três blockers e dois dos quickwins caem no mesmo trecho — `dispoe`,
`ladoQueCede`, `desceAteLivrar` e a presagem de `balao.go`. Trate-os como um
conserto só, com testes separados.

---

## Blockers

### B1 — A anti-colisão desiste e aceita sobreposição quando ainda havia espaço

Dois cenários, a mesma causa: o recuo (`!achou`) devolve a primeira escolha às
cegas, sem nunca consultar `postos`.

**B1a — perto da base do Frame** (`balao.go:79-95`, `ladoQueCede` só desce).
Frame 375x812, dois Elementos 48x48 em `Y: 740`, `X: 20` e `X: 78`, com Notas
curtas → balões `{78, 749, 216, 779}` e `{136, 749, 274, 779}`: **80 px de
sobreposição na mesma altura**, e o segundo cobre o fim do texto do primeiro. Com
seis abas — uma barra inferior típica — são **14 pares cruzando**, todos no mesmo
`y`, com **700 px vazios acima**. A mesma linha em `Y: 40/200/400/600` dá 0
pares: o defeito é exatamente a falta de espaço para descer.

**B1b — Frame estreito** (`balao.go:85`, o portão de largura). Os dois lados são
descartados antes de qualquer descida, então a anti-colisão é **pulada por
completo**. Frame 60x400 com 4 Elementos anotados em `Y=8/40/72/104` → **6 de 6
pares se cruzam**, com os balões somando 300 px de altura num Frame de 400 px:
havia espaço para empilhar. A imagem sai como um bloco preto único.

Nenhum dos dois é o caso documentado de "Frame cheio demais", porque cabe. Os
dois violam a linha que o próprio diff acrescentou ao `CONTRACT.md` §6: **"Dois
balões nunca se cruzam"**.

Correção: no ramo `!achou` de `dispoe`, rodar `desceAteLivrar` contra `postos` a
partir do desejado — e, quando nem isso resolver, tentar o espelho (subir até
livrar, limitado pelo respiro) antes de aceitar a sobreposição. Só prender à tela
depois disso. A aceitação de sobreposição continua sendo o último recurso
documentado, mas passa a ser de fato o último.

### B2 — Coordenada absurda trava o processo para sempre

`internal/notes/notes.go:195` + `balao.go:140`. `x`, `chamadaX` e `ancoraX` nunca
são limitados ao Frame.

Cenário: `rect: {x: 5, y: 5, w: 1e300, h: 5}` com `note:` →
`draftboard render doc.yaml --notes` **nunca retorna** (morto por `timeout` aos
60 s, código 124). Sem `--notes`, o mesmo Documento sai com código 0. Pilha:
`notes.(*Plano).Desenha → render.Canvas.Linha → freetype/raster.saveCell`. O
`w: 1e300` só rende um **aviso** ("Elemento fora do Frame"), nunca um Erro.

**Pré-existente** — no `main` o padrão `margin` e o `--notes float` travam igual.
Não é desculpa neste repo, e F10 promove esse caminho a único.

Correção: prender `x`, `chamadaX` e `ancoraX` ao intervalo do Frame, de modo que
nenhuma coordenada gigante chegue ao `Canvas`.

---

## Quickwins

### Q1 — O respiro não existe contra a borda da tela, nos quatro lados

`balao.go:56,85,89,144`. Os limites presos são aplicados ao **bloco de texto**,
mas o balão é `texto ± respiro`: na saturação o balão encosta em `0`, `fl` e `fa`.
Medido no golden: `{302, 30, 400, 60}` num Frame de 400 px — folga direita **0**,
visível em `testdata/f4/flutuante.webp`. Vertical idem: âncora no topo dá
`y0 = 0`; no fundo, `y1 = fa`.
**Pré-existente na fórmula** (`main:internal/notes/flutuante.go` tem a mesma
linha); o que F10 acrescentou foram as guardas novas, que replicaram o erro.
E `plano_test.go:57` **codifica o bug**, afirmando `x1 <= fl` — passa com folga 0.
Correção: presar pelo retângulo do balão, não pelo bloco de texto.

### Q2 — Frame minúsculo: o balão sai da tela

`balao.go:56,136-146`. Frame 20x20, Elemento 5x5 em `0,0` → balão
`{0, 0, 29, 29.9}`: 9 px fora à direita e 9,9 fora embaixo. Abaixo de
`2*respiro + alturaDeLinha` ≈ 30 px isso é geometricamente inevitável — o que
falta é a decisão explícita. Correção: presar o retângulo do balão à tela como
último recurso, aceitando o corte do texto, e documentar.

### Q3 — `Planeja` roda antes de `cabeNaTela` e aloca 256 MB à toa

`comandos.go:80-83`. A régua satura em `LimiteDeArea`.
`render testdata/f4/notas.yaml --notes --scale 9000` → RSS **282.771.456 B**; o
mesmo sem `--notes` → 11.911.168 B. Os dois terminam com o mesmo erro de teto.
A justificativa do comentário ("as margens do plano entram na conta do teto")
**morreu com o Chrome**: `Margens()` agora é sempre 0.
Correção: `cabeNaTela(o, indice, f, 0, 0, 0, 0)` antes do `if o.notas`.

### Q4 — `--notes=VALOR` escorre para os posicionais

`opcoes.go:57`. `expandeIguais` parte a forma `=` em dois e o valor vira
posicional. `render --notes= doc.yaml` → `argumento em excesso: "doc.yaml"`,
**culpando o Documento**; `render --notes=` → `erro: : arquivo não encontrado`,
exatamente o formato que o `main_test.go:422` declara defeituoso;
`--notes=true doc.yaml` → `argumento em excesso: "true"`. Compare com `--out=` e
`--scale=`, que têm mensagem própria.
Correção: no `case "--notes"`, tratar também a forma `=` e devolver mensagem
própria em vez de deixar escorrer.

### Q5 — Os dois testes de garantia só usam Frame largo e folgado

`plano_test.go:34,56`. `TestBaloesNaoSeCruzam` e `TestBalaoNaoSaiDoFrame` só usam
`colunaAnotada` num Frame 400x600: o ramo de recuo (`!achou`) — onde as duas
garantias caem — não é exercitado. Mutação: apagar o guarda
`if cand.y+n.a+respiro > fa { continue }` → **suíte inteira verde**, e com 20
Notas num Frame 400x300 o mutante põe 3 balões inteiramente **abaixo da tela**.
Correção: casos de Frame estreito (60x400), de coluna saturada (20 Notas em
400x300) e de balão mais alto que o Frame, nas mesmas duas asserções.

### Q6 — O critério `direitaDoElemento` de `colhe` não tem teste

`notes.go:239` / `plano_test.go:78`. `TestOrdemDeDeclaracaoNaoMudaOPlano` só
fecha o desempate por `esquerdaDoElemento`; o antigo
`TestDesempateDaOrdenacaoPelaBordaDoElemento` sumiu sem substituto. Mutação:
remover as três linhas de `direitaDoElemento` → suíte verde. Com
`rect{x:100,y:0,w:50,h:20,note:"A"}` e `rect{x:100,y:0,w:150,h:20,note:"A"}`, o
plano passa a depender da ordem de declaração — a instabilidade que o item 8
proíbe.
Correção: caso com mesma altura de âncora e mesma borda esquerda, larguras
diferentes, comparando `balao()` entre a ordem direta e a invertida.

---

## Recusado

- **"O relatório não descreve as imagens regeradas"** — recusado, e a culpa é do
  Executor. O implementador **descreveu as duas imagens em detalhe** no relatório
  de entrega; quem as perdeu fui eu, ao condensar o relatório em
  `/tmp/f10-report.md` para o validador. O achado é artefato do meu resumo, não
  da entrega. Eu também abri as duas imagens e confirmei a descrição.

## Verificado sem achado, e vale registrar

- **`desceAteLivrar` termina, com prova**: `b.y0` cresce estritamente e só assume
  valores do conjunto finito `{q.y1 + espacoEntreBaloes : q ∈ postos}`, logo no
  máximo `len(postos)` movimentos.
- **200 Notas num Frame de 50 px**: `Planeja` em 0,06 s, 0 NaN/Inf, `render`
  completo em 0,28 s.
- **Custo**: quadrático no comparador, sem impacto no relógio na faixa útil
  (16 000 Notas → 2,01 s, dominado pela medição de texto). Vira gargalo só acima
  de ~20 000 Notas, que o teto de área já torna difícil de alcançar.
- **`sort.SliceStable` não tem observável** — confirmado por dois validadores,
  independentemente. O implementador estava certo. Fica como cinto e suspensório.
- Todas as outras formas de `--notes` conferidas; texto multibyte, `\n` explícito,
  Nota vazia e só-espaços, os quatro cantos, Frame 1x1, palavra indivisível maior
  que o Frame — todos corretos ou documentados.
