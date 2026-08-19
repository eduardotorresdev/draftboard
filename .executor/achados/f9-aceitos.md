# Achados aceitos — F9

Placar: 2 blockers (5 cada) + 9 quickwins (2 cada) = **28 → retificar**.
Nenhum achado foi recusado. Nenhum virou issue.

As duas emendas de contrato (4b, 4c e 13) já estão escritas em
`.executor/CONTRATO.md` e são a fonte da verdade. Leia-as antes de mexer.

---

## Blockers

### B1 — O Rótulo é pintado antes dos filhos e some da imagem em silêncio

`internal/resolve/rotulo.go` + `resolve.go`. **Emenda 4b do contrato.**

Cenário: uma região com barra de cabeçalho, o padrão mais trivial que existe.

    rect {x:5, y:10, w:90, h:80}  label: "Resultados"
    rect {x:5, y:10, w:90, h:20}  id: cabecalho

O `inspect` imprime `rotulo="Resultados"`, o Elemento de Texto tem `Y` certo,
`A == 28`, Elevação e Tom certos — e o WebP sai **sem texto nenhum**: a barra é
desenhada por cima. A Prancheta faz igual. Nenhum aviso.

Correção congelada na emenda 4b: **duas passagens**. `posicionaRotulos` continua
antes de `atribuiElevacao`, com o Rótulo adjacente ao dono; uma segunda passagem,
**depois** de `atribuiElevacao`, move cada Elemento de `Forma: Texto` com
`Controle == ""` para o fim da sua Camada, preservando a Elevação e o Tom já
calculados.

### B2 — O caminho do Rótulo é montado sobre o caminho pré-desambiguação

`internal/resolve/rotulo.go:60` + `achatamento.go:180`. **Emenda 4c.**

Cenário: dois `rect` com `id: bloco` e `label:` no mesmo Frame → a Prancheta
emite `bloco`, `bloco/rotulo`, `bloco~2`, `bloco/rotulo~2`. O Rótulo do segundo
fica pendurado no caminho do **primeiro**, e quem parear Rótulo↔dono por prefixo
— o painel da Prancheta e o `--fix` de F11 — atribui o texto ao Retângulo errado.

Correção: `acrescenta`/`emite` devolvem o caminho já desambiguado, e é esse valor
que vai para a montagem do caminho do Rótulo.

---

## Quickwins

### Q1 — `chavesSoDeControle` ficou sem teste nenhum

`internal/schema/decode.go:432`. O caso que a cobria de carona era o de F6 que
saiu. Mutação `chavesSoDeControle = []string{}` → **313 passam**, e um `rect` com
`items:` passa de erro código 1 para **exit 0, silencioso**.
Correção: fixture em `testdata/f9/` de `rect` com `items:`, e subteste conferindo
a mensagem literal `campo "items" só é permitido em Controle`.

### Q2 — A exclusão `e.Controle != ""` não é mordida

`internal/resolve/rotulo.go:96`. Mutação para `if e.Forma != scene.Texto` → 313
passam, mas o Rótulo do `button` do catálogo muda de `26.4,14 67.2x12` com fonte
5.4 para `26,10 68x20` com fonte 9: **todo Controle rotulado sai com outra caixa**.
Correção: resolver `testdata/f9/controle-com-label.yaml` e afirmar que o Rótulo
do Controle mantém a geometria literal do catálogo.

### Q3 — O respiro horizontal em `X` não tem afirmação

`internal/resolve/rotulo.go:107`. Só o `L` é conferido.
`TestPranchetaRecortaORotuloNaAreaDele` não cobre: monta o `clipPath` esperado a
partir do próprio `rotulo.X`, logo é consistente com qualquer X. Mutação
`rotulo.X = retangulo.X` → 313 passam, e o Rótulo nasce colado na borda — o que o
item 5 proíbe.
Correção: afirmar `rotulo.X == retangulo.X + 6` e `rotulo.L == retangulo.L - 12`,
com os literais escritos.

### Q4 — `Local` só é afirmado como não-vazio

`f9_test.go:TestTodoElementoCarregaLocal`. O único teste de valor usa um `rect` de
Documento, onde o prefixo é vazio. Mutação `Local: no.Local` (sem `ctx.prefixo`)
→ passa, e o `rect` vindo de `./cartao.yaml` carrega `Local = "elements[0]"` em
vez de `"frames[0].layers[0].elements[3] -> ./cartao.yaml: elements[0]"`.
**O `--fix` de F11 editaria outro Retângulo do Documento.**
Correção: afirmar o `Local` literal do Elemento vindo de Componente e do vindo de
Slot.

### Q5 — A frase da SKILL.md sobre `rotulo=` é falsa

`internal/skill/SKILL.md:347-348` diz que `rotulo=` "vai sempre por último na
linha". Um `control: button` com `label` e `to` sai
`... controle=button rotulo="Salvar" para=t`: o token aparece num Controle, como
parâmetro, e **não** é o último.
Correção: distinguir os dois casos pela posição relativa a `controle=`.

### Q6 — A invariante de adjacência não tem teste que morda

`f9_test.go:203`. Em `rotulos.yaml` o filho não cobre a área do Rótulo, então
mover a emissão para o fim da Camada mantém o teste verde.
Correção: um filho que cubra a faixa do topo, afirmando
`rotulo.Elevacao == regiao.Elevacao+1` e **não** `filho.Elevacao+1`.
**Atenção:** depois de B1 a adjacência deixa de valer na fatia final. O teste tem
que afirmar **Elevação e Tom**, que é a razão de ser da adjacência, e não a
posição na fatia.

### Q7 — `label` entra sem teto de comprimento

**Emenda 13 do contrato: `schema.LimiteDoRotulo = 200` runas, erro de
decodificação.** Medido: `label` de 200 000 caracteres com `repeat: {n:20}` cabe
em 195 KB de YAML e custa **61 s de CPU**, produzindo uma imagem onde nada
aparece; com 1 MB e `n:1000`, ~15 h. O texto do Rótulo era a única entrada do
módulo sem teto.

### Q8 — `escapa` deixa passar os caracteres de controle C0

`internal/board/svg.go:213` é só `html.EscapeString`. Um `label` com NUL e ESC
entra cru no HTML: o arquivo deixa de ser texto válido, o parser troca o NUL por
U+FFFD e a Prancheta mostra uma coisa enquanto o WebP desenha `.notdef` — o mesmo
Documento com dois desenhos, que é a divergência que o item 5b foi fechar.
Correção: descartar ou substituir os C0 fora de tab/LF/CR antes do
`html.EscapeString`.

### Q9 — `nota: %s` no `inspect` forja Elementos na árvore

`internal/inspect/inspect.go:32`. Uma `note:` com quebra de linha seguida de seis
espaços e de algo com cara de linha de Elemento produz, na árvore, uma linha
indistinguível de um Elemento real para o agente que a lê. O `rotulo=` que você
acabou de acrescentar sai seguro com `%q`: a assimetria está na mesma função.
Correção: imprimir a Nota com `%q`, na mesma linha da chave.
**Atenção:** isso muda a árvore, e portanto a gramática na SKILL.md (linha de F9)
e todo golden de `inspect` que contenha `nota:`. Regere os que mudarem e confira
um a um.
