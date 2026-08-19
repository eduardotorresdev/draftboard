# Pendências registradas

Achados estruturais levantados durante uma entrega, que fogem ao escopo dela e
ficam registrados aqui em vez de alargar a fatia.

## As quatro margens do Canvas estão mortas

**Origem**: validação de F10 (aposentadoria do modo Margem), confirmada na
integração de F9 e F10.

**Cenário**: `render.NewCanvas(l, a, margemT, margemD, margemB, margemE, escala)`
e `render.DesenhaFrame(f, escala, margemT, margemD, margemB, margemE, ateCamada)`
recebem quatro margens. Desde que o modo Margem foi aposentado, **todos** os
chamadores de produção passam zero: `comandos.go:64` e `comandos.go:90` (via
`plano.Margens()`, que hoje devolve sempre `0,0,0,0`) e `notes/notes.go:158`. Só
os testes de `internal/render/` passam valor diferente de zero — e passam
justamente para exercitar a máquina que ninguém mais usa.

**Impacto**: o pacote `render` inteiro continua documentado e testado em termos
de Chrome — `canvas.go:6`, `canvas.go:102`, `texto.go:34`, e cerca de quarenta
ocorrências em `render_test.go` e `limites_test.go`. O `CONTEXT.md` já removeu
"Chrome" do vocabulário do domínio, e `scene.TomChrome` sobrevive só como o Tom
do balão flutuante. Quem ler o pacote hoje aprende um conceito que o sistema não
tem mais.

**Conserto**: arrancar os quatro parâmetros de `NewCanvas` e `DesenhaFrame`,
com a origem do Frame virando sempre `0,0`; reescrever os testes de recorte de
borda, que hoje provam "não vazou para o Chrome" e passariam a provar "não
vazou da tela"; renomear `scene.TomChrome` para o que ele de fato é. É
refatoração de um pacote inteiro, com golden a regerar, e não cabe em nenhuma
das fatias da entrega do Rótulo.

## O Rótulo apagado por um filho fica ilegível, não invisível

**Origem**: inspeção visual na integração de F9.

**Cenário**: um Retângulo rotulado que contém um filho apoiado no mesmo topo —
`rect {x:5,y:10,w:90,h:80} label:"Resultados"` mais `rect {x:5,y:10,w:90,h:20}`.
O Rótulo é pintado por cima do filho (isso F9 consertou), mas o Tom dele vem da
Elevação do Retângulo que o carrega, e o filho tem Elevação maior: texto Tom 300
sobre Superfície Tom 500. O texto existe na imagem, com 412 px de tinta, e é
praticamente ilegível.

**Impacto**: cosmético e restrito ao layout adversarial — no caso comum, os
filhos ficam abaixo da faixa e o contraste é o correto. Nenhum diagnóstico
apontaria isto, porque o Rótulo **cabe**: é contraste, não corte.

**Conserto**: derivar o Tom do Rótulo da Superfície que de fato está sob ele no
momento da pintura, e não da Elevação do dono. Muda a regra de Tom, que hoje é
puramente estrutural, e por isso é decisão de projeto — ADR, não conserto.
