# PRD — Draftboard

## Problem Statement

Quando um agente precisa comunicar layout — a estrutura de uma tela, onde cada
bloco de informação mora, como um modal se sobrepõe ao conteúdo — ele não tem
ferramenta adequada. As opções existentes falham por motivos diferentes e todas
caras:

- **Descrever em prosa** é ambíguo. "Um card no topo com avatar à esquerda" não
  fixa proporção, alinhamento nem hierarquia, e o leitor humano reconstrói algo
  diferente do que o agente imaginou.
- **Gerar HTML/CSS** funciona, mas custa muitos tokens, exige um navegador pra
  virar imagem, e arrasta decisões de estilo (fonte, cor, sombra, espaçamento)
  que não são a questão em discussão.
- **Ferramentas de design** (Figma e afins) exigem sessão interativa,
  autenticação e um modelo de documento grande demais pra descrever um retângulo
  cinza.
- **Gerar imagem por modelo de difusão** produz algo que *parece* um wireframe
  mas não é determinístico, não é editável e não tem estrutura consultável.

O resultado é que discussões de layout acontecem sem artefato visual, ou com
artefato caro demais pra iterar. E quando existe imagem, ela é opaca: pra saber
o que tem nela, é preciso gastar tokens olhando o pixel.

## Solution

Um binário único, `draftboard`, que lê um wireframe declarativo em YAML e produz
duas saídas complementares:

1. **Imagens WebP** — uma por Frame, em escala de cinza, prontas pra colar numa
   conversa, num PR ou num documento.
2. **Uma árvore textual no stdout** — a mesma informação, resolvida e legível
   por agente sem custo de visão.

O YAML é desenhado pra ser escrito por um agente: chaves curtas, posicionamento
absoluto em porcentagem, defaults agressivos e — a decisão central — **nenhuma
declaração de cor**. O tom de cinza de cada Elemento é derivado automaticamente
da sua Elevação, então o agente descreve *estrutura* e recebe *contraste
correto* de graça. Não existe como gerar um wireframe ilegível por combinação
ruim de cinzas.

Componentes e Slots dão reuso real: um Componente é escrito num espaço de
coordenadas próprio e reescala pra qualquer caixa onde for instanciado, então o
mesmo `card.yaml` serve num sidebar estreito e num hero largo. Slots permitem
layout reaproveitável — um `app-shell.yaml` com slots `nav` e `main` que cada
tela preenche.

Notas aninhadas nos Elementos anotam o desenho sem fazer parte dele, e o modo de
exibição é escolhido na renderização, não no documento — o mesmo arquivo gera a
versão limpa e a versão anotada.

## User Stories

### Renderização básica

1. Como agente, quero renderizar um arquivo YAML para WebP com um comando, para
   que eu produza um wireframe sem precisar de navegador ou serviço externo.
2. Como agente, quero que um Documento com vários Frames gere uma imagem por
   Frame, para que eu descreva um fluxo inteiro num arquivo só.
3. Como agente, quero que o nome do arquivo de saída derive do nome do Documento
   e do Frame, para que eu saiba de onde cada imagem veio sem abrir nenhuma.
4. Como agente, quero escolher o diretório de saída, para que as imagens não
   poluam a pasta dos meus YAMLs.
5. Como agente, quero que as dimensões declaradas no Frame sejam os pixels da
   imagem, para que não exista surpresa de DPI ou reescala.
6. Como agente, quero multiplicar a resolução com um fator de escala, para que
   eu gere uma versão maior quando outro agente for ler a imagem por visão.
7. Como usuário, quero que o binário não dependa de bibliotecas nativas, para
   que eu instale copiando um arquivo e rode em qualquer plataforma.
8. Como agente, quero que a renderização seja determinística, para que rodar
   duas vezes o mesmo arquivo produza bytes idênticos e o diff em Git seja
   confiável.

### Fluxo entre telas

Um Documento com vários Frames descreve um fluxo, mas a imagem por Frame não
mostra o fluxo — mostra as telas soltas. As duas histórias abaixo cobrem essa
lacuna e revertem, deliberadamente, a exclusão de "links entre Frames" que o
escopo original carregava.

- Como agente, quero declarar que um Elemento leva a um Frame (`to`), para que o
  gatilho de cada transição fique escrito ao lado do que o dispara.
- Como agente, quero que a árvore do `inspect` diga para onde cada gatilho leva,
  para que eu leia a navegação inteira sem gastar visão.
- Como usuário, quero uma superfície única com todas as telas e as setas entre
  elas — a Prancheta —, para que eu discuta o fluxo sem abrir dez imagens.
- Como usuário, quero navegar essa superfície com pan e zoom e clicar num
  Elemento para ver o que ele é, para que a imagem deixe de ser opaca.
- Como usuário, quero que a Prancheta seja um arquivo só que abre sem servidor e
  sem rede, para que eu a mande por anexo como mando a imagem.
- Como agente, quero que a Ligação não mexa no desenho, para que acrescentar
  fluxo a um Documento não mude nenhuma imagem já gerada.

### Geometria

9. Como agente, quero posicionar todo Elemento em porcentagem do Frame, para que
   eu não precise fazer aritmética de pixels.
10. Como agente, quero que `x,y` ancore sempre no canto superior esquerdo de
    qualquer Elemento, para que eu não tenha exceções por tipo de forma.
11. Como agente, quero declarar um Retângulo com posição e dimensões, para que
    eu construa a maior parte de qualquer wireframe.
12. Como agente, quero ligar cantos arredondados num Retângulo com um booleano,
    para que eu sugira o estilo sem gastar decisão em raio.
13. Como agente, quero que o raio arredondado seja constante em toda a tela e
    limitado pelo tamanho do Elemento, para que Retângulos pequenos não virem
    pílulas acidentalmente.
14. Como agente, quero declarar um Círculo com um único diâmetro, para que ele
    saia sempre redondo, independentemente da proporção do Frame.
15. Como agente, quero que Elementos que ultrapassam o Frame sejam recortados em
    vez de rejeitados, para que eu represente conteúdo cortado na borda.
16. Como agente, quero que a ordem de declaração determine a ordem de pintura
    dentro de uma Camada, para que sobreposição seja previsível.

### Cor automática

17. Como agente, quero não declarar cor nenhuma, para que eu gaste tokens
    descrevendo estrutura e não estilo.
18. Como agente, quero que o Tom de cada Elemento venha da sua Elevação, para
    que a hierarquia visual reflita a hierarquia estrutural automaticamente.
19. Como agente, quero que a Elevação seja inferida por contenção geométrica,
    para que eu mantenha uma lista plana de Elementos sem aninhar YAML.
20. Como agente, quero que Superfícies adjacentes sempre difiram por dois
    degraus da escala, para que o contraste sobreviva a compressão e a leitura
    por visão.
21. Como agente, quero que a escala reverta de direção ao chegar no extremo, em
    vez de saturar, para que aninhamento profundo continue legível.
22. Como agente, quero que a escala de cinza siga a convenção de Tailwind (100
    claro, 900 escuro), para que meu instinto acerte sem consultar documentação.
23. Como usuário, quero que o fundo do Frame seja sempre o mesmo tom, para que
    wireframes diferentes sejam visualmente comparáveis.

### Camadas

24. Como agente, quero agrupar Elementos em Camadas nomeadas, para que a
    estrutura do Frame tenha significado além da ordem.
25. Como agente, quero que uma Camada superior adicione um degrau de Elevação
    sobre tudo abaixo, para que um modal se destaque do conteúdo sem eu declarar
    nada.
26. Como agente, quero exportar uma imagem por Camada, para que eu mostre a tela
    sendo montada em etapas.
27. Como agente, quero que o export por Camada seja cumulativo, para que os tons
    de cada imagem batam com o render final.
28. Como agente, quero que o nome do arquivo do export por Camada inclua o nome
    da Camada, para que a sequência seja óbvia na listagem do diretório.

### Notas

29. Como agente, quero aninhar uma Nota no Elemento que ela explica, para que a
    âncora seja implícita e eu não precise inventar identificadores.
30. Como agente, quero que a Nota seja posicionada automaticamente, para que eu
    não gaste decisão nem tokens escolhendo onde ela cabe.
31. Como agente, quero que a Nota seja ligada por uma linha de chamada ao seu
    Elemento, para que a associação seja inequívoca mesmo com muitas notas.
32. Como agente, quero renderizar as Notas numa margem ao redor do Frame, para
    que elas nunca ocultem o desenho.
33. Como agente, quero renderizar as Notas flutuando sobre o desenho, perto da
    âncora, para que eu gere uma versão compacta quando o espaço importa.
34. Como agente, quero desligar as Notas na renderização, para que eu gere a
    versão limpa do mesmo arquivo sem editá-lo.
35. Como agente, quero que o modo de Nota seja opção da linha de comando e não
    do Documento, para que eu produza as três versões sem tocar no YAML.
36. Como usuário, quero que o Chrome ao redor do Frame use um tom reservado, para
    que a fronteira do Frame seja inconfundível.
37. Como agente, quero que a margem cresça o quanto for preciso pra caber as
    Notas, para que texto longo nunca seja truncado.
38. Como agente, quero que o texto da Nota quebre em várias linhas, para que eu
    escreva uma frase inteira sem estourar o layout.

### Componentes e reuso

39. Como agente, quero extrair um pedaço recorrente do wireframe pra um
    Componente, para que eu não repita a mesma geometria em várias telas.
40. Como agente, quero escrever o Componente num espaço de coordenadas próprio
    de 0 a 100, para que ele seja independente do Frame onde será usado.
41. Como agente, quero que a Instância reescale o conteúdo do Componente pra
    caixa onde foi colocada, para que o mesmo Componente sirva em tamanhos
    diferentes.
42. Como agente, quero que Círculos dentro de uma Instância continuem redondos,
    para que a reescala não os distorça.
43. Como agente, quero instanciar um Componente dentro de outro Componente, para
    que eu componha peças pequenas em peças maiores.
44. Como agente, quero declarar Slots num Componente, para que ele funcione como
    layout reaproveitável em vez de bloco fechado.
45. Como agente, quero preencher um Slot com outro Componente, para que eu monte
    telas combinando peças existentes.
46. Como agente, quero preencher um Slot com Elementos inline, para que conteúdo
    usado uma vez só não me obrigue a criar arquivo.
47. Como agente, quero que o Slot preenchido vire um novo espaço de coordenadas
    de 0 a 100, para que o conteúdo injetado seja escrito sem saber onde vai
    parar.
48. Como agente, quero declarar conteúdo padrão num Slot, para que o Componente
    renderize sozinho e sirva de exemplo.
49. Como agente, quero que um Slot não preenchido e sem padrão vire uma
    Superfície vazia visível, para que eu enxergue o buraco em vez de perder o
    layout.
50. Como agente, quero referenciar Componentes por caminho relativo ao arquivo
    que os usa, para que mover uma pasta inteira não quebre nada.
51. Como agente, quero que o tipo do arquivo seja inferido pelo seu conteúdo,
    para que eu não escreva declaração de tipo em todo arquivo.
52. Como agente, quero que a Elevação atravesse a fronteira do Componente, para
    que o contraste seja calculado contra a Superfície real onde ele foi
    colocado, e não contra o contexto onde foi escrito.

### Repetição

53. Como agente, quero repetir um Elemento N vezes ao longo de um eixo com
    espaçamento fixo, para que uma lista de oito itens custe uma linha de YAML.
54. Como agente, quero repetir uma Instância de Componente, para que um feed de
    cards seja tão barato quanto um retângulo.
55. Como agente, quero escolher o eixo da Repetição, para que eu monte tanto
    listas verticais quanto barras de ações horizontais.
56. Como agente, quero que cada clone da Repetição calcule sua própria Elevação
    normalmente, para que o resultado seja idêntico a ter escrito tudo à mão.

### Inspeção

57. Como agente, quero imprimir a árvore resolvida do Documento no stdout, para
    que eu entenda a tela inteira sem gastar tokens abrindo imagem.
58. Como agente, quero que a árvore mostre Frames, Camadas e Elementos
    hierarquicamente, para que a estrutura seja aparente na indentação.
59. Como agente, quero ver a geometria já resolvida em cada linha, para que eu
    saiba exatamente onde cada Elemento está sem simular a reescala de cabeça.
60. Como agente, quero ver o Tom calculado de cada Elemento, para que eu
    verifique o contraste sem olhar a imagem.
61. Como agente, quero ver de qual Componente cada Elemento veio, para que eu
    saiba qual arquivo editar.
62. Como agente, quero ver as Notas na árvore, para que a descrição textual seja
    autossuficiente.
63. Como agente, quero que a inspeção não escreva nada em disco, para que eu a
    use livremente durante a iteração.
64. Como agente, quero que Elementos sem identificador ganhem um caminho estável
    na árvore, para que eu me refira a eles sem precisar batizar tudo.

### Validação e erros

65. Como agente, quero que um campo desconhecido seja erro, para que meu chute
    errado apareça imediatamente em vez de sumir silenciosamente.
66. Como agente, quero receber a sugestão do campo válido mais próximo, para que
    eu corrija o erro na primeira tentativa.
67. Como agente, quero que Componente inexistente seja erro, para que caminho
    errado não vire tela vazia.
68. Como agente, quero que ciclo de referência entre Componentes seja erro
    explícito, para que eu não trave o processo.
69. Como agente, quero que aninhamento excessivo seja erro com limite claro,
    para que recursão acidental falhe rápido.
70. Como agente, quero que Elemento fora do Frame seja apenas aviso, para que
    corte intencional na borda continue possível.
71. Como agente, quero que Slot vazio seja apenas aviso, para que eu veja o
    layout parcial enquanto construo.
72. Como agente, quero validar sem renderizar, para que eu cheque a sintaxe na
    forma mais barata possível.
73. Como agente, quero que a validação saia com código diferente de zero em caso
    de erro, para que ela funcione em hook e CI.
74. Como agente, quero que a mensagem de erro aponte arquivo e localização, para
    que eu vá direto no ponto em cadeias de Componentes.

### Skill

75. Como agente, quero uma skill compacta ensinando a CLI e o formato YAML, para
    que eu produza wireframes válidos sem tentativa e erro.
76. Como usuário, quero instalar a skill com um subcomando do binário, para que
    eu não copie arquivo à mão.
77. Como usuário, quero que a skill venha embutida no binário, para que ela
    nunca fique dessincronizada da versão instalada.
78. Como usuário, quero imprimir a skill no stdout, para que eu a inspecione ou
    redirecione pra onde eu quiser.

## Implementation Decisions

### Linguagem e dependências

- Go, binário único, sem cgo. Isso é requisito, não preferência: viabiliza
  distribuição por cópia de arquivo e cross-compilação trivial.
- Rasterização vetorial via `github.com/fogleman/gg` (Go puro).
- Codificação WebP via `github.com/HugoSmits86/nativewebp` (Go puro, WebP
  lossless / VP8L). Lossless é a escolha certa: cinza chapado comprime pra quase
  nada e a borda não borra.
- Fonte das Notas via `golang.org/x/image/font/gofont/goregular` — pacote Go, sem
  asset externo pra embutir.
- A skill é embutida com `go:embed`.

### Modelo do documento

- Um **Documento** contém uma lista de **Frames**. Um **Frame** declara
  dimensões em pixels e contém **Camadas** ordenadas. Uma **Camada** contém uma
  lista plana de **Elementos**.
- Tipos de Elemento: Retângulo e Círculo. Toda forma é sólida — não existe
  contorno nem preenchimento opcional. Um anel se expressa como Círculo com
  Círculo menor por cima, que a Elevação já colore por contraste.
- Retângulo: posição e dimensões em porcentagem, mais um booleano de canto
  arredondado. Círculo: posição mais um único diâmetro em porcentagem da largura
  do Frame.
- Âncora de posição é sempre o canto superior esquerdo da bounding box,
  inclusive para Círculo. Regra única, sem exceção por tipo.
- Identificador de Elemento é opcional; existe só pra legibilidade da inspeção.
  Sem ele, a inspeção gera um caminho estável a partir da posição na árvore.
- Toda medida é porcentagem do eixo correspondente do Frame — exceto o diâmetro
  do Círculo, que usa a largura em ambos os eixos pra nunca virar elipse.

### Cor derivada

- Não existe campo de cor em nenhum nível do YAML. O fundo do Frame é fixo.
- Escala de nove tons, de quase branco a quase preto, na direção de Tailwind
  (menor = mais claro). A direção contraria a intuição de "número maior = mais
  claro", e foi escolhida deliberadamente contra ela: o autor primário é um
  agente com prior forte de Tailwind, e alinhar elimina a classe de erro mais
  provável.
- **Elevação** de um Elemento = profundidade da sua Superfície. A Superfície-pai
  é o último Elemento pintado antes dele cuja bounding box o contém; se nenhum
  contiver, é o Frame.
- Cada Camada acima da primeira adiciona um degrau de Elevação sobre tudo que
  está abaixo, mesmo sem contenção geométrica. É isso que faz uma Camada de
  modal se destacar automaticamente.
- Cada nível de Elevação desloca o Tom em dois degraus da escala. Ao não haver
  espaço pra mais um passo, a direção do deslocamento inverte. Isso garante que
  Superfícies adjacentes sempre difiram e que a escala nunca esgote, a qualquer
  profundidade.
- O Chrome usa o extremo escuro da escala, reservado — a escada de Elevação
  nunca alcança esse tom, então chrome e desenho jamais se confundem.

### Componentes, Slots e Repetição

- Um **Componente** é um arquivo YAML sem Frames, escrito num espaço local de 0 a
  100 em ambos os eixos.
- Uma **Instância** coloca um Componente numa caixa do espaço do pai; o conteúdo
  é mapeado proporcionalmente pra dentro dela. Os fatores de escala X e Y são
  independentes (a caixa pode ter proporção diferente do quadrado local), mas o
  diâmetro do Círculo usa apenas o fator de largura, preservando a redondeza.
- Um **Slot** é uma região retangular declarada no espaço local do Componente.
  Quem instancia preenche com uma referência a outro Componente ou com Elementos
  inline. A região do Slot se torna um novo espaço local de 0 a 100.
- Slot pode declarar conteúdo padrão. Sem preenchimento e sem padrão, renderiza
  uma Superfície vazia com o degrau de Elevação e emite aviso.
- Referências resolvem relativas ao arquivo que referencia, nunca ao diretório de
  trabalho. Não há registry, diretório de biblioteca nem configuração global.
- O tipo do arquivo é inferido pelas chaves presentes: com Frames é Documento,
  caso contrário é Componente. Não existe campo declarando o tipo.
- Aninhamento tem limite rígido de profundidade e detecção de ciclo; ambos são
  erro.
- A resolução é achatamento total: Instâncias e Repetições são materializadas em
  Elementos com geometria absoluta antes de qualquer cálculo de Elevação. A
  Elevação portanto atravessa fronteiras de Componente naturalmente, porque nesse
  ponto elas não existem mais.
- **Repetição** é uma propriedade de um Elemento ou Instância: quantidade, eixo e
  espaçamento. Clones são materializados na mesma fase de achatamento, então se
  comportam como se tivessem sido escritos à mão.

### Notas

- Nota é uma propriedade do Elemento que ela anota — a âncora é implícita e não
  requer identificador. Consequência aceita: não é possível anotar uma região
  vazia sem Elemento.
- Nota não participa da Elevação e não aparece no export por Camada. São dois
  planos separados: desenho e anotação.
- O modo de exibição é opção de linha de comando, nunca do Documento: margem
  (padrão), flutuante, ou desligado. O YAML descreve o que existe; a CLI decide
  como mostrar.
- No modo margem, a tela de saída é maior que o Frame: o Chrome envolve o Frame e
  as Notas são empilhadas nele, com linha de chamada até o Elemento. O
  posicionamento é automático — a resolução de colisão é essencialmente
  unidimensional, ordenando por altura da âncora, o que torna o algoritmo simples
  e o resultado estável entre edições.
- No modo flutuante, a Nota é posicionada perto da âncora, sobre o desenho, e a
  tela mantém as dimensões do Frame.
- O texto quebra em múltiplas linhas com largura máxima fixa; a margem cresce
  pra acomodar.

### Interface de linha de comando

Três verbos, três intenções:

- **render** — grava as imagens e imprime apenas os caminhos escritos. Aceita
  diretório de saída, fator de escala, modo de Nota e export por Camada.
- **inspect** — imprime a árvore resolvida no stdout e não toca em disco.
- **validate** — checa e sai com código diferente de zero em caso de erro.

Mais um subcomando auxiliar para imprimir ou instalar a skill embutida.

- Nome da imagem deriva do Documento e do Frame; no export por Camada, inclui
  também o nome da Camada.
- Export por Camada é cumulativo: cada imagem contém a Camada e todas abaixo.
  Isolado produziria tons divergentes do render final, já que a Elevação depende
  do que está embaixo.
- As dimensões declaradas no Frame são os pixels da imagem; o fator de escala as
  multiplica. Raio de canto e espessura de linha de chamada escalam junto.

### Validação

- Estrito em schema: campo desconhecido, tipo inválido, Componente inexistente,
  ciclo, e profundidade excedida são erros. Campo desconhecido reporta o campo
  válido mais próximo.
- Tolerante em geometria: Elemento fora do Frame é recortado com aviso; Slot
  vazio e Elemento de área zero são avisos.
- Erros carregam arquivo e localização, atravessando a cadeia de Componentes.

### Saída de inspeção

A árvore é o artefato que sustenta a promessa de eficiência de token: um agente
precisa entender a tela sem abrir a imagem. Cada linha carrega identificador ou
caminho, tipo, geometria já resolvida, Tom calculado, Componente de origem
quando aplicável, e a Nota quando houver. A hierarquia Frame → Camada → Elemento
aparece na indentação.

Consequência de design importante: a saída de `inspect` é a **projeção
observável de toda a fase de resolução**. Componentes, Slots, Repetição,
Camadas, Elevação e Tom são todos verificáveis por ela.

## Testing Decisions

Um bom teste aqui exercita comportamento externo: dado um conjunto de arquivos
YAML, invocar a CLI e asserir o que ela escreveu no stdout e em disco. Nenhum
teste deve conhecer a estrutura interna da cena resolvida, os nomes das fases do
pipeline, ou como a Elevação é computada — apenas o resultado observável.

Não há prior art: o repositório está vazio, este é o primeiro código. As
convenções abaixo passam a ser a prior art.

### Seam primário — a CLI

O seam principal é o próprio comando: executar com argumentos apontando pra um
diretório de fixtures e asserir stdout, código de saída e arquivos escritos.

Como a saída de `inspect` projeta toda a resolução, golden files dessa árvore
cobrem a semântica inteira sem nenhum seam interno:

- Geometria: âncora, conversão de porcentagem, recorte na borda, diâmetro de
  Círculo em Frame não-quadrado.
- Elevação e Tom: contenção geométrica, degrau por Camada, passo de dois níveis,
  reversão no extremo da escala, tom reservado do Chrome.
- Componentes: reescala em caixa de proporção diferente, redondeza preservada,
  aninhamento de Componente em Componente, Elevação atravessando fronteira.
- Slots: preenchimento por Componente, por Elementos inline, conteúdo padrão, e
  Slot vazio.
- Repetição: contagem, eixo, espaçamento, e Elevação de cada clone.
- Erros: campo desconhecido com sugestão, Componente inexistente, ciclo,
  profundidade excedida — asserindo mensagem e código de saída.
- Avisos: fora do Frame, Slot vazio, área zero — asserindo que a renderização
  ocorre mesmo assim.

Golden files de texto têm a propriedade que se quer aqui: o diff de uma mudança
de comportamento é legível na revisão.

### Seam secundário — rasterização

Um seam estreito para cena resolvida → imagem, com golden de bytes. Existe para
que uma falha de renderização não seja confundida com falha de resolução. Cobre:

- Determinismo: a mesma entrada produz bytes idênticos.
- Canto arredondado ligado e desligado, e o limite de raio em Elemento pequeno.
- Fator de escala produzindo dimensões proporcionais.
- Os três modos de Nota, incluindo o crescimento da margem com texto longo.
- Export por Camada cumulativo.
- Validade do arquivo WebP e as dimensões de saída esperadas.

### O que não se testa

Não se escreve teste para o formato interno da cena, para funções de conversão
isoladas, nem para a ordem das fases do pipeline. São detalhes de implementação
e testá-los congela decisões que devem permanecer livres.

## Out of Scope

- **Outros formatos de imagem** (PNG, SVG, PDF) como export de Frame. A imagem de
  um Frame é WebP e só. O SVG da Prancheta não é exceção a isso: ele é o meio de
  desenhar dentro do HTML, não um arquivo de imagem que se possa pedir por
  Frame.
- **Auto-layout** (stack, flex, grid, alinhamento automático). Contradiz o
  princípio de posicionamento absoluto e é a maior fonte de complexidade
  possível neste domínio.
- **Texto dentro de formas.** Texto placeholder é um Retângulo fino; texto real
  só existe em Nota. Manter isso separado é o que impede o wireframe de virar
  mockup.
- **Ícones e imagens.**
- **Cores fora da escala de cinza**, temas, e qualquer controle direto de cor.
- **Estados interativos** — um Elemento não tem estado que mude por interação, e
  a Prancheta navega entre telas prontas em vez de simular o comportamento
  delas.
- **Modo watch** e qualquer observação de sistema de arquivos.
- **Contorno / stroke** em qualquer forma. Superfícies se separam por contraste.
- **Elipse** como primitiva.
- **Diretório de biblioteca de Componentes** ou registry — só caminho relativo.
- **Anotar região sem Elemento** — consequência aceita de aninhar a Nota no
  Elemento.

## Further Notes

O vocabulário do domínio está em `CONTEXT.md` na raiz. Os termos ali (Frame,
Camada, Elemento, Superfície, Elevação, Tom, Chrome, Componente, Instância,
Slot, Repetição, Nota) devem ser usados na implementação, nas mensagens de erro
e na skill. Consistência de vocabulário entre a documentação e a saída da
ferramenta é parte do que a torna barata de aprender.

Duas decisões merecem destaque por serem contraintuitivas e deliberadas:

**A escala de cinza é invertida em relação à formulação original.** O pedido
inicial definia 100 como quase preto. Foi invertida pra alinhar com Tailwind,
porque o autor primário do YAML é um agente com esse prior — a intuição interna
("número maior = mais claro") perde pra intuição do consumidor real.

**Não existe declaração de cor.** Isso parece uma limitação e é o oposto: é a
decisão que garante que nenhum wireframe gerado seja ilegível, e que remove uma
categoria inteira de decisão do agente. Se em algum momento surgir pressão pra
adicionar controle de cor, a resposta correta é quase sempre ajustar a estrutura
(Camada, aninhamento) e deixar o algoritmo derivar o contraste.

Também vale registrar que a saída de `inspect` não é diagnóstico secundário — é
uma das duas saídas do produto, com o mesmo peso da imagem. Ela é o que torna o
wireframe consultável por agente a custo baixo, e é também o que torna toda a
semântica testável pela CLI.
