# 4. O modo Margem é aposentado e as Notas saem do padrão

Data: 2026-08-19

## Status

Aceita. Consequência da ADR-0002.

## Contexto

Antes do Rótulo aberto, a Nota fazia dois trabalhos: dizia o que cada bloco era e
explicava o que ele fazia. O modo Margem existia para o primeiro — com uma dúzia
de blocos anônimos, era preciso um texto por bloco, e um texto por bloco só cabe
num Chrome que cresce. O preço era uma linha de chamada por Nota atravessando o
desenho, e ler o wireframe virava seguir linhas com o dedo.

Com o Rótulo nomeando o bloco por dentro, o primeiro trabalho some. Sobram as
Notas explicativas, que são poucas e ficam melhor perto do que explicam.

O modo Flutuante já existia e já fazia isso, mas assumia de propósito duas
fraquezas: balões vizinhos podiam se sobrepor, e texto largo demais era cortado,
porque a tela tem o tamanho do Frame e não cresce. Era exatamente por causa
delas que o Margem era o padrão e a única garantia de leitura completa.

## Decisão

O modo Margem é removido. Com ele saem o Chrome que cresce e a linha de chamada
longa; o Chrome deixa de existir como conceito, e `TomChrome` sobrevive apenas
como o Tom do balão.

As duas fraquezas do Flutuante deixam de ser aceitas: ganha anti-colisão entre
balões, e a Nota ganha teto de 200 caracteres — cerca de quatro linhas na largura
que já existe. O teto é constante no binário, pela mesma razão que o Tom e o
corpo da fonte são: o autor precisa saber o limite enquanto escreve, e um limite
derivado do espaço livre passaria num Frame e falharia noutro.

`--notes` vira booleana e o padrão passa a ser **sem Notas**: a imagem sai limpa,
com os blocos se explicando pelo Rótulo, e a anotação é opt-in. `--notes float`,
`--notes margin` e `--notes off` passam a ser erro de uso com mensagem dizendo o
que mudou.

## Considered Options

- **Manter o Margem como saída para Frame muito cheio.** Rejeitado: com o teto de
  caracteres, o caso que ele salvava deixa de existir, e sobraria um modo raro
  para manter, testar e documentar.
- **Fazer a tela do Flutuante crescer além do Frame.** Daria a garantia de caber
  sem teto de caracteres, mas reescreveria o layout do Flutuante para recuperar
  justamente o Chrome que estamos removendo.

## Consequences

- A leitura completa de todas as Notas de uma vez deixa de existir na imagem. Ela
  passa a ser da Prancheta HTML, que já tem o botão "Notas" e a inspeção.
- Todo Documento existente renderiza diferente por omissão: as Notas somem da
  imagem até que `--notes` seja passado.
- A SKILL.md embutida é o que os agentes leem sobre a CLI, e ela documenta os
  três modos. Precisa mudar junto, ou os agentes vão gerar comandos inválidos.
