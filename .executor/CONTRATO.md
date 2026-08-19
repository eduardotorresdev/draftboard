# Contrato — Rótulo no Retângulo (F9, F10, F11)

Congelado pelo Executor. Revisão de contrato aplicada: 6 blockers, 5 quickwins e
5 estruturais resolvidos abaixo. Qualquer mudança daqui em diante passa por ele.

## Fatiamento e divisibilidade

| Feature | Escopo (itens do handoff) |
| --- | --- |
| **F9 — Rótulo no Retângulo** | 1, 2, 5, e a parte de `svg.go` + tabela de nós da SKILL.md do item 8 |
| **F10 — Aposentadoria do Margem** | 6, 7, e a seção CLI da SKILL.md do item 8 |
| **F11 — Diagnóstico e `--fix`** | 3, 4 |

Divisibilidade F9 × F10: contrato explícito 2, fronteira de arquivos 1
(`SKILL.md` e `skill_test.go` são comuns, em linhas disjuntas), independência de
dados 2, ordem livre 2, verificação isolada 2. **Soma 9 → paralelo**, cada uma em
worktree própria.

Divisibilidade F11 × (F9, F10): contrato 1, fronteira 0 (`comandos.go`,
`internal/resolve/`, `internal/scene/`), dados 1, ordem 0, verificação 0.
**Soma 2 → serial**, depois das duas.

## Propriedade de arquivos

Regra: quem não é dono não escreve. Precisou escrever fora, **para e reporta**.

| Arquivo | Dono | Recorte |
| --- | --- | --- |
| `internal/schema/` | F9 | e F11 depois, para posição de bytes |
| `internal/resolve/` | F9 | `rotulo.go` é novo |
| `internal/scene/scene.go` | F9 | menos o comentário de `TomChrome`, que é de F10 |
| `internal/render/` | **ninguém** | leitura para F9; só os **comentários** de `canvas.go` são de F10 |
| `internal/inspect/` | F9 | |
| `internal/board/svg.go`, `prancheta.css`, `recursos.go` | F9 | |
| `internal/notes/` | F10 | `margem.go` é removido |
| `opcoes.go`, `comandos.go` | F10 | e F11 depois |
| `main_test.go`, `f6_test.go` | F10 | |
| `f9_test.go`, `testdata/f9/` | F9 | novos |
| `testdata/f4/` | F10 | |
| `CONTRACT.md` §6 e §7 | F10 | `Planeja` e a linha de uso do `render` |
| `internal/skill/SKILL.md` | ambos | ver o particionamento abaixo |
| `internal/skill/skill_test.go` | ambos | ver o particionamento abaixo |
| `CONTEXT.md`, `docs/adr/` | **ninguém** | congelados |

### Particionamento da SKILL.md e do skill_test.go

**F9** — só estas linhas:
- `SKILL.md:103`, tabela do nó `rect` (ganha `label`), e o `skill_test.go:125`
  que a compara byte a byte.
- `SKILL.md`, a gramática da linha de Elemento do `inspect`, e o
  `skill_test.go:98` que a compara.
- A seção nova do Rótulo no Retângulo.
- A linha do nó `circle` **não muda**: Círculo não ganha `label`.

**F10** — só estas linhas:
- `SKILL.md:254`, o bloco de uso do `render`, e `SKILL.md:267`, a tabela da flag
  `--notes`, com o `skill_test.go:84` que a compara.
- `SKILL.md:113`, o bullet do `note` que termina em "some inteira com
  `--notes off`", com o `skill_test.go:144` que o compara.
- `SKILL.md:240`, "O Chrome usa 900, reservado" — Chrome não é mais termo do
  domínio.

## Interfaces congeladas

### 1. `schema.No.Rotulo string` — `label:` no `rect`

O `label:` declarado num nó `rect`. Vazio quando ausente. Só `TipoRetangulo` o
preenche; o Controle continua guardando o seu em `no.Controle.Rotulo`.

**`label` já é rejeitado hoje** por um caminho genérico: `chavesDeControle`
(`internal/schema/decode.go:36`) contém `"label"`, e o laço de
`decode.go:414` devolve `campo %q só é permitido em Controle` para todo nó que
não é Controle. Acrescentar um bloco no padrão de `round` **não basta**: a
rejeição genérica dispara antes.

A forma congelada:

- `chavesDeControle` **continua com `"label"`**: o laço de `decode.go:728`
  depende dela para ler o `label` do próprio Controle.
- O laço de rejeição de `decode.go:414` passa a percorrer uma lista nova,
  `chavesSoDeControle = []string{"items", "active", "value"}`, e mantém a
  mensagem `campo %q só é permitido em Controle`.
- `label` ganha regra própria, ao lado da de `round`:

      campo "label" só é permitido em Retângulo ou Controle

  disparada quando `m.valores["label"] != nil` e o Tipo não é `TipoRetangulo`
  nem `TipoControle`. Mensagem literal, com arquivo e `local`, como `round`.

### 2. `scene.Elemento.Local string`

O caminho de chaves YAML do **nó de origem**, atravessando a cadeia de
Componentes: exatamente `ctx.prefixo + no.Local`, o mesmo valor que hoje vai
para `r.aviso`/`r.erro`. Preenchido em **todo** Elemento emitido — peça de
Controle e Rótulo inclusive.

`Local` **não é único**: um Controle materializa oito peças e um Retângulo
rotulado materializa dois Elementos, todos do mesmo nó e portanto com o mesmo
`Local`. Quem identifica Elemento continua sendo `Caminho`. Quem quiser falar do
nó que o autor escreveu — F11, no Aviso e no `--fix` — deduplica por `Local`.
O comentário do campo diz as duas coisas.

`Elemento` **não** ganha o arquivo: um `rect` inline vive sempre no Documento, e
o Documento é o caminho que o comando já recebeu na linha de comando. `Origem`
não vazia é o sinal de que o Elemento veio de Componente — e é justamente onde o
`--fix` não pode tocar.

Produzido por: **F9**. Consumido por: **F11**.

### 3. `scene.Elemento.Rotulo` na cabeça do Retângulo

O campo já existe. Passa a ser preenchido **também** no Elemento de
`Forma: Retangulo` que declarou `label:`. Consequência: `Rotulo != ""` deixa de
implicar `Forma == Texto`, e o comentário do campo tem que dizer isso.

Regra que fecha a ambiguidade, e que o comentário registra:

> O `Rotulo` da cabeça existe **só para o `inspect`**. A área de referência do
> Rótulo — a que se desenha, a que se recorta e a que um dia se mede — é sempre
> a do Elemento de `Forma: Texto`. Quem desenha e quem mede despacham por
> `Forma`, nunca por `Rotulo != ""`.

Sem isso, F11 varre `Rotulo != ""`, acha dois Elementos por Rótulo, e mede o
transbordo contra a área do bloco inteiro, onde sempre cabe.

### 4. Materialização do Rótulo

O Rótulo do Retângulo é um `scene.Elemento` de `Forma: Texto`, `Interno: true`,
`Controle: ""`, emitido pelo achatamento **imediatamente depois** do seu
Retângulo, na mesma Camada, com a geometria do Retângulo como provisória.

- **Caminho**: `<caminho do Retângulo>/rotulo`, no precedente de
  `caminhoDaPeca`. Passa por `r.emite` como qualquer outro, e portanto por
  `caminhoUnico`. Tem teste próprio: sem segmento fixo, `caminhoUnico` inventa
  `grade~2` e a Prancheta mostra um Elemento fantasma no painel de inspeção.
- **Orçamento**: paga `r.debita(1, local)`, como as peças de Controle pagam em
  `controle()`. Sem isso, `rect + label + repeat: {n: 10000}` materializa 20 000
  Elementos sob um teto de 10 000.
- **Silêncio**: não emite aviso geométrico próprio — nem "fora do Frame", nem
  "área zero". A geometria dele é derivada, o autor não a declarou, e um aviso
  duplicado no mesmo `Local` não tem como ser lido.
- **Adjacência**: uma passagem posterior, antes de `atribuiElevacao`, corrige
  geometria e Alinhamento pela contenção, e identifica o dono pela adjacência —
  o Elemento anterior na fatia. Documente a invariante nos dois pontos e
  prove-a com teste.

Imediatamente depois, e não no fim da Camada, porque a Superfície do Rótulo tem
que ser o Retângulo que o carrega: é dele que vêm a Elevação e o Tom que dão
contraste ao texto. Empilhado no fim, um filho qualquer viraria a Superfície.

#### 4b. Emenda: o Rótulo sobe para o fim da Camada DEPOIS da Elevação

A ordem de emissão acima resolve a Elevação e **quebra a pintura**: emitido antes
dos filhos, o Rótulo é pintado antes deles, e qualquer filho que cubra a faixa do
topo o apaga da imagem em silêncio. O caso é o mais trivial que existe — uma
região com barra de cabeçalho:

    rect {x:5, y:10, w:90, h:80}  label: "Resultados"
    rect {x:5, y:10, w:90, h:20}  id: cabecalho

O `inspect` imprime `rotulo="Resultados"`, a geometria e o Tom saem certos, e o
WebP sai sem texto nenhum. Isso desfaz a promessa da ADR-0002 — "faixa no topo,
fora do caminho deles" — sem nenhum aviso.

Congelado, então, em duas passagens e não uma:

1. **Antes de `atribuiElevacao`** — `posicionaRotulos` fixa geometria e
   Alinhamento pela contenção, com o Rótulo ainda adjacente ao dono.
2. **Depois de `atribuiElevacao`** — uma segunda passagem move cada Elemento de
   `Forma: Texto` com `Controle == ""` para o **fim da sua Camada**, preservando
   a Elevação e o Tom já calculados.

É o que dá as duas coisas ao mesmo tempo: o Tom vem do Retângulo que carrega o
Rótulo, e o texto é pintado por cima de tudo que a Camada desenha. Uma Camada
posterior continua cobrindo — isso é a Elevação funcionando, não um defeito.

A adjacência continua sendo a invariante das duas primeiras fases, e continua
precisando de teste. Ela deixa de valer só depois da segunda passagem.

#### 4c. Emenda: o caminho do Rótulo usa o caminho JÁ desambiguado

`caminhoUnico` pode sufixar o caminho do Retângulo, e o Rótulo tem que pendurar
no caminho que o dono **de fato recebeu**. Com dois `rect` de `id: bloco` e
`label:` no mesmo Frame, montar o caminho do Rótulo sobre o valor bruto produz
`bloco`, `bloco/rotulo`, `bloco~2`, `bloco/rotulo~2` — e o Rótulo do segundo
bloco fica pendurado no caminho do primeiro. Quem parear Rótulo↔dono por prefixo
— o painel da Prancheta e o `--fix` de F11 — atribui o texto ao Retângulo errado.

`acrescenta`/`emite` devolvem o caminho já desambiguado, e é esse valor que vai
para a montagem do caminho do Rótulo.

### 5. Regra de posição

Retângulo que **contém outro Elemento** → faixa no topo, `scene.AEsquerda`.
Retângulo **vazio** → caixa centrada na vertical, `scene.AoCentro`.

- A contenção é `contemGeometricamente` (`internal/resolve/elevacao.go:53`).
  **Chame a função existente**; não escreva uma segunda.
- A varredura **ignora o próprio Retângulo**, por identidade de ponteiro, não por
  geometria: `contemGeometricamente(e, e)` é `true` porque a comparação é
  inclusiva nas quatro bordas. Sem essa exclusão todo Retângulo rotulado se
  encontra, vira faixa no topo, e o caso `AoCentro` nunca acontece.
- Elemento de `Forma: Texto` **não conta como filho** — senão o próprio Rótulo
  faz todo Retângulo rotulado parecer cheio.
- Um Elemento de **geometria coincidente conta** como filho.
- A varredura olha todos os Elementos do Frame, de todas as Camadas, como a
  Elevação já faz.

Constantes, em `internal/resolve/rotulo.go`, em px do espaço do Frame:

    alturaDoRotulo  = 28.0   // altura da caixa reservada
    respiroDoRotulo = 6.0    // folga em cada ponta horizontal

- `A = min(alturaDoRotulo, retangulo.A)` — satura na altura do Retângulo.
- Faixa: `Y = retangulo.Y`. Centrada: `Y = retangulo.Y + (retangulo.A - A)/2`.
- `X = retangulo.X + respiroDoRotulo`,
  `L = max(0, retangulo.L - 2*respiroDoRotulo)`.
- O respiro é **geometria**, entregue já encolhida no Elemento de Texto. Nem
  `canvas.go` nem `svg.go` mudam por causa dele. Se o respiro morasse no
  rasterizador, a Prancheta desenharia encostado na borda e o WebP afastado — a
  segunda regra que a ADR-0002 proíbe.
- `fracaoDoRotulo = 0.45` fica **intocada**: 0.45 × 28 ≈ 12,6 px, a mesma ordem
  do corpo da Nota. É o que impede uma região de 400 px de virar fonte de 180.
- A seção do Rótulo na SKILL.md cita `alturaDoRotulo` por valor: o autor precisa
  saber quanto texto cabe.
- Não existe chave para forçar: nem `label-at:`, nem `section:`.

### 5b. Recorte do Rótulo na Prancheta

O raster já recorta o Rótulo na própria área (`mascaraDaArea`,
`internal/render/canvas.go:163`). O SVG **não**: o `<text>` de `svg.go` só é
recortado pelo `clipPath` do Frame, então um Rótulo largo demais vaza por cima
dos vizinhos na Prancheta e é cortado no WebP. Com Rótulo de Controle isso quase
nunca aparecia; com `label:` livre do autor vira o caso comum.

F9 fecha a divergência: **o Rótulo é recortado na sua própria área nos dois
desenhistas**. Tem teste, e é o que a DoD chama de "sem segunda regra".

### 6. `notes.Planeja(f scene.Frame, escala float64) *Plano`

O parâmetro `Modo` some junto com o tipo. `notes.Modo`, `notes.Margem`,
`notes.Flutuante` e `notes.Desligado` deixam de existir. Quem não quer Notas não
chama `Planeja`: `comandos.go` passa `nil` adiante, e os métodos de `*Plano` já
são seguros com receptor nulo — confira `Margens` e `Desenha`, não confie.

`CONTRACT.md` §6 publica a assinatura antiga e §7 publica `--notes margin|float|off`
com padrão `margin`. F10 atualiza os dois.

### 7. `notes.LimiteDaNota = 200`

Constante exportada, em **runas** (`utf8.RuneCountInString`), não em bytes: o
texto é português e um acento não pode custar dois caracteres do orçamento.
Constante no binário, pela mesma razão que o Tom e o corpo da fonte são.

**F10 só a declara.** O layout continua medindo o texto **inteiro**: não trunca,
não avisa, não muda nenhum retângulo do Plano por causa dela. Quem transforma
Nota longa em Erro é F11, e até lá uma Nota de 500 runas desenha como hoje.

Foi decidido assim porque a alternativa — "respeitar o teto no layout" — não tem
observável: ou a régua mede 500 runas e o balão estoura o Frame, ou mede 200 e o
desenho corta sem diagnosticar. O golden `flutuante.webp` congelaria a escolha
de quem implementasse.

### 8. Ordem total em `colhe`

A anti-colisão exige estabilidade que `colhe` (`internal/notes/notes.go`) não
entrega hoje: `sort.Slice` **não é estável** e o comparador não fecha ordem
total. Num Frame de 100 px, `rect {x:0,y:0,w:50,h:20, note:"A"}` e
`rect {x:25,y:0,w:25,h:20, note:"A"}` empatam nos três critérios; a ordem varia
entre execuções, `disporFlutuante` decide direita/esquerda a partir de
`esquerdaDoElemento`, e a imagem muda sem a geometria ter mudado.

Congelado: `colhe` passa a usar `sort.SliceStable` **e** ganha
`esquerdaDoElemento` como quarto critério, antes do texto. A ordem de declaração
vira o desempate final, e só chega a valer quando geometria e texto são
idênticos. O comentário de `colhe` registra a mudança.

### 9. `opcoes.notas bool`

Padrão `false`. `--notes` sem valor liga os balões flutuantes.

Regra de consumo do argumento seguinte, congelada porque `expandeIguais`
transforma `--notes=off` em dois argumentos indistinguíveis de `--notes off`:

> `--notes` espia o argumento seguinte. Se ele for **exatamente** `margin`,
> `float` ou `off`, consome-o e devolve o erro de uso de migração. Qualquer
> outra coisa não é consumida.

Mensagem literal:

    opção "--notes" não aceita mais valor: os modos margin, float e off acabaram; use "--notes" sozinho para os balões flutuantes, ou omita a opção para renderizar sem Notas

Sai por `usoInvalido`, código 1, como manda `CONTRACT.md` §7.
`draftboard render --notes doc.yaml` tem que funcionar: `doc.yaml` não é um dos
três nomes e portanto não é consumido.

### 10. `f6_test.go:219` chama `--notes off`

`TestToggleDizOEstadoPelaPosicao` roda `executa("render", "estados.yaml",
"--notes", "off")`. Depois do item 9 isso vira código 1 e o gate de F10 fica
vermelho. **F10 é dono do arquivo para esta correção**: remove a flag — o novo
padrão já é sem Nota. F9 não toca em `f6_test.go`.

### 11. Ordem do sufixo na linha do `inspect`

Congelada, com `rotulo=` por último:

    <caminho> <retangulo|circulo|texto> <X>,<Y> <L>x<A> tom=<T> elev=<E>[ round][ de=<componente>][ controle=<nome> <parâmetros>][ para=<frame>][ rotulo="<texto>"]

`skill_test.go:98` compara essa gramática byte a byte; ela é de F9.

### 13. Emenda: teto de comprimento do `label` do Retângulo

`schema.LimiteDoRotulo = 200`, em **runas**, conferido na decodificação do
`label` de um `rect`. Erro de decodificação, que aborta como os outros erros de
decodificação abortam.

Existe porque o texto do Rótulo era a única entrada do módulo sem teto, e o
rasterizador desenha glifo a glifo mesmo quando a máscara descarta tudo: um
`label` de 200 000 caracteres com `repeat: {n:20}` cabe em 195 KB de YAML e
custa 61 s de CPU, produzindo uma imagem onde nada aparece. Com 1 MB e `n:1000`,
projeta 15 horas. `LimiteDeClones`, `LimiteDeElementos`, `LimiteDeProfundidade` e
`limiteDaFonte` existem todos para fechar essa mesma amplificação.

200 é o mesmo número de `notes.LimiteDaNota`, contado do mesmo jeito, e pela
mesma razão: são as duas entradas de texto do autor, e o limite tem que ser
sabido enquanto se escreve. Um Rótulo é nome de bloco — 200 runas já é folgado.

Diferente do teto da Nota, este **é conferido e recusa**: nomear um bloco com 200
runas não é problema que a máquina conserte, e a decodificação é onde os erros
que exigem julgamento do autor já moram. Não confundir com o diagnóstico de
transbordo de F11, que é sobre caber na caixa, não sobre o tamanho da entrada.

A seção do Rótulo na SKILL.md cita o teto por valor.

### 12. Vocabulário: `scene.Texto` × o `_Avoid_` do Rótulo

"texto" é `_Avoid_` explícito de **Rótulo** no `CONTEXT.md`, e `scene.Texto` é o
nome de uma Forma desde a ADR-0001. A isenção é nominal e vale só para o
identificador da Forma e para o que a árvore imprime. **Em prosa, comentário e
mensagem, o conceito se chama Rótulo.** Ninguém renomeia `scene.Texto`.

## Definição de pronto

### F9 — Rótulo no Retângulo

Os itens de geometria são afirmações de teste sobre `resolve.Arquivo`, porque o
Elemento de Texto é `Interno` e o `inspect` o esconde de propósito.

- [ ] `go build ./... && go vet ./... && gofmt -l . && go test ./...` passa.
- [ ] `inspect` de um `rect` com `label: "Grade"` imprime `rotulo="Grade"` ao
      final da linha do Retângulo, e **não** imprime a peça de Texto.
- [ ] `f9_test.go`: Retângulo **com filho** → o Elemento de Texto tem
      `Alinhamento == scene.AEsquerda` e `Y == retangulo.Y`.
- [ ] `f9_test.go`: Retângulo **isolado** → `Alinhamento == scene.AoCentro`.
      Este caso prova a exclusão do próprio Retângulo na varredura.
- [ ] `f9_test.go`: Retângulo de 400 px de altura → o Elemento de Texto tem
      `A == 28`, e não 400.
- [ ] `f9_test.go`: Retângulo de 10 px de altura → `A == 10`.
- [ ] `f9_test.go`: Retângulo estreito demais para o respiro → `L == 0`, sem
      pânico e sem aviso.
- [ ] `f9_test.go`: o Caminho do Rótulo é `<caminho do Retângulo>/rotulo`.
- [ ] `f9_test.go`: Elevação e Tom do Rótulo saem do Retângulo que o carrega,
      inclusive quando o Retângulo tem filhos pintados depois.
- [ ] `f9_test.go`: `rect + label + repeat` debita o Rótulo no
      `LimiteDeElementos`.
- [ ] `label:` em `circle`, `use` ou `slot` dá a mensagem literal do item 1, com
      arquivo e localização. `label:` em `control` continua funcionando.
- [ ] `board` recorta o Rótulo na área dele, como o WebP faz.
- [ ] Todo Elemento resolvido carrega `Local` não vazio, provado por teste.
- [ ] A tabela de nós da SKILL.md ganha `label` no `rect`, e a seção do Rótulo
      cita `alturaDoRotulo` por valor.

### F10 — Aposentadoria do Margem

- [ ] O gate passa.
- [ ] `internal/notes/margem.go` não existe; `notes.Modo` não existe;
      `grep -rn "notes\.Margem" --include='*.go' .` não acha nada fora de
      `.claude/worktrees/`.
- [ ] `render doc.yaml` não desenha Nota; a imagem tem as dimensões do Frame.
- [ ] `render doc.yaml --notes` desenha os balões flutuantes.
- [ ] `render doc.yaml --notes float|margin|off` sai 1 com a mensagem literal do
      item 9.
- [ ] `render --notes doc.yaml` funciona: o arquivo não é consumido.
- [ ] Dois, três e dez Elementos anotados com âncoras vizinhas produzem balões
      que **não** se intersectam, provado sobre os retângulos do Plano.
- [ ] `colhe` usa `sort.SliceStable` e tem `esquerdaDoElemento` como critério; o
      caso de empate do item 8 tem teste.
- [ ] Embaralhar a ordem de declaração sem mexer na geometria dá a mesma imagem.
- [ ] `testdata/f4/margem.webp` removido; os outros dois regerados e **olhados**,
      com o que mudou descrito no relatório.
- [ ] `notes.LimiteDaNota == 200`, em runas, e nada no layout a consulta.
- [ ] `CONTRACT.md` §6 e §7 atualizados.
- [ ] A SKILL.md não menciona mais os três modos nem o Chrome.

### F11 — Diagnóstico por corrigibilidade e `inspect --fix`

Roda **serial**, depois de F9 e F10 integrados (divisibilidade 2). Dono único de
tudo que toca. Esta é a segunda versão, escrita depois da revisão de contrato:
oito blockers, e o que os fecha está marcado com **(rev)**.

#### 14. `scene.Elemento.Espaco`

    // Espaco é a caixa, em px, do espaço local em que as porcentagens do nó
    // foram projetadas — o `espaco` de achatamento.go. Preenchido em todo
    // Elemento.
    type Espaco struct{ X, Y, L, A float64 }

**(rev)** É a caixa inteira, e não só a largura: o eixo vertical projeta por
`esp.A` e o Círculo projeta o diâmetro por `esp.L` nos dois eixos. Publicar só
`L` obrigaria a renomear um campo público de `scene` no primeiro pedido de
sugerir `h:`.

É geometria, não diagnóstico: nasce em `resolve/` sem violar a proibição do item
15, porque não decide nada sobre o que cabe. Existe porque a única forma de
dizer «o `w` mínimo é 47%» é desfazer a projeção, e só quem projetou conhece a
base. O Rótulo herda a do Retângulo que o carrega.

`acrescenta`, `emite` e `rotuloDoRetangulo` não recebem `espaco` hoje: as três
mudam de assinatura.

#### 15. `internal/diag` — onde o diagnóstico nasce

    package diag
    // Confere mede o Documento já resolvido e devolve o que não cabe, separado
    // por corrigibilidade. alargavel responde se a máquina consegue consertar o
    // nó sozinha, e por que não quando não consegue; nil significa que nada é
    // alargável e tudo que não couber é Erro.
    func Confere(arquivo string, doc *scene.Documento,
        alargavel func(local string) (bool, string)) ([]scene.Aviso, []*scene.Erro)

`diag` importa `render` (para medir glifos) e `notes` (para `LimiteDaNota`), e
**não importa `fix`**: a corrigibilidade chega como predicado, para que `diag`
seja testável com um predicado de mentira. Quem liga os dois é `comandos.go`.

**Não pode nascer em `internal/resolve/`**: medir texto exige a fonte, e a
resolução é o único lugar do sistema que hoje calcula geometria sem depender do
freetype. Quem inverte isso paga com o pacote inteiro.

A régua é uma só por chamada, `render.NewCanvas(1, 1, 0, 0, 0, 0, 1)`, criada
antes do laço: cada Canvas carrega o seu cache de faces, e um por Elemento em
Documento de 10.000 Elementos é 10.000 caches. O diagnóstico é do Documento, não
da invocação: `--scale` não muda o que cabe.

**(rev)** A medida, em código real, é

    largura, _ := regua.MedeTexto(e.Rotulo, render.TamanhoDoRotulo(e.A))
    cabe := largura <= e.L

`MedeTexto` é método de `*Canvas` e devolve dois valores.

**(rev)** Os Elementos medidos são exatamente os de `Forma == scene.Texto &&
Controle == ""` — a mesma condição de `ehRotuloDeRetangulo`. O Rótulo de
Controle fica **fora do escopo de F11**: a caixa dele é escolha do catálogo, o
YAML do autor não tem `rect.w` para alargar (tem `box.w`, que dimensiona o
Controle inteiro), e classificá-lo junto travaria o `--fix` do Documento
inteiro. Registrado como pendência.

**(rev)** F11 **não diagnostica transbordo vertical**, e isso é decisão, não
esquecimento: `posiciona` satura a altura da área em `min(alturaDoRotulo, A)` e
`TamanhoDoRotulo` é `0,45 * A`, então a caixa de linha fica em torno de `0,52 *
A` e cabe por construção. Não existe Rótulo de Retângulo que estoure na
vertical.

#### 16. As categorias, decididas por corrigibilidade real

**(rev)** A categoria não é escolhida por onde o Retângulo mora, e sim pelo que
a máquina de fato consegue consertar. É **Aviso** quando **todas** valem:

- `Origem == ""` — o Retângulo foi escrito direto no Documento;
- `e.Espaco.L` é finita e maior que zero;
- `alargavel(e.Local)` responde `true` — o `Local` resolve, o nó é `rect`, tem
  `w` declarado como escalar simples, e não está dentro de uma Repetição.

Qualquer outra combinação é **Erro**. Sem essa inversão, `diag` emitiria Aviso
para nó que `fix` não sabe endereçar, e um lote com um único nó assim não
escreveria nenhuma das outras correções.

Cada exclusão tem uma razão que a mensagem carrega:

| Situação | Categoria | Por quê |
| --- | --- | --- |
| Cabe tudo acima | **Aviso** | `--fix` alarga o `w` e acabou. |
| `Origem != ""` | **Erro** | Alargar no Componente muda **todas** as Instâncias. |
| Nó sem `w` declarado | **Erro** | Não há valor a trocar; escrever um `w` novo é escolher a largura por conta. |
| Nó dentro de Repetição | **Erro** | O `w` é o passo da Repetição (`tamanhoNoEixo`): alargar reposiciona todos os clones. |
| `Espaco.L` zero ou não finita | **Erro** | A porcentagem não tem base; a resolução já avisou da área zero. |
| Nota acima de `notes.LimiteDaNota` runas | **Erro** | Não há largura a alargar. |

O Rótulo largo demais é **cortado, nunca quebrado** — isto já é o que
`desenhaRotulo` faz e F11 não muda. O diagnóstico existe justamente porque o
corte é silencioso.

**(rev)** A largura necessária é a do **Retângulo**, não a da área do Rótulo: a
resolução já descontou o respiro (item 5). O par se acha pelo Caminho — o Rótulo
é `<caminho do dono>/rotulo`, com `resolve.SegmentoDoRotulo` exportado para que
não haja duas grafias do sufixo:

    necessarioNoRetangulo := largura + (dono.L - rotulo.L)
    w := math.Ceil(100 * necessarioNoRetangulo / dono.Espaco.L)

Arredondado **para cima**: um `w` que arredonda para baixo continua cortando, e
um Aviso que sugere um conserto que não conserta é pior que nenhum. Sem o
respiro de volta, o `w` sai sistematicamente `2 * respiroDoRotulo` px curto — 34
onde precisava de 37, medido na revisão.

**(rev)** Mensagens congeladas, literais. `%s` de `w` é
`strconv.FormatFloat(v, 'g', -1, 64)`:

    o Rótulo %q não cabe no Retângulo: precisa de %.0f px e tem %.0f; use w: %s
    o Rótulo %q não cabe no Retângulo: precisa de %.0f px e tem %.0f; %s
    a Nota tem %d runas, acima do limite de %d; encurte-a

O `%s` final da segunda é a razão, uma destas, também literais:

    o Retângulo vem de um Componente, e alargá-lo lá muda todas as Instâncias
    o Retângulo não declara "w"
    o Retângulo está dentro de uma Repetição, e alargá-lo reposiciona os clones
    o espaço do Retângulo tem largura zero

Nenhuma mensagem diz `caixa` (é `_Avoid_` de Retângulo) nem `texto` (é `_Avoid_`
de Rótulo e de Nota). A isenção nominal do item 12 vale só para o identificador
`scene.Texto`.

#### 17. O Erro de F11 não aborta

Os Erros de `diag` são impressos como erro, o processo sai com 1, **e o comando
faz o seu trabalho assim mesmo**: `render` escreve as imagens, `inspect` imprime
a árvore, `board` escreve a Prancheta.

Os Erros antigos — decode, Componente ausente, teto de área, teto de nós —
continuam abortando. A diferença é de natureza, e é a mesma que o `CONTEXT.md`
já escreve na verbete de Erro: aqueles impedem de saber o que desenhar, este
descreve um desenho que já existe e está errado.

Ordem de saída no stderr: Avisos da resolução, Avisos de `diag`, Erros de
`diag`. No formato de `scene.Aviso`/`scene.Erro` que já existe, com `Arquivo` e
`Local` — é `Local` que o `--fix` endereça.

Vale nos quatro comandos: `render`, `inspect`, `validate`, `board`. **(rev)**
Inclusive o Erro da Nota longa em `render` sem `--notes`, que não desenha Nota
nenhuma: o artefato diagnosticado é o **Documento**, não a invocação, pela mesma
razão que a régua é de escala 1. Um Documento não fica correto por ser
renderizado com a opção que esconde o defeito.

#### 18. `internal/fix` — a cirurgia de bytes

    package fix
    // Abre lê o arquivo YAML cru e indexa os nós endereçáveis.
    func Abre(arquivo string) (*Arquivo, error)
    // Alargavel diz se o Local endereça um `rect` com `w` alargável pela
    // máquina, e a razão quando não.
    func (a *Arquivo) Alargavel(local string) (bool, string)
    // Alarga registra a troca do `w` de um nó. Nada é escrito até Grava.
    func (a *Arquivo) Alarga(local string, w float64) error
    // Grava escreve todas as trocas de uma vez e devolve o que trocou, na ordem
    // em que foram registradas.
    func (a *Arquivo) Grava() ([]Troca, error)
    type Troca struct{ Local string; De, Para float64 }

**(rev)** É um punhado com estado, e não uma lista de correções, porque a mesma
leitura do arquivo serve ao predicado do item 16 e à cirurgia: parsear duas
vezes abriria a janela para o arquivo mudar entre uma e outra.

`Alarga` recusa `w` não finito ou não positivo. `Grava` num punhado vazio não
toca o arquivo.

**(rev)** Gramática dos segmentos que `Alargavel` aceita, e ela é fechada:

    frames[i] .layers[i] .elements[i] .default[i] .slots.<nome>

`Local` que contenha `" -> "` é de Componente e nunca chega aqui — `Origem !=
""` já o barrou no item 16. O segmento `.slots.<nome>` é ambíguo quando o nome
do Slot tem ponto: casa-se o **nome mais longo** que exista de fato entre as
chaves do mapa `slots` daquele nó. Local que não resolva devolve
`(false, ...)`, nunca pânico.

**(rev)** A posição vem de `Line`/`Column` do nó do **valor**, e `Column` conta
**runas, não bytes**: converter percorrendo a linha com `utf8.DecodeRune` até a
coluna pedida. Num projeto inteiro em português, um `label` acentuado na mesma
linha do `w` desloca o splice e corrompe o arquivo do autor.

**(rev)** Todos os deslocamentos são calculados contra o buffer **original** e
aplicados em ordem **decrescente** de (linha, coluna), para que uma troca não
empurre a seguinte. A ordem devolvida em `Troca` continua sendo a de registro.

Só os bytes do escalar são trocados. Não reserializa o documento: reserializar
perde comentários, ordem e estilo de bloco, e o autor não pediu para reformatar
o arquivo — pediu um número maior.

`schema.No` **não** ganha posição. A posição é conhecimento de quem faz
cirurgia, e mantê-la fora do schema é o que impede o resto do sistema de começar
a raciocinar sobre linhas de arquivo.

**(rev)** Escrita: `Grava` monta o buffer inteiro em memória, resolve symlink
até o alvo real, escreve em arquivo temporário **no mesmo diretório**, copia o
modo do original e faz `os.Rename`. Nada de truncar em cima: um processo morto
no meio levaria o Documento junto. Se o arquivo mudou de tamanho ou de mtime
desde `Abre`, `Grava` recusa com

    o arquivo mudou no disco desde a leitura; rode o comando de novo

Erro de permissão sai como `*scene.Erro` do domínio, não como o erro cru do
sistema operacional.

#### 19. `inspect --fix`

Conserta e imprime a árvore já corrigida numa chamada só.

- Só os **Avisos** de `diag` viram troca — Erro nunca, por definição da
  categoria (item 16). Componente jamais é tocado.
- Só o `w` é alargado. Nunca `h`, nunca `x`, nunca `y`, nunca o Rótulo.
- Depois de gravar, o Documento é **resolvido de novo** e é essa árvore que vai
  para o stdout. Imprimir a árvore velha diria ao agente que o conserto não
  aconteceu.
- As linhas de troca vão para o **stderr**, uma por conserto, antes de tudo:

      frames[0].layers[0].elements[2]: w 20 → 47

  Os dois números com `strconv.FormatFloat(v, 'g', -1, 64)`, e `De` é o valor
  **decodificado**, não o texto original: um `w: 2e1` imprime `20`.
- **(rev)** A segunda resolução **imprime tudo** — Avisos dela e diagnóstico de
  `diag` — depois das linhas de troca. Alargar um Retângulo encostado na borda
  direita produz um Aviso novo de "fora do Frame", e escondê-lo faria o
  `inspect --fix` mostrar menos que o `inspect` puro.
- Sem nada a consertar, `--fix` não escreve no arquivo e se comporta como
  `inspect` puro.
- `--fix` só existe em `inspect`. Em qualquer outro comando é erro de uso.
- Código de saída depois do conserto: 0 se não sobrou Erro, 1 se sobrou.

#### 20. Superfícies que seguem F11

- `internal/skill/SKILL.md`: seção de diagnóstico (as categorias e o que cada
  uma quer do autor), `inspect --fix` no bloco de uso, e a regra de que o Rótulo
  é cortado e nunca quebrado.
- `CONTRACT.md`: `diag`, `fix`, `--fix`, o §7 do `--notes` com a forma `=`
  (pendência que F10 deixou), e a nova categoria de Erro que não aborta.
- `docs/PRD.md`: aposentar o modo margem e o Chrome do texto (itens 32, 36, 37,
  §356-366, §452 e o glossário do §489), e registrar o diagnóstico por
  corrigibilidade.
- **(rev)** Comentários que ainda falam de Chrome ou de modo margem fora de
  `internal/render/`: `f6_test.go:46`, `main_test.go:455`, `main_test.go:531`.
  Mais `internal/render/texto.go:34`, que é o único de dentro do pacote que F11
  arruma. `main_test.go` entra na lista de caminhos de F11, só para comentário.

**Fora do escopo**, e registrado em `docs/PENDENCIAS.md`: as quatro margens de
`render.NewCanvas`/`DesenhaFrame`, mortas em produção desde F10, e o resto do
pacote `render` ainda documentado em termos de Chrome. E o Rótulo de Controle
largo demais, que F11 não diagnostica (item 15).

#### Definição de pronto — F11

- [ ] `go run . render <doc com Rótulo largo demais>` escreve a imagem, imprime
      o Aviso com o `w` mínimo e sai com 0.
- [ ] **O `w` sugerido conserta de primeira**: aplicá-lo e rodar de novo não
      produz Aviso. `go run . inspect --fix` duas vezes seguidas: a segunda não
      imprime troca nenhuma e não muda um byte do arquivo.
- [ ] O mesmo Rótulo dentro de um Componente escreve a imagem, imprime o Erro
      com a razão do Componente, e sai com **1**.
- [ ] **(rev)** Retângulo rotulado **sem `w` declarado**, e outro **dentro de uma
      Repetição**: Erro, cada um com a sua razão. Um Documento que misture os
      dois com cinco Retângulos consertáveis conserta os cinco e sai com 1.
- [ ] **(rev)** Slot de `box: {w: 0}` preenchido com Retângulo rotulado: Erro de
      largura zero, e nenhum `+Inf` em mensagem nem em arquivo.
- [ ] `go run . inspect --fix <doc>` reescreve o `w`, imprime
      `frames[0]...: w 20 → 47` no stderr e a árvore **corrigida** no stdout.
- [ ] **(rev)** `--fix` num Documento cujo único diagnóstico é Erro não escreve
      byte nenhum e sai com 1.
- [ ] **(rev)** Sobrevivem à cirurgia: comentário na mesma linha do `w`; `label`
      **acentuado** na mesma linha do `w`; dois `w` na mesma linha; `w: 2e1` em
      estilo de fluxo; `w` em estilo de bloco.
- [ ] **(rev)** `--fix` num arquivo somente-leitura falha com `*scene.Erro` do
      domínio e deixa o arquivo intacto.
- [ ] **(rev)** Consertar um Retângulo encostado na borda direita imprime o
      Aviso novo de "fora do Frame" na mesma chamada, e o código de saída
      continua 0.
- [ ] Nota de 201 runas: Erro, imagem escrita, saída 1. 200 runas: nada. A conta
      é em runas — 200 runas acentuadas não disparam.
- [ ] **(rev)** Os quatro comandos diagnosticam, e os quatro códigos de saída
      são conferidos: `render`, `inspect`, `validate` e `board` saem 1 com um
      Erro de `diag`, tendo escrito o que tinham para escrever.
- [ ] `--fix` em `render`, `board` ou `validate` é erro de uso.
- [ ] O diagnóstico não muda com `--scale 1` e `--scale 8`.
- [ ] SKILL.md, CONTRACT.md e PRD.md atualizados, e

          grep -rn Chrome --include='*.go' . | grep -v '^\./\.claude/' \
            | grep -v '^\./internal/render/' | grep -v TomChrome

      não acha nada.

## Gate

Local, sem CI de teste:

    go build ./... && go vet ./... && gofmt -l . && go test ./...

Baseline em `cb909b9`: verde, 298 testes. Depois de F9 e F10 integrados: 350.
