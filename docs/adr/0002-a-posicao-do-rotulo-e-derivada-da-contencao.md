# 2. A posição do Rótulo é derivada da contenção, não declarada

Data: 2026-08-19

## Status

Aceita. Estende a ADR-0001, que criou o Rótulo restrito ao Controle.

## Contexto

O Rótulo nasceu fechado dentro do Controle: o catálogo escolhia a caixa e o
alinhamento de cada texto, e o autor do Documento não tinha como nomear um bloco
seu. Nomear era trabalho da Nota — que vive no plano de anotação, é ancorada por
uma linha de chamada e obriga o leitor a atravessar o desenho com os olhos para
descobrir o que cada Retângulo é.

Abrir `label:` para o Retângulo resolve isso, mas cria uma pergunta que o
Controle nunca teve: um Retângulo tem tamanho arbitrário e pode ou não conter
outros Elementos. O Rótulo de uma região que contém filhos precisa ficar no topo,
fora do caminho deles; o de um bloco vazio quer o centro. Alguém tem que decidir
qual dos dois, e havia três candidatos: um discriminante novo (`section:`), uma
chave modificadora (`label-at:`), ou a própria engine.

## Decisão

A posição é derivada de `contemGeometricamente` — a mesma relação que já decide a
Superfície de cada Elemento e, por consequência, a Elevação e o Tom. Retângulo
que contém outros Elementos apoia o Rótulo numa faixa no topo, à esquerda;
Retângulo vazio o centraliza.

Não existe chave para forçar. Ganhar um filho move o Rótulo sozinho, do mesmo
jeito que ganhar uma Superfície já muda o Tom sozinho hoje.

## Considered Options

- **`section:` como quarto tipo de Elemento.** Previsível e local, e custaria o
  mesmo em tokens que `rect:` — é a mesma chave trocada, não uma chave a mais.
  Rejeitado porque acrescenta um termo ao vocabulário e uma decisão ao autor em
  cada bloco, para descrever algo que a engine já sabe.
- **`label-at: top|center`.** Rejeitado pelo mesmo motivo, agravado: é uma chave
  nova cuja única razão de existir é desfazer uma derivação.

## Consequences

- A faixa do Rótulo tem altura fixa em px do espaço do Frame, e não uma fração da
  altura do Retângulo. Sem isso, `fracaoDoRotulo` daria uma fonte de 180 px numa
  região de 400 px de altura. A regra de tamanho da ADR-0001 fica intocada: o que
  muda é a caixa que a resolução reserva.
- A derivação precisa da contenção, que hoje só é calculada em `atribuiElevacao`,
  depois do achatamento. A materialização do Elemento de Forma `Texto` passa a
  depender dessa passagem.
- O Rótulo não quebra linha: texto que não cabe é cortado, e o corte é
  denunciado. Um Rótulo que estoura o bloco no wireframe vai estourar o
  componente na UI, e acomodar isso com quebra automática esconderia um problema
  real de projeto.
- Só `rect` aceita `label:`. Num Círculo a faixa retangular no topo cairia fora
  da forma, e isso exigiria uma segunda regra de posição para o mesmo conceito.
