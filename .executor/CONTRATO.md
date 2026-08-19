# Contrato — Rótulo no Retângulo (F9, F10, F11)

Congelado pelo Executor antes do primeiro agente. Qualquer mudança passa por ele.

## Fatiamento e divisibilidade

| Feature | Escopo (itens do handoff) |
| --- | --- |
| **F9 — Rótulo no Retângulo** | 1, 2, 5, e a parte de `svg.go` + tabela de nós da SKILL.md do item 8 |
| **F10 — Aposentadoria do Margem** | 6, 7, e a seção CLI da SKILL.md do item 8 |
| **F11 — Diagnóstico e `--fix`** | 3, 4 |

Divisibilidade F9 × F10: contrato explícito 2, fronteira de arquivos 1
(`internal/skill/SKILL.md` é comum, em seções disjuntas), independência de dados 2,
ordem livre 2, verificação isolada 2. **Soma 9 → paralelo**, cada uma em worktree
própria, porque as duas editam a SKILL.md.

Divisibilidade F11 × (F9, F10): contrato 1, fronteira 0 (`comandos.go`,
`internal/resolve/`, `internal/scene/` são comuns), dados 1, ordem 0, verificação 0.
**Soma 2 → serial**, depois das duas.

## Propriedade de arquivos

| Arquivo | F9 | F10 | F11 |
| --- | --- | --- | --- |
| `internal/schema/` | ✅ | — | ✅ (posição de bytes) |
| `internal/resolve/` | ✅ | — | — |
| `internal/scene/scene.go` | ✅ | comentário de `TomChrome` | — |
| `internal/render/texto.go`, `canvas.go` | ✅ | — | ✅ (arquivo novo) |
| `internal/inspect/` | ✅ | — | ✅ |
| `internal/board/svg.go` | ✅ | — | — |
| `internal/notes/` | — | ✅ | — |
| `opcoes.go` | — | ✅ | ✅ |
| `comandos.go` | — | ✅ | ✅ |
| `main_test.go` | — | ✅ | ✅ |
| `f9_test.go` (novo) | ✅ | — | — |
| `f11_test.go` (novo) | — | — | ✅ |
| `testdata/f4/` | — | ✅ | — |
| `testdata/f9/` (novo) | ✅ | — | — |
| `internal/skill/SKILL.md` | tabela de nós, seção do Rótulo | seção CLI, `--notes` | seção de diagnóstico |
| `CONTEXT.md`, `docs/adr/` | congelados — ninguém edita | | |

F9 **não** escreve em `main_test.go`: seus testes de ponta a ponta vão para
`f9_test.go`, seguindo `f2_test.go`/`f6_test.go`/`f8_test.go`.

## Interfaces congeladas

### 1. `schema.No.Rotulo string`

O `label:` declarado num nó `rect`. Vazio quando ausente. Só `TipoRetangulo` o
preenche — o Controle continua guardando o seu em `no.Controle.Rotulo`.
`circle`, `use` e `slot` recusam `label` com erro no padrão de `round`/`to`.

### 2. `scene.Elemento.Local string`

O caminho de chaves YAML do nó que originou o Elemento, já atravessando a cadeia
de Componentes — exatamente o mesmo valor que `resolve` passa hoje para
`r.aviso`/`r.erro` (`ctx.prefixo + no.Local`). Preenchido em **todo** Elemento
emitido, inclusive nas peças de Controle e no Rótulo materializado.

Produzido por: **F9**. Consumido por: **F11**, que o usa para posicionar o Aviso
e para achar a linha do arquivo no `--fix`.

### 3. `scene.Elemento.Rotulo` na cabeça do Retângulo

O campo já existe. Passa a ser preenchido **também** no Elemento de
`Forma: Retangulo` que declarou `label:`, para que o `inspect` imprima sem
depender da peça interna. Consequência: `Rotulo != ""` deixa de implicar
`Forma == Texto`; o comentário do campo tem que dizer isso.

Quem desenha continua despachando por `Forma`: só `scene.Texto` vira glifo, no
`canvas.go` e no `svg.go`. A cabeça do Retângulo nunca desenha o Rótulo duas vezes.

### 4. Materialização do Rótulo — adjacência

O Rótulo do Retângulo é um `scene.Elemento` de `Forma: Texto`, `Interno: true`,
`Controle: ""`, emitido pelo achatamento **imediatamente depois** do seu
Retângulo, na mesma Camada, com a geometria do Retângulo como provisória.

Uma passagem posterior — antes de `atribuiElevacao` — corrige geometria e
Alinhamento a partir da contenção. Ela identifica o dono pela **adjacência**:
o Retângulo é o Elemento anterior na fatia. A invariante fica documentada nos
dois pontos e tem teste próprio.

Imediatamente depois, e não no fim da Camada, porque a Superfície do Rótulo tem
que ser o Retângulo que o carrega: é dele que saem a Elevação e o Tom que dão
contraste ao texto. Empilhado no fim, um filho qualquer viraria a Superfície.

### 5. Regra de posição

Retângulo **contém outro Elemento** → faixa no topo, `scene.AEsquerda`.
Retângulo **vazio** → caixa centrada na vertical, `scene.AoCentro`.

- A contenção é `contemGeometricamente`, a mesma de `elevacao.go`. Não se
  duplica a regra: a passagem nova chama a função existente.
- Elementos de `Forma: Texto` **não contam como filho** — senão o próprio
  Rótulo faria todo Retângulo rotulado parecer cheio.
- A contenção olha todos os Elementos do Frame, de todas as Camadas, como a
  Elevação já faz.
- A altura da caixa reservada é **constante em px do espaço do Frame**,
  saturada na altura do Retângulo. `fracaoDoRotulo = 0.45` fica intocada.
- Não existe `label-at:` nem `section:`.

### 6. `notes.Planeja(f scene.Frame, escala float64) *Plano`

O parâmetro `Modo` some junto com o tipo. `notes.Modo`, `notes.Margem`,
`notes.Flutuante` e `notes.Desligado` deixam de existir. Quem não quer Notas não
chama `Planeja`: `comandos.go` passa `nil`, e os métodos de `*Plano` já são
seguros com receptor nulo.

Produzido por: **F10**.

### 7. `notes.LimiteDaNota = 200`

Constante exportada, em **runas** (`utf8.RuneCountInString`), não em bytes: o
texto é português e um acento não pode custar dois caracteres do orçamento.

F10 **exporta e respeita** o teto no layout, mas **não emite diagnóstico** e
**não trunca**. Quem transforma a Nota longa em Erro é F11.

Produzido por: **F10**. Consumido por: **F11**.

### 8. `opcoes.notas bool`

`--notes` sem valor liga os balões flutuantes. Padrão `false`.
`--notes float`, `--notes margin` e `--notes off` viram erro de uso, com
mensagem que diz o que mudou.

Cuidado com `expandeIguais`: `--notes=x` chega como dois argumentos, e a
booleana não pode consumir o posicional seguinte como valor.

Produzido por: **F10**.

## Definição de pronto

### F9 — Rótulo no Retângulo

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` passa.
- `draftboard inspect` de um `rect` com `label: "Grade"` e filhos imprime
  `... rotulo="Grade"` na linha do Retângulo, e **não** imprime a peça de Texto.
- O mesmo Documento renderiza uma imagem com o texto numa faixa no topo à
  esquerda do Retângulo; sem filhos, centralizado.
- Um Retângulo de 400 px de altura com Rótulo produz fonte da ordem de 12 px,
  não de 180 px.
- Um Retângulo de 10 px de altura com Rótulo não produz faixa maior que ele.
- `label:` em `circle`, `use` ou `slot` é erro de decodificação com o arquivo e
  a localização, no padrão de `round`.
- `draftboard board` desenha o mesmo Rótulo na Prancheta, sem uma segunda regra
  de tamanho ou de posição.
- Todo Elemento resolvido carrega `Local` não vazio.

### F10 — Aposentadoria do Margem

- O gate passa.
- `internal/notes/margem.go` não existe. `notes.Modo` não existe. Nada no repo
  cita `notes.Margem`.
- `draftboard render doc.yaml` **não** desenha Nota nenhuma; a imagem tem as
  dimensões do Frame.
- `draftboard render doc.yaml --notes` desenha os balões flutuantes.
- `draftboard render doc.yaml --notes float` sai com código 1 e mensagem de uso
  dizendo que os modos acabaram.
- Dois Elementos anotados com âncoras vizinhas produzem balões que **não** se
  sobrepõem, provado por teste sobre os retângulos do Plano.
- `testdata/f4/margem.webp` some; `flutuante.webp` e `desligado.webp` são
  regerados e **olhados** antes de aceitos.
- `notes.LimiteDaNota` existe, vale 200 e conta runas.

### F11 — Diagnóstico e `--fix`

Definida por escrito quando F9 e F10 estiverem integradas.

## Gate

Local, sem CI de teste:

    go build ./... && go vet ./... && gofmt -l . && go test ./...

Baseline em `cb909b9`: verde, 298 testes.
