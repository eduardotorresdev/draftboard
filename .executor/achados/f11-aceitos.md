# Achados aceitos — F11

Placar: 3 blockers × 5 + 6 quickwins × 2 = **27**. Todos aceitos, nenhum
rejeitado. Duas reclassificações minhas, explicadas abaixo.

O B1 foi achado pelas duas dimensões (contract-behavior e security-data) com o
mesmo diagnóstico e o mesmo conserto; contado uma vez. Eu reproduzi B1 e B2 na
mão antes de aceitar, e medi o número do Q3 eu mesmo.

## B1 — o respiro volta por diferença, e some no Retângulo estreito

`internal/diag/diag.go:210`. `necessarioNoRetangulo := largura + (dono.L - e.L)`
supõe que a diferença é sempre `2 * respiroDoRotulo`, mas `posiciona` satura em
`max(0, dono.L - 2*respiro)`: num Retângulo mais estreito que 12 px a diferença
vale `dono.L`.

Reproduzido por mim, Frame 400×300, `rect: {x:0, y:0, w:2, h:20}` com
`label: "Config"`:

    frames[0].layers[0].elements[0]: w 2 → 12
    aviso: ... precisa de 37 px e tem 36; use w: 13     <- na MESMA chamada
    frames[0].layers[0].elements[0]: w 12 → 13          <- segunda chamada

Derruba as duas caixas da definição de pronto: "conserta de primeira" e "a
segunda não muda um byte". Também acontece com `w: 0` e `w: -5`.

**Conserto**: emenda 21 do contrato. `resolve.RespiroDoRotulo` exportado, e a
conta soma `2*resolve.RespiroDoRotulo` por valor.

## B2 — `fix` lê a primeira chave `w`, a decodificação lê a última

`internal/fix/fix.go:404`. `filho` devolve a primeira ocorrência; `mapa()` em
`decode.go` monta um `map` iterando, então a última vence. Chave repetida é
aceita hoje — conferi: `validate` sai 0 num nó com dois `w`.

Cenário: `- {rect: {x:5, y:10, w:20, h:15, w:21}, id: a, label: "Configurações
avançadas"}` → `--fix` grava `w: 40, ..., w: 21`, a largura desenhada continua
21%, o Aviso continua saindo, e toda execução seguinte reescreve o arquivo
imprimindo `w 40 → 40`. Nunca converge.

**Conserto**: `filho` varre até o fim e devolve o último par com a chave,
casando com a semântica da decodificação. Aceitável também recusar o nó com
chave repetida como não alargável — mas então a razão tem que ser uma das
literais do item 16, e nenhuma serve; prefira casar com a decodificação.

## B3 — o arredondamento para cima não tem teste que morda

`internal/diag/diag_test.go`. Todo caso do repositório cai numa fração de
exatamente 0,5, onde `Ceil` e `Round` dão o mesmo número: `"Resultados da busca"`
mede 118 px, o respiro devolve 12, e `100*130/400 = 32,5`. Vale para o teste
unitário e para `largo.yaml`, `misto.yaml` e `na-borda.yaml`.

Mutação `math.Ceil` → `math.Round` em `diag.go:213`: **415 testes verdes**. O
implementador provou `Ceil` → `Floor`, que é a mutação fraca.

É a expectativa 2 de `tests-maintenance`, escrita antes da implementação
justamente porque este é o erro fácil.

**Conserto**: um caso calibrado numa fração pequena, conferindo o `W` devolvido
por `Alargamentos` contra o inteiro esperado, e mantendo o round-trip de aplicar
o `w` e exigir silêncio.

## Q1 — a régua com hinting subestima o que a rasterização pinta

`internal/diag/diag.go:110`. Medi eu mesmo, vinte `l` a 12,6 px, largura em px do
espaço do Frame:

| escala | `HintingFull` | `HintingNone` |
| --- | --- | --- |
| 1 | 60,000 | 67,500 |
| 3 | 66,667 | 67,396 |
| 8 | 67,500 | 67,422 |
| 20 | 67,000 | 67,438 |

A régua diz 60 e a imagem pinta 66,7. O validador capturou o corte: dois `l`
comidos pelo `mascaraDaArea` num Documento que o `--fix` já tinha declarado
consertado.

**Conserto**: emenda 22. `render` publica a medida do diagnóstico com
`font.HintingNone` e `diag` a usa. A rasterização continua com `HintingFull`.

## Q2 — o teto da fonte não é perseguido, mas tem que estar escrito

`internal/diag/diag.go:176`. Acima de `--scale ~20` o rasterizador satura a fonte
em 256 px de dispositivo e pinta um Rótulo menor que a régua descreve: Frame
100×60, `rect: {x:0,y:0,w:90,h:90}`, `label: "Configurações"`, `--scale 30` avisa
um corte que naquela escala não acontece.

**Conserto**: emenda 23 — não se persegue, fica escrito na tabela de diagnóstico
da SKILL.md como limite conhecido.

## Q3 — sem `Sync()` antes do rename

`internal/fix/fix.go:246`. `escreveNoLugar` fecha e renomeia sem sincronizar: o
rename é atômico para o processo, não para a máquina. Queda de energia logo
depois do comando devolver 0 deixa o Documento com 0 byte — exatamente o que o
comentário de `Grava` diz estar evitando.

**Conserto**: `tmp.Sync()` antes de `tmp.Close()`.

## Q4 — a preservação do modo não é conferida por teste nenhum

`internal/fix/fix_test.go:46`. `TestGravaRecusaArquivoSomenteLeitura` usa 0444,
recusado antes da escrita, e nenhum caso lê a permissão depois de um `Grava` bem
sucedido. Mutação: apagar `os.Chmod(nome, modo)` em `fix.go:251` → suíte verde.
Na prática, um Documento 0640 viraria 0600 e o grupo perderia o acesso por ter
sido corrigido.

**Conserto**: gravar a fixture com modo distinto de 0600 e comparar
`os.Stat(...).Mode().Perm()` depois do `Grava`.

## Q5 — a guarda de mtime amostra depois de ler

`internal/fix/fix.go:78`. `Abre` lê o conteúdo e **depois** amostra `Size`/
`ModTime`: se o editor do autor salvar entre as duas, `Grava` compara com o mtime
pós-edição, aprova, e grava o buffer antigo com o `w` alargado. A edição do autor
é revertida em silêncio.

Reclassificado de structural para quickwin: o conserto cabe no arquivo já tocado
e a guarda já é promessa do item 18 — uma guarda que não guarda é defeito da
funcionalidade entregue, não refatoração.

**Conserto**: amostrar antes da leitura e conferir de novo depois, recusando
quando os dois stats diferirem.

## Q6 — as duas leituras de `consertaEInspeciona` não estão amarradas

`comandos.go:233`. `resolve.Arquivo` lê uma vez e `fix.Abre` lê outra; os `Local`
medidos na primeira são aplicados na segunda. Se o arquivo for reescrito entre as
duas trocando a ordem dos `elements`, `frames[0].layers[0].elements[0]` passa a
endereçar outro Retângulo e o `--fix` alarga um `w` cujo Rótulo cabia.

Reclassificado de structural para quickwin pela mesma razão: o item 18 congela
que "`Aplica` recebe (ou reconfere) o conteúdo que `diag` viu". É contrato não
cumprido.

**Conserto**: capturar tamanho e mtime antes de `resolve.Arquivo` e conferi-los
em `consertaEInspeciona`, recusando com a mensagem que `fix` já tem.

## Confirmado e conforme — não mexer

O validador de contract-behavior rodou e aprovou, e vale registrar para que a
retificação não regrida nada disso: Aviso com `w` em porcento e saída 0;
Componente → Erro com imagem escrita e saída 1; Nota de 201 runas → Erro/imagem/1
e 200 runas em silêncio; `--fix` idempotente no caso comum, em fluxo, em bloco,
com comentário na linha, com dois `w` acentuados na mesma linha, e com símlink
preservado; `--fix` com só Erro não toca o mtime; somente-leitura e diretório sem
permissão dão `*scene.Erro` do domínio; borda direita imprime o Aviso novo e sai
0; os quatro verbos diagnosticam com os artefatos escritos; Componente ausente
continua abortando sem imagem; `--fix` em `render|board|validate` é opção
desconhecida; `--scale 1` e `--scale 8` dão o mesmo diagnóstico; o `grep Chrome`
da definição de pronto está vazio. Varredura de 160 combinações rótulo × w × h:
todas cabem depois do `--fix`, nenhuma cai no caso estreito do B1.
