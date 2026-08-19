# Expectativas dos validadores — F11

Escrito antes da implementação. Depende do escopo, não do resultado.

## contract-behavior

O que tem que funcionar, com Documento concreto:

1. Retângulo de `w: 20` num Frame de 400 px com `label: "Configurações avançadas"`
   → `render` escreve a imagem, sai **0**, e o stderr traz um Aviso com um `w`
   em porcento. Aplicar esse `w` à mão e rodar de novo: **sem Aviso**. O número
   sugerido tem que consertar de primeira — arredondado para cima, nunca para
   baixo.
2. O mesmo Retângulo dentro de um Componente → **Erro**, imagem escrita mesmo
   assim, saída **1**. A mensagem não sugere `w`: ali alargar é juízo do autor.
3. Nota de 201 runas → Erro, saída 1, imagem escrita. 200 runas → nada. A conta
   é em **runas**: 201 caracteres acentuados não podem virar 400 bytes e
   disparar com 100 runas.
4. `inspect --fix` → arquivo reescrito, `w 20 → 47` no stderr, árvore
   **corrigida** no stdout. Rodar duas vezes: a segunda não muda byte nenhum e
   não imprime troca nenhuma.
5. `--fix` num Documento cujo único problema vem de Componente → **nenhum byte
   escrito**, saída 1.
6. `--fix` em `render`, `board` ou `validate` → erro de uso, com a mensagem no
   formato das outras opções desconhecidas.
7. Os quatro comandos conferem. `validate` sem `--fix` diagnostica e sai 1 sem
   escrever nada.
8. Erro antigo continua abortando: Componente ausente → nenhuma imagem escrita.
   Erro de F11 → imagem escrita. A diferença tem que ser observável, não
   declarada.
9. O diagnóstico não depende de `--scale`: o mesmo Documento com `--scale 1` e
   `--scale 8` dá exatamente os mesmos Avisos e Erros.
10. Retângulo alto o bastante para `TamanhoDoRotulo(A)` passar de `limiteDaFonte`
    (256 px): a medida ainda diz a verdade sobre o que a imagem mostra?

## security-data

1. `--fix` escreve no arquivo do autor. Escrita atômica? Um erro no meio deixa o
   YAML truncado? Permissões preservadas? Symlink seguido?
2. `Local` vem da resolução e endereça um nó. Um `Local` forjado — Documento com
   Componente que devolve `Local` com `../` ou com índice fora de faixa — faz
   `fix` escrever fora do arquivo pedido?
3. Rótulo com caracteres de controle, RTL override, combining marks: a medição
   entra em laço? A mensagem de Aviso, impressa com `%q`, escapa tudo? (F9 já
   fechou a árvore do `inspect` e o SVG da Prancheta; a mensagem nova é
   superfície nova.)
4. Custo: Documento com 10.000 Elementos rotulados — `Confere` mede 10.000
   textos. Quanto custa? A régua de escala 1 é criada uma vez ou por Elemento?
   Cada `Canvas` novo carrega um cache de faces.
5. `w` sugerido gigante: um Rótulo longuíssimo num Retângulo minúsculo sugere
   `w: 4000`? Aplicar isso estoura o teto de área no próximo `render`? O Aviso
   sugere um conserto que quebra outra coisa?
6. `--fix` com o arquivo somente-leitura, ou num diretório sem permissão: a
   mensagem é do domínio ou é o erro cru do sistema operacional?

## tests-maintenance

1. Cada categoria do item 16 tem teste que **morde**: trocar Aviso por Erro numa
   delas tem que ficar vermelho.
2. O arredondamento para cima tem teste próprio: trocar `Ceil` por `Round`
   quebra? O caso tem que estar calibrado num valor cuja fração seja pequena —
   um caso com fração 0,5 passa nos dois.
3. A regra "Erro de F11 não aborta" tem teste que confere o **arquivo escrito**,
   não só o código de saída.
4. `fix` tem teste com YAML em bloco, YAML em fluxo, e comentário na mesma linha
   do `w`.
5. Golden que mudar foi **olhado** e descrito no relatório.
6. Teste de idempotência do `--fix` (rodar duas vezes) existe e é o que prova o
   arredondamento na prática.
7. A régua de escala 1 é conferida contra o desenho real: um teste que meça na
   régua e conte tinta na imagem, senão a medida e a pintura podem divergir
   silenciosamente — foi exatamente esse tipo de divergência que F9 fechou entre
   a Prancheta e o WebP.
