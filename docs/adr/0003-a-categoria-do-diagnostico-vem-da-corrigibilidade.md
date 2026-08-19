# 3. A categoria do diagnóstico vem da corrigibilidade, não do impacto

Data: 2026-08-19

## Status

Aceita.

## Contexto

Até aqui o Draftboard tinha duas categorias com significados convencionais:
Aviso é o problema tolerável, Erro é o que impede a resolução. A distinção era de
impacto, e funcionava porque todo Erro era de fato intransponível — um arquivo
ilegível, um Componente ausente, uma chave desconhecida.

O Rótulo que não cabe no Retângulo quebra isso. É sempre o mesmo problema, com o
mesmo impacto — o texto é cortado, o desenho sai —, mas a saída depende de onde
ele foi declarado. Num Retângulo escrito direto no Documento, uma máquina
consegue calcular o `w` mínimo e aplicar. Dentro de um Componente reusado por
várias Instâncias em tamanhos diferentes, não existe correção mecânica: alargar
o Componente mexe em todos os usos, alargar o `box` de uma Instância infla um
bloco inteiro por causa de uma palavra, e encurtar o texto exige julgamento.

O mesmo vale para a Nota acima do teto de 200 caracteres: nenhuma máquina
encurta um texto.

## Decisão

A categoria passa a ser definida pela corrigibilidade mecânica.

- **Aviso**: a máquina resolve. Não faz o comando falhar, e a mensagem carrega o
  valor que a correção aplicaria — que é também o que o `inspect --fix` aplica.
- **Erro**: exige julgamento do autor. Faz o comando falhar.

Como consequência direta, o Erro deixa de implicar aborto. Um Rótulo que
transborda dentro de um Componente imprime como erro e faz `render` sair com
código 1, **mas as imagens são escritas assim mesmo** — inclusive as dos Frames
que estavam corretos. Os Erros antigos, os que impedem a resolução de
prosseguir, continuam abortando: a diferença não é de categoria, é de se há ou
não desenho a produzir.

## Considered Options

- **Manter tudo como Aviso, e sair com código 1 quando algum for incorrigível.**
  Rejeitado: a diferença entre "conserte com `--fix`" e "pense" ficaria visível
  só no código de saída, que é justamente o canal que o autor não lê.
- **Usar o `scene.Erro` existente, que aborta.** Rejeitado: um `label` longo num
  Componente reusado derrubaria o Documento inteiro, apagando a saída de Frames
  que não têm problema nenhum.

## Consequences

- Medir "cabe ou não cabe" exige métrica de fonte, e a ADR-0001 mantém freetype
  fora de quem só calcula geometria. Este diagnóstico não pode nascer na
  resolução: nasce no plano do desenho e sobe até o comando.
- `render` passa a poder sair com código 1 tendo escrito todas as imagens. Quem
  usa o código de saída como porta de CI ganha o comportamento certo de graça;
  quem o usa como "a saída existe?" precisa mudar.
- O `LimiteDeAvisos = 1_000` que já existe passa a valer também para estes.
