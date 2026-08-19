# Expectativas de validação — F10

Escrito **antes** do resultado do implementador, de propósito.

## O que se espera que funcione

- `render doc.yaml` → sem Nota, imagem do tamanho do Frame.
- `render doc.yaml --notes` → balões flutuantes.
- `render doc.yaml --notes float|margin|off` → código 1 com mensagem de uso.
- `render --notes doc.yaml` → funciona; o arquivo não vira valor da opção.
- Balões vizinhos não se sobrepõem.
- `notes.LimiteDaNota == 200`, em runas.

## contract-behavior

- `notes.Modo` e `margem.go` sumiram de verdade, e não sobrou nenhum resquício
  do Chrome que cresce nem da linha de chamada longa.
- `Planeja` tem a assinatura congelada e `comandos.go` passa `nil` quando as
  Notas estão desligadas. `Margens` e `Desenha` continuam seguros com receptor
  nulo — confira, não confie.
- O teto de área (`cabeNaTela`) recebe margens 0,0,0,0 quando não há plano.
- Anti-colisão: prove pelos retângulos do Plano, não por pixel. Dois Elementos
  anotados com âncoras a poucos px de distância; três; dez. Nenhum par se
  intersecta.
- Estabilidade: embaralhar a ordem de declaração sem mexer na geometria produz
  exatamente a mesma imagem. O teste já existe — ele continua passando por
  mérito, ou por acidente?
- Escala: o layout inteiro escala junto, e as margens não mudam com a escala.
- Frame estreito, Frame minúsculo, Elemento anotado colado na borda: o balão
  fica preso dentro do Frame.
- Nota vazia ou só com espaços continua sendo ausência de Nota.
- `LimiteDaNota` é **respeitado no layout** mas **não emite diagnóstico e não
  trunca**. Se o implementador truncou ou avisou, é quebra de contrato.

## security-data

- `--notes` seguido de fim de argumentos; `--notes=true`; `--notes=`;
  `--notes --layers`; `--notes` duas vezes.
- Nota com quebra de linha explícita, com uma palavra maior que o Frame, com
  runas multibyte.
- Anti-colisão com muitas Notas: o algoritmo não pode ser quadrático sobre um
  Frame com milhares de Elementos anotados. Se for, meça e reporte.
- Nenhum caminho novo entra em laço infinito quando nada mais cabe no Frame.

## tests-maintenance

- `internal/notes/notes_test.go` foi reescrito, não mutilado: os casos que
  provavam o Margem ou viraram casos do Flutuante ou sumiram com justificativa.
- Os goldens de `testdata/f4` foram **olhados**. O relatório descreve o que
  mudou nas imagens. Se não descrever, é achado.
- Nenhum arquivo fora da propriedade de F10 foi tocado.
- A SKILL.md não menciona mais os três modos nem o Chrome, e o que sobrou está
  correto.
- Comentários explicam por quê. Português em tudo. `Chrome` não é mais termo do
  domínio e não deve aparecer como se fosse.
