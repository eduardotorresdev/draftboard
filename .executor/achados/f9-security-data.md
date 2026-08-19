# Achados — F9 / security-data

1. **[quickwin, com decisão de contrato] `label` entra sem teto de comprimento.**
   `internal/schema/decode.go:386`. O rasterizador desenha a string inteira glifo
   a glifo mesmo quando a máscara da própria área descarta tudo; multiplicado por
   `repeat`, poucos KB de YAML viram minutos de CPU.
   Cenário medido: `rect {x:10,y:10,w:5,h:5}` com `label` de 200 000 caracteres e
   `repeat: {n:20}` → arquivo de 195 KB, `render` leva **61 s** e produz um WebP
   onde nenhum caractere aparece. Com 1 MB e `n:1000`, ~15 h projetadas. RSS fica
   em 18 MB: o custo é tempo, não memória. `board` não é afetado.
   Pré-existente para o `label:` de Controle, mas F9 põe a chave em todo `rect` e
   a torna repetível. Todos os outros tetos do módulo existem para fechar
   exatamente essa amplificação; o texto do Rótulo era a única entrada sem um.

2. **[quickwin] `escapa` deixa passar os caracteres de controle C0 para a
   Prancheta.** `internal/board/svg.go:213` é só `html.EscapeString`, que
   neutraliza os cinco caracteres de marcação e deixa passar os C0 crus.
   Cenário: um `label` com NUL e ESC no meio → o HTML da Prancheta contém os
   bytes crus; o arquivo deixa de ser texto válido, o parser HTML5 troca o NUL
   por U+FFFD e a Prancheta mostra uma coisa enquanto o WebP desenha `.notdef` —
   o mesmo Documento com dois desenhos, que é a divergência que o item 5b foi
   fechar. O `inspect` acerta, porque imprime com `%q`.
   Correção: em `escapa`, descartar ou substituir os C0 fora de tab/LF/CR antes
   do `html.EscapeString`. É o único ponto por onde texto do autor entra na
   Prancheta.

3. **[quickwin] `nota: %s` no `inspect` imprime texto do autor cru, e uma quebra
   de linha forja Elementos na árvore.** `internal/inspect/inspect.go:32`.
   Cenário: uma `note:` cujo texto contenha uma quebra de linha seguida de seis
   espaços e de algo com a cara de uma linha de Elemento — a árvore ganha uma
   linha indistinguível de um Elemento real para o agente que a lê. O `rotulo=`
   recém-acrescentado sai seguro com `%q`: a assimetria está dentro da mesma
   função.
   Correção: imprimir a Nota com `%q`, na mesma linha da chave. Atenção: isso
   muda a árvore, e portanto a gramática na SKILL.md e os goldens que a contêm.

## Verificado sem achado, com prova

- NaN e ±Inf nunca chegam à geometria: `numero` recusa em `decode.go:611`.
- Área zero, negativa e 1e300 com `label`: nenhum pânico; Rótulo de área ≤ 0 é
  descartado nos dois desenhistas; `alturaDoRotulo` satura a fonte em 12,6 px, o
  que evita o estouro de `limiteDeTracado` que os Rótulos de Controle ainda têm.
- Retângulo fora do Frame: o Rótulo é recortado duas vezes, não vaza.
- `label: ""` não materializa Elemento nem debita orçamento.
- Escape dos cinco caracteres de marcação correto no conteúdo e nos atributos.
- **O Rótulo paga o `LimiteDeElementos`**: 6000 rects passam, os mesmos 6000 com
  `label` estouram o teto com o erro certo.
- Unicidade de Caminho conferida com `id: a`, um id igual ao segmento do Rótulo e
  um `control` de mesmo id no mesmo Frame: seis caminhos distintos.
- Recusa de `label` em `circle`, `use` e `slot`, inclusive `label:` nulo.
- `Local` preenchido nos três pontos de construção, com prefixo correto
  atravessando `use` + `slots` + `repeat`.
- Nenhum id da Prancheta é controlado pelo autor.
