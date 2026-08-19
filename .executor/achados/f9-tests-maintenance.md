# Achados — F9 / tests-maintenance

Todos provados por mutação que **sobreviveu** ao `go test ./...` (313 passed).
O código entregue está correto; o que falta é o teste que o protege.

1. `internal/schema/decode.go:432` — o laço novo de `chavesSoDeControle` ficou
   sem nenhum teste, porque o caso que o cobria de carona era o de F6 que foi
   removido. Mutação `chavesSoDeControle = []string{}` → 313 passed, e um `rect`
   com `items:` passa de erro código 1 para **exit 0, silencioso**.
   Correção: fixture em `testdata/f9/` de `rect` com `items:`, e subteste em
   `TestLabelSoEmRetanguloOuControle` conferindo a mensagem literal
   `campo "items" só é permitido em Controle`.

2. `internal/resolve/rotulo.go:96` — a exclusão `e.Controle != ""` não é mordida
   por teste. Mutação para `if e.Forma != scene.Texto` → 313 passed, mas o
   Rótulo do `button` do catálogo muda de `26.4,14 67.2x12 font-size 5.4` para
   `26,10 68x20 font-size 9`: todo Controle rotulado sai com outra caixa.
   Correção: resolver `testdata/f9/controle-com-label.yaml` em `f9_test.go` e
   afirmar que `e0/rotulo` mantém a geometria literal do catálogo.

3. `internal/resolve/rotulo.go:107` — o respiro horizontal em `X` não tem
   afirmação; só o `L` é conferido. `TestPranchetaRecortaORotuloNaAreaDele` não
   cobre: ele monta o `clipPath` esperado a partir do próprio `rotulo.X`, então é
   consistente com qualquer X. Mutação `rotulo.X = retangulo.X` → 313 passed, e o
   Rótulo nasce colado na borda — exatamente o que o item 5 proíbe.
   Correção: afirmar `rotulo.X == retangulo.X + 6` e
   `rotulo.L == retangulo.L - 12`, com os literais escritos.

4. `f9_test.go:TestTodoElementoCarregaLocal` — só afirma `Local != ""`, nunca o
   valor, e o único teste de valor usa um `rect` de Documento, onde `prefixo` é
   vazio. Mutação `Local: no.Local` (sem `ctx.prefixo`) → passa, e o `rect` vindo
   de `./cartao.yaml` carrega `Local = "elements[0]"` em vez de
   `"frames[0].layers[0].elements[3] -> ./cartao.yaml: elements[0]"`. O `--fix`
   de F11 editaria **o nó errado** — outro Retângulo do Documento.
   Correção: afirmar o `Local` literal do Elemento vindo de Componente e do vindo
   de Slot.

## Julgamentos pedidos

- Remoção de `f6_test.go` / `testdata/f6/label-em-rect.*`: **legítima no mérito**
  (o golden afirmava o que o item 1 revoga; manter deixaria o gate vermelho por
  design), **ilegítima na propriedade** (o item 10 diz "F9 não toca em
  f6_test.go"; a regra manda parar e reportar — reportou, mas seguiu). Conflito
  com F10 é benigno: hunks em 187 vs 219.
- Cobertura migrada: **mais larga** no eixo do `label` (cobre circle, use e slot,
  mensagem literal, código 1, stdout vazio, mais o caso positivo do Controle),
  **mais estreita** no eixo que o caso antigo carregava de carona — o laço
  genérico de rejeição, que é o achado 1.
- Nenhum outro arquivo fora da propriedade de F9 foi tocado.
- `SKILL.md` e `skill_test.go` ficaram dentro do particionamento; as linhas de
  F10 (84, 144, :254, :267, :113, :240) estão intactas; a linha do `circle` não
  mudou.
