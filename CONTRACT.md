# CONTRATO — draftboard

Congelado. Mudança aqui volta ao orquestrador, nunca é negociada entre implementadores.
Vocabulário obrigatório: o de `CONTEXT.md`, em código, mensagens de erro e skill.

## 0. Módulo e árvore

`module github.com/eduardotorresdev/draftboard`, Go 1.26, **sem cgo**.

`go.mod` e `go.sum` **já estão prontos e são congelados**. Não rode `go mod tidy`,
não adicione dependência. Deps disponíveis: `gopkg.in/yaml.v3`, `github.com/fogleman/gg`,
`github.com/HugoSmits86/nativewebp`, `golang.org/x/image` (inclui `font/gofont/goregular`).

Dono de cada diretório (ninguém escreve fora do seu):

| Diretório | Dono |
| --- | --- |
| `internal/scene/` | **orquestrador — congelado, ninguém edita** |
| `internal/schema/`, `internal/resolve/`, `internal/inspect/` | F1, depois F2 |
| `internal/render/` | F3 |
| `internal/notes/` | F4 |
| `internal/skill/`, `SKILL.md` | F5 |
| `main.go` | F1 (dispatch + `render`/`inspect`/`validate`); F5 acrescenta só o caso `skill` |
| `testdata/` | subdiretório por funcionalidade: `testdata/f1/`, `testdata/f2/`, ... |

## 1. `internal/scene` — o contrato central

Já implementado e congelado. Leia `internal/scene/scene.go`. Resumo:

```go
type Tom int                    // 100 (quase branco) .. 900 (quase preto)
const TomFrame Tom = 100        // fundo fixo de todo Frame
const TomChrome Tom = 900       // extremo reservado ao Chrome
func TomDaElevacao(elevacao int) Tom
func (t Tom) Cinza() uint8

type Forma int; const (Retangulo Forma = iota; Circulo)

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
}
type Camada struct { Nome string; Elementos []Elemento }
type Frame struct { Nome string; L, A int; Camadas []Camada }
type Documento struct { Nome string; Frames []Frame }
type Aviso struct { Arquivo, Local, Msg string }
```

`scene.Documento` é achatado: Instâncias, Slots e Repetições **já não existem** nele.

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

## 4. `inspect` — formato de árvore (congelado, dono F1)

Indentação de 2 espaços por nível. Coordenadas arredondadas para inteiro (`math.Round`).

```
documento <nome>
  frame <nome> <L>x<A>
    camada <nome>
      <caminho> <retangulo|circulo> <X>,<Y> <L>x<A> tom=<T> elev=<E>[ round][ de=<componente>]
        nota: <texto>
```

`<caminho>`: segmentos separados por `/`. Segmento de Elemento é `e<indice>` na sua lista,
ou o `id` declarado quando houver. Instância acrescenta um segmento por nível de
Componente; Slot acrescenta o segmento `<nome-do-slot>`. Exemplos:
`e0`, `header`, `e3/e1`, `e3/body/e0`.

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

- Sem `--layers`: `<doc>-<frame>.webp`
- Com `--layers`: `<doc>-<frame>-<nn>-<camada>.webp`, `nn` de dois dígitos a partir de `01`.
- `<doc>` é o nome do arquivo sem diretório e sem extensão. Todo componente do nome passa
  por slug: minúsculas, cada sequência de caracteres fora de `[a-z0-9]` vira um `-` único,
  `-` das pontas removidos.

## 8. Erros × avisos (congelado)

**Erro** (código 1): chave desconhecida (com sugestão), tipo inválido, mais de uma chave
discriminante no mesmo nó, nenhuma chave discriminante, `frames` vazio, `w`/`h` ausente ou
≤ 0, Componente inexistente, ciclo de referência entre Componentes, profundidade de
aninhamento > **16**, `repeat.n` < 1, `axis` fora de {x,y}, `slot` em Documento,
`box` ausente em `use`/`slot`.

**Aviso** (código 0, renderiza mesmo assim): Elemento fora do Frame (recortado),
Elemento de área zero, Slot sem preenchimento e sem `default` (renderiza Superfície vazia
com o degrau de Elevação).

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
