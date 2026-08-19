# Expectativas de validação — F9

Escrito **antes** do resultado do implementador, de propósito: dependem do
escopo, não do que ele fez. Auditam o código, não a explicação do autor.

## O que se espera que funcione

- `label: "Grade"` num `rect` com filhos → texto numa faixa no topo, à esquerda.
- `label:` num `rect` vazio → texto centrado na vertical.
- `label:` em `circle`, `use` ou `slot` → erro de decodificação com arquivo e
  localização, no padrão de `round`.
- `label:` em `control` → continua funcionando exatamente como antes.
- `inspect` imprime `rotulo="..."` na linha do Retângulo e **omite** a peça de
  `Forma: Texto`.
- `board` desenha o mesmo Rótulo, sem uma segunda regra de tamanho ou posição.
- Todo Elemento resolvido carrega `Local` não vazio e correto.

## contract-behavior

- A contenção veio de `contemGeometricamente`, e não de uma cópia dela.
- Elemento de `Forma: Texto` não conta como filho — verifique que um Retângulo
  rotulado e vazio é tratado como vazio, e não como cheio por causa do próprio
  Rótulo.
- A contenção atravessa Camadas, como `atribuiElevacao` faz.
- A altura reservada é constante em px do Frame, não fração da altura do
  Retângulo. Prove com um Retângulo de 400 px: a fonte tem que ficar na ordem de
  12 px, não de 180.
- A faixa satura na altura do Retângulo quando ele é mais baixo que ela.
- `fracaoDoRotulo = 0.45` continua intocada.
- A Superfície do Rótulo é o Retângulo que o carrega: confira Elevação e Tom do
  Elemento de Texto contra os do Retângulo, e contra um filho que possa ter
  virado Superfície por engano.
- A invariante de adjacência (Rótulo logo depois do dono) está documentada nos
  dois pontos e tem teste.
- `Rotulo != ""` na cabeça do Retângulo **não** faz o desenho sair duas vezes,
  nem no WebP nem no SVG.
- Repetição, Instância, Slot e Componente: um `rect` com `label` dentro de
  Componente instanciado N vezes materializa N Rótulos, cada um na sua caixa.
- `repeat` sobre um `rect` com `label` clona o Rótulo junto, com caminhos únicos.

## security-data

- Retângulo de área zero, negativa, NaN ou infinita com `label`: nada de pânico,
  nada de divisão que produza NaN na geometria do Rótulo.
- Retângulo mais alto que o Frame, ou fora dele: o Rótulo é recortado, não vaza.
- `label` com string vazia, só espaços, ou com caracteres de controle.
- `label` gigantesco: nenhum caminho aloca em função do comprimento do texto.
- O teto `LimiteDeElementos` é debitado pelo Rótulo materializado? Se não for,
  um `repeat` de mil rects com `label` materializa mil Elementos a mais do que o
  orçamento previu. Isso é blocker.
- `caminhoUnico` continua garantindo caminho único com o segmento novo.

## tests-maintenance

- Os testes provam a **regra**, não a implementação: contenção, saturação e
  respiro são test-first naturais.
- Existe teste de cada erro novo de decodificação, com a mensagem conferida.
- Nenhum golden foi regerado sem que a mudança fosse explicada.
- Nenhum arquivo fora da propriedade de F9 foi tocado (ver
  `.executor/CONTRATO.md`).
- Comentários explicam por quê. Português em tudo. Vocabulário do `CONTEXT.md`
  nos identificadores e nas mensagens; nenhum `_Avoid_` usado.
