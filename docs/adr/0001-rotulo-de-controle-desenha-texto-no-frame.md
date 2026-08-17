# 1. O Rótulo de Controle desenha texto real dentro do Frame

Data: 2026-08-17

## Status

Aceita. Revoga parcialmente uma decisão de escopo do `docs/PRD.md`.

## Contexto

O PRD lista, entre o que está explicitamente fora de escopo:

> **Texto dentro de formas.** Texto placeholder é um Retângulo fino; texto real
> só existe em Nota. Manter isso separado é o que impede o wireframe de virar
> mockup.

A introdução dos Controles reabre a questão. Um `button` sem rótulo é um
retângulo arredondado; um `tabs` sem rótulo é uma fileira de barras cinza. As
duas coisas são desenháveis com as primitivas que já existiam, e as duas
comunicam menos do que o autor quis dizer: um wireframe cuja barra de abas não
diz "Perfil, Conta, Faturas" obriga o leitor a inferir, ou a ler a margem.

O Rótulo poderia ter virado Nota, que já existe e já renderiza texto — mas a
Nota vive no plano de anotação, sai com `--notes off`, some no export por Camada
e é ancorada na margem. Um nome de aba não é um comentário sobre o desenho: é o
desenho.

## Decisão

O `scene.Elemento` ganha a Forma `Texto` e os campos que a sustentam. Só a
resolução de um Controle a produz.

As linhas de defesa que o PRD tinha nesse ponto são substituídas por outras,
mais estreitas, e que ficam registradas aqui como parte da decisão:

- **Não existe nó `text:`** no YAML. Texto solto continua sendo Retângulo fino.
- **Não existe campo de tamanho de fonte.** O tamanho é derivado da altura da
  área, como o Tom é derivado da Elevação.
- **Não existe campo de alinhamento, peso ou família.** O alinhamento é
  propriedade do Controle, não do autor.
- **O Rótulo não é Superfície.** Nada se apoia sobre ele para efeito de
  Elevação.

## Consequências

- `internal/scene/` deixou de ter duas Formas e passou a ter três. É a primeira
  emenda ao pacote congelado desde o início do projeto.
- `Forma.String()` deixou de ter ramo padrão implícito: uma Forma nova agora
  aparece na árvore em vez de se disfarçar de Retângulo.
- A resolução continua sem conhecer métrica de fonte. Ela entrega ao
  rasterizador a **área** que o Rótulo deve ocupar, e não a caixa das glifas —
  sem isso, `freetype` viraria dependência de quem só calcula geometria.
- Rótulo que não cabe na área é recortado nela. Não há aviso: a resolução, que é
  quem emite avisos, é justamente quem não sabe medir texto.
- A pressão seguinte previsível é `size:`, `align:` e depois `color:`. A
  resposta é a mesma que o PRD dá para cor: ajustar a estrutura, não acrescentar
  o botão.
