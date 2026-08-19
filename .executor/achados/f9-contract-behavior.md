# Achados — F9 / contract-behavior

1. **[blocker] O Rótulo é pintado antes dos filhos e some da imagem em silêncio.**
   `internal/resolve/rotulo.go:44-72` + `resolve.go:48`.
   Cenário: `rect {x:5,y:10,w:90,h:80} label:"Resultados"` seguido de
   `rect {x:5,y:10,w:90,h:20} id:cabecalho` — região com barra de cabeçalho, o
   padrão mais trivial que existe. O `inspect` imprime `rotulo="Resultados"`, o
   Elemento de Texto tem `Y == retangulo.Y`, `A == 28`, Elevação e Tom certos, e
   o WebP sai **sem texto nenhum**: a barra é desenhada por cima. A Prancheta faz
   igual. Nenhum aviso. Desfaz a promessa da ADR-0002 e da própria SKILL.md
   ("faixa no topo, fora do caminho dos filhos").
   Correção: manter a emissão adjacente — a Superfície tem de continuar sendo o
   Retângulo — e acrescentar uma passagem chamada **depois** de
   `atribuiElevacao` que move cada Elemento de `Forma: Texto` com
   `Controle == ""` para o fim da sua Camada, preservando a Elevação e o Tom já
   calculados.

2. **[blocker] O caminho do Rótulo é montado sobre o caminho pré-desambiguação.**
   `internal/resolve/rotulo.go:60`. Quebra a regra `<caminho do Retângulo>/rotulo`
   do item 4.
   Cenário: dois `rect` com `id: bloco` e `label:` no mesmo Frame → a Prancheta
   emite `bloco`, `bloco/rotulo`, `bloco~2`, `bloco/rotulo~2`: o Rótulo do
   segundo fica pendurado no caminho do **primeiro**. Quem parear Rótulo↔dono por
   prefixo — o painel da Prancheta e o `--fix` de F11 — atribui o texto ao
   Retângulo errado.
   Correção: `acrescenta`/`emite` devolvem o caminho já desambiguado, e é esse
   valor que vai para `rotuloDoRetangulo` em `achatamento.go:180`.

3. **[quickwin] A frase nova da SKILL.md sobre `rotulo=` é falsa.**
   `internal/skill/SKILL.md:347-348` diz que `rotulo=` "vai sempre por último na
   linha". Cenário: `control: button` com `label` e `to` sai
   `... controle=button rotulo="Salvar" para=t` — o token aparece num Controle,
   como `<parâmetros>` de `detalheDoRotulo`, e **não** é o último. Num `rect` com
   `label` e `to`, aparece depois de `para=`. Quem ler a SKILL.md e pegar "o
   sufixo final `rotulo=`" lê `para=t` como parte do texto.
   Correção: distinguir os dois casos pela posição relativa a `controle=`.

4. **[quickwin] A invariante de adjacência está documentada mas não tem teste que
   morda.** `f9_test.go:203`. Em `rotulos.yaml` o filho não cobre a área do
   Rótulo, então mover a emissão para o fim da Camada mantém o teste verde — a
   única razão declarada para a adjacência não está coberta.
   Correção: acrescentar em `rotulos.yaml` um filho que cubra a faixa do topo e
   afirmar `rotulo.Elevacao == regiao.Elevacao+1`, e não `filho.Elevacao+1`.
