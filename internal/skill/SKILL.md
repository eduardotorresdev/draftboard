---
name: draftboard
description: Use ao escrever, validar ou renderizar wireframes declarativos em YAML com o binário draftboard, que produz imagens WebP em escala de cinza, uma Prancheta HTML navegável do fluxo e uma árvore textual da estrutura.
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

Exatamente **uma** chave discriminante por nó: `rect`, `circle`, `use`, `slot` ou `control`.

| Nó | Valor | Chaves adicionais permitidas |
| --- | --- | --- |
| `rect: {x, y, w, h}` | geometria em % | `round`, `label`, `id`, `note`, `to`, `repeat` |
| `circle: {x, y, d}` | geometria em % | `id`, `note`, `to`, `repeat` |
| `use: "./comp.yaml"` | caminho relativo | `box` (**obrigatório**), `slots`, `id`, `note`, `repeat` |
| `slot: "nome"` | nome do Slot | `box` (**obrigatório**), `default`, `id`, `note`, `repeat` |
| `control: nome` | nome do catálogo | `box` (**obrigatório**), `id`, `note`, `to`, `repeat`, e os campos do Controle |

- `round: true` só existe em `rect`. Cantos retos é o padrão.
- `label` só existe em `rect` e em `control`: é o Rótulo, o nome do bloco no plano do
  desenho. Num `circle`, num `use` ou num `slot` é erro.
- `slots` só existe em `use`. `default` (lista de nós) só existe em `slot`.
- `slot` só é válido dentro de Componente — nunca em Documento.
- `control` é fechado: não aceita `slots`, `default` nem `round`, e não recebe conteúdo.
- `note` é a Nota: texto anexado ao Elemento, fora do desenho. A Nota **não participa da Elevação e não aparece no export por Camada** (`--layers`); some inteira com `--notes off`.
- `to` é a Ligação: o nome do Frame para onde este Elemento leva. Só aparece na Prancheta (`board`) e na árvore do `inspect` — **não muda o desenho nem a Elevação**, e a imagem WebP sai igual com e sem ele.
- `id` renomeia o segmento do Elemento no caminho da árvore.
- Toda chave desconhecida, em qualquer nível, é erro com sugestão da chave próxima.

## Controles

O Controle é uma peça pronta do catálogo embutido: em vez de montar um botão com
vários `rect`, você declara o que ele é e recebe o desenho inteiro. É **fechado** —
não se abre, não recebe Slot, e no `inspect` ocupa **uma linha só**, com os
parâmetros declarados. O que ele materializa por dentro não aparece na árvore.

```yaml
- control: tabs
  box: {x: 4, y: 10, w: 92, h: 6}
  items: ["Perfil", "Conta", "Faturas"]
  active: 2
- control: slider
  box: {x: 4, y: 30, w: 40, h: 4}
  value: 70
- control: button
  box: {x: 4, y: 40, w: 18, h: 7}
  label: "Salvar"
```

| Controle | Campos | Padrão |
| --- | --- | --- |
| `button` | `label` | sem `label`, o rótulo vira barra cinza |
| `input` | `label` | sem `label`, o rótulo vira barra cinza |
| `tabs` | `items`, `active` | `items: 3`, `active: 1` |
| `slider` | `value` | `value: 50` |
| `checkbox` | `label`, `active` | `active: 0` (desmarcado) |
| `radio` | `items`, `active` | `items: 3`, `active: 1` |
| `toggle` | `active` | `active: 0` (desligado) |
| `accordion` | `items`, `active` | `items: 3`, `active: 1`; a seção ativa abre |
| `dropdown` | `label` | sem `label`, o rótulo vira barra cinza |
| `avatar` | `label` | sem `label`, é só o disco |
| `badge` | `label` | sem `label`, o rótulo vira barra cinza |
| `progress` | `value` | `value: 50` |

- `items` aceita **número** (itens sem texto) ou **lista de rótulos** (itens com texto).
- `active` é base 1; `active: 0` deixa nenhum item ativo.
- Em `checkbox` e `toggle`, `active` só aceita 0 ou 1: eles têm dois estados, não lista.
- `value` vai de 0 a 100.
- O tamanho da fonte do Rótulo é derivado da altura da área — **não existe campo de
  fonte, de alinhamento nem de cor**, pela mesma razão que não existe campo de Tom.
- Rótulo que não cabe é recortado na área do Controle.

## Rótulo no Retângulo

`label` num `rect` desenha o nome do bloco **dentro dele**, no plano do desenho: ao
contrário da Nota, participa da Elevação, sai no export por Camada e está na imagem.

```yaml
- rect: {x: 4, y: 6, w: 92, h: 40}
  label: "Resultados"
  # contém filhos -> faixa no topo, à esquerda
- rect: {x: 4, y: 52, w: 44, h: 12}
  label: "Filtros"
  # vazio -> caixa centrada na vertical
```

**A posição é derivada, não declarada.** Retângulo que contém geometricamente outro
Elemento apoia o Rótulo numa faixa no topo, alinhada à esquerda, fora do caminho dos
filhos; Retângulo vazio o centraliza na vertical. Ganhar um filho move o Rótulo
sozinho. Não existe chave para forçar nem para desligar.

- A caixa do Rótulo tem **28 px de altura** no espaço do Frame, saturando na altura do
  Retângulo quando ele é mais baixo, e recua 6 px de cada lado. A altura é fixa, e não
  uma fração do bloco: um bloco de 400 px de altura teria um Rótulo de mockup.
- Cabe cerca de 28 px de altura de texto, e só uma linha: o Rótulo **não quebra**.
  Texto que não cabe é recortado na área dele. Um Rótulo que estoura o bloco no
  wireframe vai estourar o componente na UI.
- O tamanho da fonte é derivado da altura da caixa, como no Controle — sem campo de
  fonte, de alinhamento nem de cor.
- O texto do Rótulo tem teto de **200 runas**: acima disso a decodificação recusa o
  Documento. É o mesmo teto da Nota, e pela mesma razão — Rótulo é nome de bloco.
- No `inspect`, o Rótulo aparece como sufixo `rotulo="..."` na linha do próprio
  Retângulo; o texto não vira uma linha a mais na árvore.

## Ligações e Prancheta

`to` liga um Elemento a um Frame do mesmo Documento: é o gatilho que leva àquela tela.
`draftboard board` reúne todos os Frames numa **Prancheta** — um HTML autocontido, com
as Ligações desenhadas entre as telas, que se navega com pan/zoom e onde clicar num
Elemento mostra caminho, geometria, Elevação, Tom e Nota.

```yaml
frames:
  - name: login
    w: 480
    h: 720
    layers:
      - name: conteudo
        elements:
          - control: button
            box: {x: 10, y: 46, w: 80, h: 8}
            label: "Entrar"
            to: dashboard
  - name: dashboard
    w: 1280
    h: 800
    layers:
      - name: conteudo
        elements:
          - control: button
            box: {x: 86, y: 2, w: 12, h: 4}
            label: "Sair"
            to: login
```

`draftboard board fluxo.yaml` escreve `fluxo.html`.

Regras:

- O valor de `to` é o `name` de um Frame do mesmo Documento. Nome desconhecido é erro, com sugestão do nome próximo.
- `to` **não é permitido em Componente**: um Componente não conhece Frame.
- `to` **não é permitido em `use` nem em `slot`**: só em `rect`, `circle` e `control`, que são os que deixam um Elemento no Documento resolvido.
- `to` **não convive com `repeat`** no mesmo nó: repetir o gatilho não repete a seta.
- Ligar um Frame a si mesmo é válido e vira um laço.
- A posição dos Frames na Prancheta é automática — não existe campo de posição. Frames sem Ligação de entrada ficam na primeira coluna, e cada Ligação empurra o destino uma coluna à direita.
- A Prancheta não busca nada de fora: abre por `file://`, sem rede e sem arquivo ao lado.

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
- Ordem de pintura: Camadas na ordem declarada, Elementos na ordem declarada dentro da Camada.
- Cada Camada `i` parte de um piso: `base[0] = 0` e `base[i] = max(base[i-1], maiorElevacaoNaCamada[i-1]) + 1`.
- O pai de um Elemento é o **último Elemento já pintado** (em qualquer Camada ≤ `i`) cuja bounding box contém a do Elemento, com contenção inclusiva. Sem pai, `elevacaoDoPai = base[i]`.
- `Elevacao = max(elevacaoDoPai, base[i]) + 1` — pode subir mais de um degrau de uma vez quando o pai está numa Camada inferior.
- Elemento recortado pela borda do Frame entra na contenção com a bounding box **declarada**, não com a recortada.
- Superfícies adjacentes sempre diferem visivelmente; a escala nunca esgota.

Consequência prática: para dar contraste a um Elemento, **aninhe-o** dentro de outro
(contenção geométrica) ou coloque-o numa Camada acima — não tente escolher um cinza.

## CLI

```
draftboard render   <arquivo.yaml> [--out DIR] [--scale N] [--notes margin|float|off] [--layers]
draftboard board    <arquivo.yaml> [--out DIR]
draftboard inspect  <arquivo.yaml>
draftboard validate <arquivo.yaml>
draftboard skill    [--install [DIR]] [--sync [DIR]] [--yes] [--no]
draftboard version
draftboard update   [--check] [--yes] [--no]
```

| Flag | Verbo | Padrão | Efeito |
| --- | --- | --- | --- |
| `--out DIR` | `render`, `board` | `.` | diretório de saída |
| `--scale N` | `render` | `1` | multiplicador float > 0 de toda a imagem |
| `--notes MODO` | `render` | `margin` | `margin` (Notas no Chrome), `float` (sobre o Frame), `off` (sem Notas) |
| `--layers` | `render` | desligado | uma imagem por Camada, cumulativa (a Camada e todas abaixo) |
| `--install [DIR]` | `skill` | `~/.claude/skills` | grava a skill em `<DIR>/draftboard/SKILL.md` |
| `--sync [DIR]` | `skill` | `~/.claude/skills` | regrava a skill só se o conteúdo mudou, perguntando antes |
| `--check` | `update` | desligado | só reporta se há versão nova; não escreve nada |
| `--yes` | `skill`, `update` | desligado | responde `s` a qualquer pergunta |
| `--no` | `skill`, `update` | desligado | responde `n` a qualquer pergunta |

- `render` imprime no stdout **apenas os caminhos escritos**, um por linha, na ordem de geração.
- `board` escreve **um** arquivo, a Prancheta do Documento inteiro, e imprime só o caminho dele no stdout. Não aceita `--scale` nem `--notes`: a Prancheta é vetorial e mostra a Nota na inspeção.
- `inspect` imprime a árvore no stdout e **não escreve nada em disco**.
- `validate` não imprime nada no stdout quando passa.
- `skill` sem `--install` imprime a skill no stdout.
- `version` imprime versão, commit e data do build no stdout.
- `update` troca o binário pelo do último lançamento e imprime no stdout o caminho substituído.
- `update --check` só reporta se há versão nova e **não escreve nada**.
- `skill --sync` regrava a skill só quando ela mudou, e não grava nada quando a entrada não é um terminal.
- Avisos vão para stderr com prefixo `aviso: `; erros vão para stderr com prefixo `erro: `.
- Sucesso sai com código `0`; erro sai com `1`.

### Nomes dos arquivos de saída

| Modo | Nome |
| --- | --- |
| sem `--layers` | `<doc>-<frame>.webp` |
| com `--layers` | `<doc>-<frame>-<nn>-<camada>.webp`, `nn` de dois dígitos a partir de `01` |
| `board` | `<doc>.html` |

`<doc>` é o nome do arquivo sem diretório e sem extensão. Todo componente do nome passa
por slug: minúsculas, cada sequência fora de `[a-z0-9]` vira um `-` único, `-` das pontas
removidos.

## Formato do `inspect`

Indentação de 2 espaços por nível; coordenadas arredondadas para inteiro.

```
documento <nome>
  frame <nome> <L>x<A>
    camada <nome>
      <caminho> <retangulo|circulo|texto> <X>,<Y> <L>x<A> tom=<T> elev=<E>[ round][ de=<componente>][ controle=<nome> <parâmetros>][ para=<frame>][ rotulo="<texto>"]
        nota: "<texto>"
```

`<caminho>`: segmentos separados por `/`. O segmento de um Elemento é `e<indice>` na sua
lista, ou o `id` declarado quando houver. Uma Instância acrescenta um segmento por nível
de Componente; um Slot acrescenta o segmento com o nome do Slot.
Exemplos: `e0`, `header`, `e3/e1`, `e3/body/e0`. `de=` só aparece para Elementos vindos
de Componente. `para=` só aparece para Elementos que declaram Ligação. `rotulo=` fecha a
linha de um Retângulo com `label`; num Controle ele é um dos parâmetros e vem logo depois
de `controle=`, antes de `para=`. A Nota e o Rótulo saem entre aspas, com escapes: são as
duas entradas de texto livre do autor, e cruas forjariam linhas na árvore.

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
| nome de Controle fora do catálogo (a mensagem sugere o nome válido mais próximo) |
| campo de Controle usado noutro nó, ou campo que aquele Controle não aceita |
| `active` além da quantidade de itens, ou `value` fora de 0..100 |
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

1. Escreva o Documento; use `control` para as peças de interface e extraia em
   Componente o que se repete com variação de conteúdo.
2. `draftboard validate home.yaml` — silêncio significa válido.
3. `draftboard inspect home.yaml` — confira caminhos, geometria em px e a escada de Elevação.
4. `draftboard render home.yaml --out ./out --scale 2` — gere as imagens.
