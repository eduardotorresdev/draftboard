# CONTRATO — draftboard

Congelado. Mudança aqui volta ao orquestrador, nunca é negociada entre implementadores.
Vocabulário obrigatório: o de `CONTEXT.md`, em código, mensagens de erro e skill.

## 0. Módulo e árvore

`module github.com/eduardotorresdev/draftboard`, Go 1.26, **sem cgo**.

`go.mod` e `go.sum` **já estão prontos e são congelados**. Não rode `go mod tidy`,
não adicione dependência. Isso vale inclusive para F7: `internal/update` usa só
`net/http`, `encoding/json`, `archive/tar`, `compress/gzip`, `crypto/sha256`, `os`,
`os/exec`, `runtime`, `strconv` e `strings` — sem semver de terceiros, sem isatty,
sem biblioteca de self-update. Deps disponíveis: `gopkg.in/yaml.v3`, `github.com/fogleman/gg`,
`github.com/HugoSmits86/nativewebp`, `golang.org/x/image v0.24.0` (inclui
`font/gofont/goregular`) e `github.com/golang/freetype` (carregador de fonte).

Dono de cada diretório (ninguém escreve fora do seu):

| Diretório | Dono |
| --- | --- |
| `internal/scene/` | **orquestrador — congelado, ninguém edita** |
| `internal/controls/` | catálogo de Controles |
| `internal/schema/`, `internal/resolve/`, `internal/inspect/` | F1, depois F2 |
| `internal/render/` | F3 |
| `internal/notes/` | F4 |
| `internal/skill/`, `SKILL.md` | F5; **F7 acrescenta apenas `EstaSincronizada`** |
| `internal/update/`, `.github/workflows/` | F7 |
| `internal/board/` | F8 |
| `main.go` | F1 (dispatch + `render`/`inspect`/`validate`); F5 acrescenta só o caso `skill`; F7 acrescenta os casos `version` e `update` e o parâmetro `stdin` de `run`; F8 acrescenta só o caso `board` |
| `testdata/` | subdiretório por funcionalidade: `testdata/f1/`, `testdata/f2/`, ... |

## 1. `internal/scene` — o contrato central

Já implementado e congelado. Leia `internal/scene/scene.go`. Resumo:

```go
type Tom int                    // 100 (quase branco) .. 900 (quase preto)
const TomFrame Tom = 100        // fundo fixo de todo Frame
const TomChrome Tom = 900       // extremo reservado ao Chrome
func TomDaElevacao(elevacao int) Tom
func (t Tom) Cinza() uint8

type Forma int; const (Retangulo Forma = iota; Circulo; Texto)
type Alinhamento int; const (AoCentro Alinhamento = iota; AEsquerda)

type Elemento struct {
    Caminho     string   // caminho estável na árvore; último segmento = id se houver
    ID          string   // "" quando não declarado
    Forma       Forma
    X, Y, L, A  float64  // px absolutos no Frame; âncora = canto superior esquerdo, sempre
    Arredondado bool
    Elevacao    int
    Tom         Tom
    Origem      string   // caminho relativo do Componente de origem; "" se inline
    Nota        string   // "" se não há
    Destino     string   // Ligação: nome do Frame de destino; "" se não há (ver §11)
    Rotulo      string   // texto do Elemento de Forma Texto; "" nas demais Formas
    Controle    string   // nome de catálogo do Controle de origem; "" se não veio de um
    Detalhe     string   // parâmetros do Controle formatados para o inspect; só na cabeça
    Interno     bool     // peça de dentro de um Controle: existe no desenho, não na árvore
    Alinhamento Alinhamento // posição do Rotulo na sua área; só vale em Forma Texto
}
type Camada struct { Nome string; Elementos []Elemento }
type Frame struct { Nome string; L, A int; Camadas []Camada }
type Documento struct { Nome string; Frames []Frame }
type Aviso struct { Arquivo, Local, Msg string }
```

`scene.Documento` é achatado: Instâncias, Slots, Repetições e Controles **já não
existem** nele — o Controle deixou os Elementos que materializou.

**Adendo 1c (Forma Texto).** A Forma `Texto` é produzida apenas pela resolução de um
Controle; não há nó de texto no YAML. `X, Y, L, A` são a **área reservada** ao Rótulo,
nunca a caixa das glifas: medir texto exige a fonte, e a fonte não entra na resolução.
`Forma.String()` devolve `retangulo`, `circulo` ou `texto`, por `switch` — Forma nova
não pode se disfarçar de Retângulo na árvore. Ver
`docs/adr/0001-rotulo-de-controle-desenha-texto-no-frame.md`.

### 1b. Entrada da resolução (dono F1)

```go
package resolve

// Arquivo lê o Documento no caminho dado, valida, achata Instâncias, Slots e
// Repetições, e calcula Elevação e Tom de cada Elemento.
// Devolve erro do tipo *scene.Erro quando a resolução falha.
func Arquivo(caminho string) (*scene.Documento, []scene.Aviso, error)
```

`scene.Erro` (congelado, já em `internal/scene`): campos `Arquivo`, `Local`, `Msg`;
`Error()` devolve `"<arquivo>: <local>: <msg>"`. A CLI prefixa `erro: ` ao imprimir.

## 2. Schema YAML (congelado)

Tipo do arquivo inferido pelo conteúdo: tem `frames` → **Documento**; senão → **Componente**.
Toda chave desconhecida em qualquer nível é **erro** com sugestão da chave válida mais próxima.

### Documento

```yaml
frames:
  - name: home
    w: 1280
    h: 800
    layers:
      - name: conteudo
        elements: [ ...nós de elemento... ]
```

### Componente (espaço local 0..100 em ambos os eixos, sem `frames`)

```yaml
elements: [ ...nós de elemento... ]
```

### Nó de elemento

Exatamente **uma** das chaves discriminantes: `rect`, `circle`, `use`, `slot`.

```yaml
- rect: {x: 5, y: 5, w: 90, h: 20}       # x,y,w,h em % do eixo correspondente
  round: true                             # opcional, só em rect
  id: header                              # opcional
  note: "Cabeçalho fixo"                  # opcional
  to: dashboard                           # opcional; Ligação, só em Documento (ver §11)
  repeat: {n: 3, axis: y, gap: 2}         # opcional

- circle: {x: 8, y: 8, d: 4}              # d em % da LARGURA do espaço, em ambos os eixos

- use: "./card.yaml"                      # Instância; caminho relativo AO ARQUIVO QUE REFERENCIA
  box: {x: 5, y: 30, w: 40, h: 30}        # caixa no espaço do pai; obrigatória
  slots:                                   # opcional
    body: {use: "./avatar.yaml"}
    foot: {elements: [ {rect: {x:0,y:0,w:100,h:100}} ]}

- slot: "body"                            # declaração de Slot, só em Componente
  box: {x: 10, y: 40, w: 80, h: 50}
  default: [ ...nós de elemento... ]      # opcional
```

Chaves permitidas por nó: `rect|circle|use|slot` + `round`, `id`, `note`, `repeat`,
`box` (só `use`/`slot`), `slots` (só `use`), `default` (só `slot`).

### Conversões de geometria (congeladas)

Espaço do Frame (px `FL`×`FA`) — todo valor é % do eixo:
`X = x/100*FL`, `Y = y/100*FA`, `L = w/100*FL`, `A = h/100*FA`.
Círculo: `L = A = d/100*FL` (largura em ambos os eixos, para nunca virar elipse).

Espaço local de Componente/Slot mapeado numa caixa px `(bx,by,bl,ba)`:
`X = bx + x/100*bl`, `Y = by + y/100*ba`, `L = w/100*bl`, `A = h/100*ba`.
Círculo: `L = A = d/100*bl`.

`repeat`: `n` clones (n≥1), `axis` ∈ {`x`,`y`}, `gap` em % do eixo do espaço local.
Clone `i` desloca `i * (tamanho + gap)` no eixo, onde `tamanho` é `w`/`h` do rect,
`d` do círculo, ou `box.w`/`box.h` da Instância — tudo em unidades do espaço local,
antes da conversão para px. Repetição materializa clones no achatamento.

### 2c. Controles (adendo, congelado)

O nó `control` é a quinta chave discriminante. Ele é **fechado**: recusa
`round`, `slots` e `default`, exige `box`, e aceita `id`, `note` e `repeat` como
qualquer outro nó.

Os campos de Controle (`label`, `items`, `active`, `value`) são chaves válidas
de nó para efeito de sugestão, e a permissão real é conferida contra o Controle
declarado: `label` é legítimo num `button` e ilegítimo num `slider`.

| Controle | Campos | Padrões |
| --- | --- | --- |
| `button` | `label` | — |
| `input` | `label` | — |
| `tabs` | `items`, `active` | `items: 3`, `active: 1` |
| `slider` | `value` | `value: 50` |

`items` aceita número ou lista de rótulos; `active` é base 1 e admite 0 para
nenhum ativo; `value` vai de 0 a 100. O teto de itens de um Controle é
`controls.LimiteDeItens = 1_000`, pela mesma razão do teto de clones.

O catálogo vive em `internal/controls/` e é o único lugar que conhece o desenho
interno de um Controle. Acrescentar um Controle é acrescentar uma `Definicao`
ali; `schema`, `resolve`, `render` e `inspect` não têm caso por Controle.

A primeira peça do layout é a **cabeça**: ocupa a `box` declarada, herda `id` e
`note` do nó, e é a única que aparece na árvore. As demais são internas
(`Interno = true`), nunca carregam Nota, e cada uma é debitada do orçamento do
Frame — o teto do §8b conta Elementos materializados, e um Controle materializa
vários por nó.

O Rótulo é um Elemento de Forma `Texto`, cuja geometria é a **área** reservada
ao texto, não a caixa das glifas: a resolução não conhece métrica de fonte. O
tamanho da fonte é `0,45` da altura da área, derivado na rasterização. O Rótulo
**nunca é Superfície** para efeito de Elevação (§3). Texto que não cabe é
recortado na área.

A justificativa da revogação parcial do `docs/PRD.md` neste ponto está em
`docs/adr/0001-rotulo-de-controle-desenha-texto-no-frame.md`.

### Adendo 2d (catálogo da fatia 2, congelado)

O catálogo cresce para doze Controles. Nenhum campo novo entrou: os oito abaixo
cabem em `label`, `items`, `active` e `value`, e é isso que prova que a fronteira
do §2c está no lugar certo — acrescentar Controle não tocou em `schema`,
`resolve`, `render`, `inspect` nem na CLI.

| Controle | Campos | Padrões |
| --- | --- | --- |
| `checkbox` | `label`, `active` | `active: 0` |
| `radio` | `items`, `active` | `items: 3`, `active: 1` |
| `toggle` | `active` | `active: 0` |
| `accordion` | `items`, `active` | `items: 3`, `active: 1` |
| `dropdown` | `label` | — |
| `avatar` | `label` | — |
| `badge` | `label` | — |
| `progress` | `value` | `value: 50` |

Nos Controles de dois estados — `checkbox` e `toggle` — `active` só aceita 0 ou
1, e qualquer outro valor é erro: quem escreve 2 ali quase sempre achou que era
uma lista.

A cabeça do `avatar` é a única de Forma `Círculo`. Ela continua ocupando a `box`
declarada, como manda o §2c; a rasterização é que inscreve o Círculo no menor
lado da caixa.

Um Controle nunca materializa peça de área zero: `progress` em 0 é o trilho
vazio, e não um preenchimento de largura nula. O aviso de área zero do §6 é para
o que o autor escreveu, nunca para o que o catálogo desenhou por ele.

## 3. Elevação e Tom (congelado, dono F1)

Após o achatamento, para cada Frame, em ordem de pintura (Camadas na ordem declarada,
Elementos na ordem declarada dentro da Camada):

- `base[0] = 0` (o próprio Frame). `base[i] = max(base[i-1], maiorElevacaoNaCamada[i-1]) + 1`.
- Para um Elemento na Camada `i`: `pai` = **o último Elemento já pintado** (em qualquer
  Camada ≤ `i`) cuja bounding box contém a bounding box do Elemento (contenção
  inclusiva). Se não houver, `elevacaoDoPai = base[i]`, senão `elevacaoDoPai = pai.Elevacao`.
- `Elevacao = max(elevacaoDoPai, base[i]) + 1`.
- `Tom = scene.TomDaElevacao(Elevacao)`.

Elemento recortado pela borda usa a bounding box **declarada** (não recortada) na contenção.

Um Elemento de Forma `Texto` **nunca é pai**: o Rótulo é tinta sobre a Superfície que o
sustenta, não uma Superfície nova. Sem essa regra, qualquer Elemento desenhado por cima
de um Rótulo ganharia um degrau de Elevação só por passar em cima de um texto.

## 4. `inspect` — formato de árvore (congelado, dono F1)

Indentação de 2 espaços por nível. Coordenadas arredondadas para inteiro (`math.Round`).

```
documento <nome>
  frame <nome> <L>x<A>
    camada <nome>
      <caminho> <retangulo|circulo> <X>,<Y> <L>x<A> tom=<T> elev=<E>[ round][ de=<componente>][ controle=<nome>[ <parâmetros>]]
        nota: <texto>
```

`<caminho>`: segmentos separados por `/`. Segmento de Elemento é `e<indice>` na sua lista,
ou o `id` declarado quando houver. Instância acrescenta um segmento por nível de
Componente; Slot acrescenta o segmento `<nome-do-slot>`. Exemplos:
`e0`, `header`, `e3/e1`, `e3/body/e0`.

**Adendo 4b (clones de Repetição).** Um nó com `repeat` acrescenta ao próprio segmento o
sufixo `#<indice>`, com o índice começando em `0` e emitido **sempre**, inclusive quando
`n` é 1. O sufixo é determinístico, independe da ordem de pintura e diz *qual* clone é —
informação que o desempate do §8c não carrega. Exemplo: `e1#0`, `e1#1`, `grade#2/e0`.
Os dois mecanismos coexistem e se aplicam nesta ordem: primeiro `#<indice>`, depois, se o
caminho resultante ainda colidir, o `~2`/`~3` do §8c.

**Adendo 4c (Controle opaco).** Elemento com `Interno` verdadeiro **não é impresso**: o
Controle mostra a cabeça e os parâmetros, e omite as peças que materializou. As peças
continuam existindo na cena, no desenho e no orçamento — a opacidade é só da árvore. Os
seus caminhos seguem as mesmas regras (`<caminho-do-controle>/<segmento>`) e continuam
passando pela unicidade do §8c.

`de=` aparece só quando `Origem != ""`.

Nada é escrito em disco por `inspect`.

## 5. `internal/render` — API do Canvas (congelado, dono F3, consumido por F4)

```go
package render

// Canvas é a tela de saída. Todas as coordenadas dos métodos são em pixels do
// espaço do Frame (antes da escala); o Canvas aplica o fator internamente.
type Canvas struct{ /* ... */ }

// NewCanvas cria a tela. l e a são as dimensões do Frame em px. As quatro
// margens são o Chrome em px do espaço do Frame; podem ser 0. escala multiplica
// tudo. O Chrome é pintado com scene.TomChrome e o Frame com scene.TomFrame.
func NewCanvas(l, a int, margemT, margemD, margemB, margemE, escala float64) *Canvas

// DesenhaElemento pinta um Elemento resolvido, recortado ao Frame.
func (c *Canvas) DesenhaElemento(e scene.Elemento)

// Primitivas para o plano de anotação. Coordenadas relativas ao canto superior
// esquerdo da TELA INTEIRA (Chrome incluso), não ao Frame.
func (c *Canvas) Retangulo(x, y, l, a float64, t scene.Tom)
func (c *Canvas) Linha(x1, y1, x2, y2, espessura float64, t scene.Tom)
func (c *Canvas) Texto(x, y float64, s string, tamanho float64, t scene.Tom) // y = topo da linha
func (c *Canvas) MedeTexto(s string, tamanho float64) (l, a float64)
func (c *Canvas) QuebraTexto(s string, tamanho, larguraMax float64) []string

// OrigemDoFrame devolve o deslocamento do canto superior esquerdo do Frame
// dentro da tela, em px do espaço do Frame (= margemE, margemT).
func (c *Canvas) OrigemDoFrame() (x, y float64)

// CodificaWebP escreve WebP lossless. Determinístico: mesma entrada, mesmos bytes.
func (c *Canvas) CodificaWebP(w io.Writer) error

// DesenhaFrame cria o Canvas, pinta o fundo e todos os Elementos das Camadas
// indicadas (todas quando ateCamada < 0), e devolve o Canvas para anotação.
func DesenhaFrame(f scene.Frame, escala float64, margemT, margemD, margemB, margemE float64, ateCamada int) *Canvas
```

`internal/notes` importa `internal/render`. **`internal/render` nunca importa `internal/notes`.**

### 5b. Teto de área (adendo, congelado)

```go
// LimiteDeArea é o número máximo de pixels da tela de saída.
const LimiteDeArea = 64 << 20 // 67 108 864 px (~256 MB de RGBA)
```

`NewCanvas` **satura** nas dimensões que caibam nesse teto em vez de alocar sem limite,
e documenta a saturação. `DesenhaElemento` **recorta a bounding box do Elemento ao
retângulo do Frame antes de rasterizar** — sem isso, extensão acima de 2²⁵ px de
dispositivo trava o rasterizador do `freetype` num laço de CPU sem fim.

A CLI (`§7`, dono F1) recusa **antes** de chegar aqui: `--scale` deve ser `> 0` e a área
final `(margemE+l+margemD)*(margemT+a+margemB)*escala²` não pode passar de `LimiteDeArea`.
Excedeu, é **erro** com código 1, nunca pânico nem swap.

`Canvas` **não é seguro para uso concorrente**: um `Canvas` por goroutine.

### Adendo 5c (teto de tamanho da fonte)

O tamanho de fonte pedido ao `freetype` satura em **256 px de dispositivo**.
`truetype.NewFace` aloca de uma vez a máscara das 512 entradas do cache de
glifos, ao custo de `512 * (1,3 * tamanho)²` bytes — memória proporcional ao
**quadrado** do tamanho. Sem saturação, `--scale 10000` matava o processo por
falta de memória em vez de sair com código 1 e mensagem, porque a régua do plano
de anotação (§6) mede texto na escala do desenho final **antes** de a CLI poder
conferir o teto de área: o Chrome, que entra nessa conta, depende da medição.

256 px deixa o cache em ~56 MB, a mesma ordem de grandeza da tela que o teto de
área já permite. Acima de escala ~23 as Notas passam a sair relativamente
menores que o Frame; a medição e o desenho usam a mesma fonte saturada, então o
layout continua coerente consigo mesmo.

## 6. `internal/notes` — plano de anotação (congelado, dono F4)

```go
package notes

type Modo int
const (Margem Modo = iota; Flutuante; Desligado) // Margem é o padrão da CLI

// Plano calcula o layout das Notas de um Frame sem desenhar.
type Plano struct{ /* ... */ }

// Planeja resolve a posição de todas as Notas do Frame. escala é o fator da CLI.
func Planeja(f scene.Frame, m Modo, escala float64) *Plano

// Margens devolve o Chrome necessário em px do espaço do Frame. No modo
// Flutuante e Desligado devolve 0,0,0,0.
func (p *Plano) Margens() (t, d, b, e float64)

// Desenha pinta Notas e linhas de chamada sobre um Canvas já criado com essas
// margens. No modo Desligado não faz nada.
func (p *Plano) Desenha(c *render.Canvas)
```

Notas não participam da Elevação e **não aparecem no export por Camada**.

## 7. CLI (congelada, dono F1; F5 acrescenta só `skill`)

```
draftboard render   <arquivo.yaml> [--out DIR] [--scale N] [--notes margin|float|off] [--layers]
draftboard board    <arquivo.yaml> [--out DIR]
draftboard inspect  <arquivo.yaml>
draftboard validate <arquivo.yaml>
draftboard skill    [--install [DIR]]
```

- `--out` padrão `.`; `--scale` padrão `1` (float > 0); `--notes` padrão `margin`;
  `--layers` liga o export por Camada, cumulativo (cada imagem tem a Camada e todas abaixo).
- `render` imprime no **stdout apenas os caminhos escritos**, um por linha, na ordem de geração.
- `inspect` imprime a árvore no stdout e não toca em disco.
- `validate` não imprime nada no stdout em caso de sucesso.
- Avisos vão para **stderr**, com prefixo `aviso: `. Erros vão para **stderr**, com
  prefixo `erro: `, e saem com **código 1**. Sucesso sai com **0**.
- Formato de mensagem: `erro: <arquivo>: <local>: <mensagem>` — `<local>` é o caminho de
  chaves YAML, e atravessa a cadeia de Componentes (`home.yaml -> ./card.yaml`).

### Nomes de arquivo de saída

- `board`: `<doc>.html` — um arquivo por Documento, não por Frame.
- Sem `--layers`: `<doc>-<frame>.webp`
- Com `--layers`: `<doc>-<frame>-<nn>-<camada>.webp`, `nn` de dois dígitos a partir de `01`.
- `<doc>` é o nome do arquivo sem diretório e sem extensão. Todo componente do nome passa
  por slug: minúsculas, cada sequência de caracteres fora de `[a-z0-9]` vira um `-` único,
  `-` das pontas removidos.

### 7b. Verbos `version` e `update` (adendo, congelado, dono F7)

```
draftboard skill    [--install [DIR]] [--sync [DIR]] [--yes] [--no]
draftboard version
draftboard update   [--check] [--yes] [--no]
```

- `run` passa a receber `stdin`: `run(args []string, stdin io.Reader, stdout, stderr io.Writer) int`.
  Um verbo só o consome — `skill --sync`, que pergunta antes de regravar.
- `version` imprime três linhas no stdout: `draftboard <versao>`, `commit: <sha>`,
  `data: <RFC 3339>`. Sem os `-ldflags -X` do release, valem `dev`, `desconhecido` e
  `desconhecida` — é o estado de quem instalou por `go install`.
- `update` imprime no stdout **apenas o que foi escrito**: o caminho do binário
  substituído e, se a skill foi regravada, o caminho dela. Status, avisos e a pergunta
  vão para stderr.
- `update --check` **não escreve nada** e imprime exatamente uma linha de status:
  `atualização disponível: <nova> (atual: <atual>)` ou `já na versão mais recente: <v>`.
- **Códigos de saída seguem 0/1 do §7**: `--check` sai 0 sempre que a CONSULTA
  funcionou, e 1 só quando ela falhou. Não existe código 2 para "há versão nova".
- Versão atual não reconhecível (`dev`) conta como **desatualizada**: o `update`
  segue, depois de um aviso. A garantia contra binário adulterado é a soma SHA-256,
  não a comparação de Versão.
- `skill --sync` regrava a skill **só quando o conteúdo mudou**, e pergunta antes.
  Já sincronizada não imprime nada. **Entrada que não é um terminal nunca grava**:
  avisa no stderr e sai 0. `--yes`/`--no` respondem por quem não tem terminal.
- `--install` e `--sync` juntos são erro de uso.
- Erros de `version`, `update` e `skill` usam a forma `erro: <mensagem>` **sem** o
  prefixo `<arquivo>: <local>:` do §7 — não há Documento envolvido. Avisos seguem
  `aviso: <mensagem>`.
- Seam de teste: `DRAFTBOARD_LANCAMENTOS_URL`, quando não vazia, substitui a URL base
  da consulta. Não é documentada na skill, e **nenhum token de autenticação é anexado
  quando a URL base não é a padrão**.

## 8. Erros × avisos (congelado)

**Erro** (código 1): chave desconhecida (com sugestão), tipo inválido, mais de uma chave
discriminante no mesmo nó, nenhuma chave discriminante, `frames` vazio, `w`/`h` ausente ou
≤ 0, Componente inexistente, ciclo de referência entre Componentes, profundidade de
aninhamento > **16**, `repeat.n` < 1, `axis` fora de {x,y}, `slot` em Documento,
`box` ausente em `use`/`slot`.

**Aviso** (código 0, renderiza mesmo assim): Elemento fora do Frame (recortado),
Elemento de área zero, Slot sem preenchimento e sem `default` (renderiza Superfície vazia
com o degrau de Elevação).

### 8b. Tetos de materialização (adendo, congelado)

```go
// em internal/resolve
const LimiteDeClones    = 1_000   // clones por Repetição
const LimiteDeElementos = 10_000  // Elementos materializados por Frame
```

`repeat.n` deve ser um número **finito** e inteiro em `[1, LimiteDeClones]`. Valor não
finito, ou que estoure `int64` na conversão, é **erro** — nunca comportamento dependente
de plataforma. O total de Elementos materializados num Frame não pode passar de
`LimiteDeElementos`; excedeu, é **erro** com código 1, nomeando o Frame e o total.

Os dois tetos existem porque Repetições encadeadas através de Componentes multiplicam:
oito Componentes com `repeat: {n: 10}` cada, dentro do limite de 16 níveis, materializam
10⁸ Elementos a partir de ~1 KB de YAML. O princípio é o mesmo do `§5b` — recusar rápido
em vez de consumir memória até o processo morrer.

### 8c. Unicidade do `<caminho>` (adendo, congelado)

As duas regras de `§4` podem colidir: um Elemento com `id: X` e um Slot chamado `X` no
mesmo espaço produzem o mesmo `<caminho>`. A resolução garante unicidade dentro de cada
Frame: ao gerar um `<caminho>` já emitido, acrescenta o sufixo `~2`, `~3`, … na ordem de
pintura. A regra é determinística, portanto o `<caminho>` continua estável entre edições
que não mudem a ordem.

Sugestão de chave: distância de Levenshtein ≤ 3 contra as chaves válidas daquele nível;
formato `erro: card.yaml: elements[0]: campo desconhecido "rond"; você quis dizer "round"?`.
Sem candidato dentro do limite, omite a sugestão.

## 9. Testes

Seam primário é a CLI: fixtures em `testdata/<f>/`, invocar via `main` (use
`os/exec` sobre um binário construído em `TestMain`, ou uma função `run(args, stdout, stderr) int`
exportada internamente). Golden files de texto para `inspect` e para mensagens de erro.
Seam secundário é `internal/render`: `scene.Frame` construído à mão → bytes WebP golden.

Nenhum teste conhece a estrutura interna da resolução, nomes de fase do pipeline, ou como
a Elevação é computada — só o resultado observável.

Portão obrigatório antes de entregar: `gofmt -l .` (vazio), `go build ./...`,
`go vet ./...`, `go test ./...`.

### 9b. Testes do `update` (adendo, dono F7)

O seam primário do `update` é `internal/update` com `httptest.Server`, e não a CLI: a
CLI só alcança um servidor de teste pelo `DRAFTBOARD_LANCAMENTOS_URL` do adendo 7b.
Fixtures de Lançamento em `testdata/f7/`, com `%BASE%` no lugar do endereço do
servidor. O tarball e o `checksums.txt` são montados em tempo de teste — soma golden
estática quebraria a cada mudança de payload.

**Nenhum teste chama `update.Executavel()`, e `Opcoes.Destino` sempre aponta para
dentro de um `t.TempDir()`.** Um teste que deixe o Destino no padrão substituiria o
próprio binário de teste em execução.

Toda falha de `Aplica` é afirmada em dobro: os bytes originais do destino sobrevivem
**e** o diretório continua com exatamente uma entrada — a primeira metade prova que
nada foi trocado, a segunda que nenhum temporário ficou para trás.

## 10. Distribuição (adendo, congelado, dono F7)

O `update` depende de um contrato de nomes com o workflow de release. Mudar qualquer
linha desta tabela quebra o updater das versões já instaladas.

| Item | Valor |
| --- | --- |
| Ativo | `draftboard_<tag>_<goos>_<goarch>.tar.gz` — a tag entra **verbatim**, com o `v` |
| Conteúdo do Ativo | **exatamente uma** entrada regular, na raiz, chamada `draftboard` |
| Somas | `checksums.txt`, formato `sha256sum`: 64 hex, dois espaços, nome-base do Ativo |
| Consulta | `GET <base>/releases/latest` |
| Plataformas | darwin/arm64, darwin/amd64, linux/amd64, linux/arm64 |

- **O cliente não embute a lista de plataformas**: ele calcula o nome esperado a partir
  de `runtime.GOOS`/`GOARCH` e o procura entre os Ativos publicados. Ativo ausente é o
  erro. Publicar uma plataforma nova não exige tocar no cliente.
- `/releases/latest` exclui draft e prerelease. O workflow publica com `--prerelease`
  toda tag que contenha `-`, então uma `v1.0.0-rc.1` é baixável por URL mas **nunca é
  oferecida** pelo `update`.
- Um nome que aparece zero ou duas-ou-mais vezes no `checksums.txt` é Lançamento
  quebrado: as duas situações são erro, nunca sorteio.
- A troca do binário é sempre `os.Rename` sobre o **arquivo real** (`os.Executable` +
  `EvalSymlinks`), com o temporário criado no diretório do destino — nunca em
  `os.TempDir()`, que costuma estar em outro device e não tem move atômico na
  biblioteca padrão. **Nada é renomeado antes de a soma conferir.**
- Limite conhecido, registrado de propósito: SHA-256 sobre HTTPS protege contra
  corrupção e truncamento, **não** contra Lançamento comprometido. Assinatura de
  verdade exigiria dependência nova (minisign, cosign) ou shell-out, e as duas coisas
  são proibidas pelo §0.

## 11. Ligações e Prancheta (adendo, dono F8)

Duas coisas novas, e uma reversão explícita: o PRD original listava "links entre
Frames" como fora de escopo. A fatia reverte essa exclusão; a exclusão de outros
**formatos de imagem** continua de pé.

### Mudanças aditivas em contratos congelados

Três, todas por acréscimo — nenhuma assinatura existente mudou:

| Onde | O quê | Por quê |
| --- | --- | --- |
| `scene.Elemento` | campo `Destino string` | a Ligação é do Elemento, e o modelo resolvido é o único lugar por onde ela chega à Prancheta |
| `internal/render` | `func Raio(l, a float64) float64` | a regra do canto arredondado passa a ter um dono só: raster e SVG não podem divergir |
| `internal/render` | `func TamanhoDoRotulo(a float64) float64` | mesma razão, para a altura da fonte do Rótulo |

`internal/render` **não muda de comportamento**: `Destino` é ignorado no raster.
Acrescentar `to` a um Documento não altera nenhum byte do WebP dele.

### A chave `to`

```yaml
- control: button
  box: {x: 4, y: 40, w: 18, h: 7}
  label: "Entrar"
  to: dashboard
```

Regras, todas **erro**:

| Situação | Mensagem |
| --- | --- |
| destino que não é `name` de nenhum Frame do Documento | `Ligação para Frame desconhecido "x"; você quis dizer "y"?` (sugestão quando há nome próximo) |
| `to` em Componente | `Ligação só pode ser declarada em Documento, não em Componente` |
| `to` em `use` ou `slot` | `campo "to" só é permitido em Retângulo, Círculo ou Controle` |
| `to` junto de `repeat` | `campo "to" não pode ser usado com "repeat"` |
| `to` vazio | `Ligação sem nome de Frame em "to"` |

`to` em `use`/`slot` é recusado porque nenhum dos dois deixa um Elemento seu na
cena: eles viram o conteúdo que expandiram, e a seta não teria de onde sair.
Ligar um Frame a si mesmo é válido. Nome de Frame repetido resolve pelo primeiro.

O `Destino` mora no mesmo lugar que o `ID` e a `Nota`: no Elemento do `rect`/
`circle`, e na **cabeça** do Controle.

### `inspect`

A linha do Elemento ganha ` para=<frame>` no fim, depois de `<parâmetros>`.

### `internal/board`

```go
const LimiteDeElementos = 50_000
func Elementos(d *scene.Documento) int
func Escreve(w io.Writer, d *scene.Documento) error
```

- A saída é **determinística**: mesmo Documento, mesmos bytes.
- **Autocontida**: CSS e JS inline, zero requisição de rede, zero arquivo ao
  lado. Abre por `file://`. O único endereço no arquivo é o espaço de nomes do
  SVG.
- Todo texto vindo do YAML é escapado antes de entrar no documento.
- A CLI recusa o Documento acima de `LimiteDeElementos` **antes** de montar o
  HTML e antes de criar o diretório de saída, no mesmo desenho de `cabeNaTela`.

### Layout (derivado, nunca declarado)

Não existe campo de posição de Frame, pela mesma razão que não existe campo de
Tom. A coluna de um Frame é a **distância mais curta** até uma tela de entrada
(Frame sem Ligação de entrada), por busca em largura na ordem de declaração:

- A distância é a mais curta, não a mais longa: quase todo fluxo tem Ligação de
  volta, e o caminho mais longo jogaria a tela de entrada para o fim.
- Fluxo inteiramente em ciclo não tem tela de entrada: a primeira declarada é a
  entrada.
- Trecho desligado do grafo recomeça na coluna 0.
- Documento sem Ligação nenhuma vira grade de `ceil(sqrt(n))` colunas.
- Auto-Ligação não conta como entrada e não move o Frame.

### Desenho

Os Frames são SVG gerado a partir de `scene.Documento`, com a mesma escala de
Tom, o mesmo raio de canto e o mesmo tamanho de Rótulo do raster. O que **não** é
igual ao WebP: a fonte, que na Prancheta é a pilha do sistema e não a `goregular`
embutida. Consequência aceita — embutir a fonte inflaria o arquivo por um ganho
que não é o propósito da Prancheta.

As peças internas de um Controle são desenhadas (elas existem no desenho) mas não
recebem clique: quem clica num Controle seleciona o Controle, nunca o seu miolo.
O Controle é fechado também na Prancheta.
