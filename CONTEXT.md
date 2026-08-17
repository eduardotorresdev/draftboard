# Draftboard

Gerador de wireframes declarativos: lê YAML e produz imagens WebP em escala de
cinza, mais uma árvore textual da estrutura. Escrito para ser consumido por
agentes — o custo em tokens é critério de design de primeira classe.

## Language

### Documento

**Documento**:
Arquivo YAML que contém um ou mais Frames. É a unidade que se renderiza.
_Avoid_: projeto, arquivo de entrada, spec

**Frame**:
Viewport de dimensões declaradas em pixels que contém Camadas. Cada Frame vira
uma imagem separada.
_Avoid_: tela, canvas, viewport, artboard

**Camada**:
Grupo nomeado e ordenado de Elementos dentro de um Frame. Define ordem de
pintura, unidade de export isolado, e adiciona um degrau de Elevação sobre tudo
que está abaixo dela.
_Avoid_: layer, grupo, z-index

**Elemento**:
Forma posicionada num Frame. Existem três tipos: Retângulo, Círculo e Rótulo.
Retângulo e Círculo são sólidos — não existe contorno.
_Avoid_: shape, node, objeto, widget

**Retângulo**:
Elemento de quatro lados, com cantos retos por padrão e arredondados sob demanda.
_Avoid_: box, rect, quadrado, caixa

**Círculo**:
Elemento redondo definido por um único diâmetro. Nunca vira elipse.
_Avoid_: bola, ellipse, disco

**Nota**:
Anotação textual aninhada no Elemento que ela explica. Não faz parte do desenho:
não participa da Elevação nem do export por Camada, e some inteira quando
desligada na renderização.
_Avoid_: label, texto, comentário, callout, balão

### Geometria e cor

**Superfície**:
Área pintada sobre a qual um Elemento se apoia — o Frame, uma Camada anterior, ou
outro Elemento que o contém geometricamente.
_Avoid_: fundo, background, parent

**Elevação**:
Distância de um Elemento até o Frame, contada em Superfícies empilhadas. É a única
entrada do cálculo de Tom.
_Avoid_: profundidade, z, nível, depth

**Tom**:
Valor de cinza atribuído automaticamente a partir da Elevação. Nunca é declarado
no YAML. A escala vai de 100 (quase branco) a 900 (quase preto), em passos de 100.
_Avoid_: cor, shade, fill, tonalidade

**Chrome**:
Superfície ao redor do Frame onde as Notas são posicionadas quando renderizadas
na margem. Usa o extremo escuro da escala, reservado.
_Avoid_: margem, gutter, moldura, borda

### Reuso

**Componente**:
Arquivo YAML escrito num espaço de coordenadas próprio de 0 a 100, reutilizável
em qualquer tamanho. Não contém Frames.
_Avoid_: partial, template, include, macro

**Instância**:
Uso de um Componente dentro de um Frame ou de outro Componente, colocado numa
caixa que reescala o conteúdo.
_Avoid_: chamada, referência, uso

**Slot**:
Região retangular declarada por um Componente para receber conteúdo de quem o
instancia. Pode ter conteúdo padrão.
_Avoid_: buraco, placeholder, children, hole

**Repetição**:
Clonagem de um Elemento ou Instância ao longo de um eixo, com espaçamento fixo.
_Avoid_: loop, array, list, repeat

**Controle**:
Elemento composto do catálogo embutido no binário, invocado por nome em vez de
por arquivo. É fechado: não recebe Slot nem conteúdo, e a árvore mostra só a sua
cabeça e os parâmetros declarados. Materializa-se em vários Elementos antes da
Elevação e, como Instância e Repetição, não existe mais no Documento resolvido.
_Avoid_: widget, componente nativo, primitiva, bloco

**Rótulo**:
Texto desenhado dentro de um Controle. Distingue-se da Nota por estar no plano
do desenho: participa da Elevação e aparece no export por Camada. Não é
Superfície — nada se apoia sobre ele.
_Avoid_: label, caption, legenda, texto

### Distribuição

**Versão**:
Identificação do binário instalado, no formato de uma tag `vX.Y.Z`. Binário
construído sem essa informação se identifica como `dev`.
_Avoid_: release local, build, número

**Lançamento**:
Conjunto publicado no GitHub sob uma tag: os Ativos de cada plataforma mais o
arquivo de somas.
_Avoid_: release, publicação, tag

**Ativo**:
Arquivo de um Lançamento — o `.tar.gz` de uma plataforma, ou o arquivo de somas.
_Avoid_: asset, artefato, pacote, tarball
