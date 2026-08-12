---
name: draftboard
description: Use ao escrever, validar ou renderizar wireframes declarativos em YAML com o binário draftboard, que produz imagens WebP em escala de cinza e uma árvore textual da estrutura.
---

# draftboard

Wireframes declarativos em YAML → imagens WebP em escala de cinza + árvore textual.
Toda geometria é percentual; **nenhuma cor é declarada** — o Tom (cinza) vem só da Elevação.

## Exemplo completo (Documento com dois Frames)

```yaml
# home.yaml
frames:
  - name: home
    w: 1280
    h: 800
    layers:
      - name: conteudo
        elements:
          - rect: {x: 0, y: 0, w: 100, h: 10}
            id: header
            note: "Cabeçalho fixo"
          - rect: {x: 4, y: 14, w: 92, h: 24}
            round: true
          - circle: {x: 92, y: 2, d: 5}
            id: avatar
      - name: overlay
        elements:
          - rect: {x: 25, y: 30, w: 50, h: 40}
            round: true
            note: "Diálogo de confirmação"
  - name: listagem
    w: 1280
    h: 800
    layers:
      - name: conteudo
        elements:
          - rect: {x: 4, y: 4, w: 92, h: 10}
            repeat: {n: 5, axis: y, gap: 2}
```

`draftboard render home.yaml` escreve `home-home.webp` e `home-listagem.webp`.

## Exemplo de Componente com Slot instanciado

```yaml
# card.yaml — Componente: espaço local 0..100 nos dois eixos, sem `frames`
elements:
  - rect: {x: 0, y: 0, w: 100, h: 100}
    round: true
  - circle: {x: 5, y: 6, d: 14}
  - slot: "body"
    box: {x: 6, y: 30, w: 88, h: 62}
    default:
      - rect: {x: 0, y: 0, w: 100, h: 25}
```

```yaml
# home.yaml — instanciando o Componente e preenchendo o Slot
frames:
  - name: home
    w: 1280
    h: 800
    layers:
      - name: conteudo
        elements:
          - use: "./card.yaml"
            box: {x: 5, y: 10, w: 40, h: 35}
            id: cartao
            slots:
              body:
                elements:
                  - rect: {x: 0, y: 0, w: 100, h: 45}
          - use: "./card.yaml"
            box: {x: 55, y: 10, w: 40, h: 35}
            repeat: {n: 2, axis: y, gap: 5}
            slots:
              body: {use: "./avatar.yaml"}
```

Caminho de `use` é **relativo ao arquivo que referencia**.
Preenchimento de Slot é `{use: "./x.yaml"}` **ou** `{elements: [ ... ]}`.

## Estrutura do arquivo

O tipo é inferido pelo conteúdo: tem `frames` → **Documento**; senão → **Componente**.

| Nível | Chaves |
| --- | --- |
| Documento | `frames` (lista, não vazia) |
| Frame | `name`, `w` (>0), `h` (>0), `layers` |
| Camada | `name`, `elements` |
| Componente | `elements` |

## Nós de elemento

Exatamente **uma** chave discriminante por nó: `rect`, `circle`, `use` ou `slot`.

| Nó | Valor | Chaves adicionais permitidas |
| --- | --- | --- |
| `rect: {x, y, w, h}` | geometria em % | `round`, `id`, `note`, `repeat` |
| `circle: {x, y, d}` | geometria em % | `id`, `note`, `repeat` |
| `use: "./comp.yaml"` | caminho relativo | `box` (**obrigatório**), `slots`, `id`, `note`, `repeat` |
| `slot: "nome"` | nome do Slot | `box` (**obrigatório**), `default`, `id`, `note`, `repeat` |

- `round: true` só existe em `rect`. Cantos retos é o padrão.
- `slots` só existe em `use`. `default` (lista de nós) só existe em `slot`.
- `slot` só é válido dentro de Componente — nunca em Documento.
- `note` é a Nota: texto anexado ao Elemento, fora do desenho.
- `id` renomeia o segmento do Elemento no caminho da árvore.
- Toda chave desconhecida, em qualquer nível, é erro com sugestão da chave próxima.

## Geometria

Todo valor é **percentual do eixo correspondente** do espaço em que o nó está.
Âncora é sempre o canto superior esquerdo, inclusive para Círculo.

No Frame de `FL`×`FA` px:

```
X = x/100*FL    Y = y/100*FA    L = w/100*FL    A = h/100*FA
Círculo:  L = A = d/100*FL      (largura nos dois eixos — nunca vira elipse)
```

No espaço local de Componente/Slot mapeado na caixa px `(bx, by, bl, ba)`:

```
X = bx + x/100*bl    Y = by + y/100*ba    L = w/100*bl    A = h/100*ba
Círculo:  L = A = d/100*bl
```

### Repetição

```yaml
repeat: {n: 3, axis: y, gap: 2}
```

`n` clones (`n` ≥ 1), `axis` ∈ {`x`, `y`}, `gap` em % do eixo do espaço local.
O clone `i` desloca `i * (tamanho + gap)` no eixo, com `tamanho` = `w`/`h` do `rect`,
`d` do `circle`, ou `box.w`/`box.h` da Instância — em unidades do espaço local, antes
da conversão para px.

## Elevação e Tom

**Não existe declaração de cor.** O Tom (cinza de 100, quase branco, a 900, quase preto)
é derivado automaticamente da Elevação, que é a distância do Elemento até o Frame contada
em Superfícies empilhadas.

- O Frame tem Elevação 0 e Tom 100. O Chrome usa 900, reservado.
- Cada Camada adiciona um degrau sobre tudo abaixo dela.
- Um Elemento contido geometricamente em outro fica um degrau acima dele.
- Superfícies adjacentes sempre diferem visivelmente; a escala nunca esgota.

Consequência prática: para dar contraste a um Elemento, **aninhe-o** dentro de outro
(contenção geométrica) ou coloque-o numa Camada acima — não tente escolher um cinza.

## CLI

```
draftboard render   <arquivo.yaml> [--out DIR] [--scale N] [--notes margin|float|off] [--layers]
draftboard inspect  <arquivo.yaml>
draftboard validate <arquivo.yaml>
draftboard skill    [--install [DIR]]
```

| Flag | Verbo | Padrão | Efeito |
| --- | --- | --- | --- |
| `--out DIR` | `render` | `.` | diretório de saída |
| `--scale N` | `render` | `1` | multiplicador float > 0 de toda a imagem |
| `--notes MODO` | `render` | `margin` | `margin` (Notas no Chrome), `float` (sobre o Frame), `off` (sem Notas) |
| `--layers` | `render` | desligado | uma imagem por Camada, cumulativa (a Camada e todas abaixo) |
| `--install [DIR]` | `skill` | `~/.claude/skills` | grava a skill em `<DIR>/draftboard/SKILL.md` |

- `render` imprime no stdout **apenas os caminhos escritos**, um por linha, na ordem de geração.
- `inspect` imprime a árvore no stdout e **não escreve nada em disco**.
- `validate` não imprime nada no stdout quando passa.
- `skill` sem `--install` imprime a skill no stdout.
- Avisos vão para stderr com prefixo `aviso: `; erros vão para stderr com prefixo `erro: `.
- Sucesso sai com código `0`; erro sai com `1`.

### Nomes dos arquivos de saída

| Modo | Nome |
| --- | --- |
| sem `--layers` | `<doc>-<frame>.webp` |
| com `--layers` | `<doc>-<frame>-<nn>-<camada>.webp`, `nn` de dois dígitos a partir de `01` |

`<doc>` é o nome do arquivo sem diretório e sem extensão. Todo componente do nome passa
por slug: minúsculas, cada sequência fora de `[a-z0-9]` vira um `-` único, `-` das pontas
removidos.

## Formato do `inspect`

Indentação de 2 espaços por nível; coordenadas arredondadas para inteiro.

```
documento <nome>
  frame <nome> <L>x<A>
    camada <nome>
      <caminho> <retangulo|circulo> <X>,<Y> <L>x<A> tom=<T> elev=<E>[ round][ de=<componente>]
        nota: <texto>
```

`<caminho>`: segmentos separados por `/`. O segmento de um Elemento é `e<indice>` na sua
lista, ou o `id` declarado quando houver. Uma Instância acrescenta um segmento por nível
de Componente; um Slot acrescenta o segmento com o nome do Slot.
Exemplos: `e0`, `header`, `e3/e1`, `e3/body/e0`. `de=` só aparece para Elementos vindos
de Componente.

## Erros × avisos

**Erro** — aborta, código 1:

| Situação |
| --- |
| chave desconhecida (a mensagem sugere a chave válida mais próxima) |
| tipo inválido |
| mais de uma chave discriminante no mesmo nó, ou nenhuma |
| `frames` vazio |
| `w` ou `h` ausente ou ≤ 0 |
| Componente inexistente, ou ciclo de referência entre Componentes |
| profundidade de aninhamento maior que 16 |
| `repeat.n` < 1, ou `axis` fora de {`x`, `y`} |
| `slot` declarado em Documento |
| `box` ausente em `use` ou `slot` |

Formato: `erro: <arquivo>: <local>: <mensagem>`, onde `<local>` é o caminho de chaves YAML
e atravessa a cadeia de Componentes. Exemplo:

```
erro: card.yaml: elements[0]: campo desconhecido "rond"; você quis dizer "round"?
```

**Aviso** — renderiza mesmo assim, código 0:

| Situação |
| --- |
| Elemento fora do Frame (é recortado) |
| Elemento de área zero |
| Slot sem preenchimento e sem `default` (vira Superfície vazia, com o degrau de Elevação) |

## Receita

1. Escreva o Documento; extraia em Componente o que se repete com variação de conteúdo.
2. `draftboard validate home.yaml` — silêncio significa válido.
3. `draftboard inspect home.yaml` — confira caminhos, geometria em px e a escada de Elevação.
4. `draftboard render home.yaml --out ./out --scale 2` — gere as imagens.
