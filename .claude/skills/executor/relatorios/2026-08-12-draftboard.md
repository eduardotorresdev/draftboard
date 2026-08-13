# draftboard — relatório de execução

**Data**: 2026-08-12 · **Escopo de origem**: `docs/PRD.md` (78 user stories) + `CONTEXT.md` · **Branch**: `prawn` · **Repo**: `draftboard`

Escopo recebido: *"implementa esse software seguindo as docs de PRD"*. Um binário Go único, sem cgo, que lê wireframes declarativos em YAML e produz WebP em tons de cinza mais uma árvore textual resolvida no stdout. O software é escrito e lido por agentes; custo de token é critério de projeto de primeira classe.

## Funcionalidades

| # | funcionalidade | trilhas | divisibilidade | rodadas |
| --- | --- | --- | --- | --- |
| F1 | CLI, schema, resolução, `inspect` | back | — (raiz; congelei `internal/scene` e o contrato antes) | 4 |
| F2 | Componentes, Instâncias, Slots, Repetição | back | **6/10** → paralelo após congelar contrato + stub | 3 |
| F3 | rasterização de Frame para WebP | back | **9/10** → paralelo direto | 3 |
| F4 | Notas nos três modos | back | **8/10** → paralelo direto | 2 |
| F5 | skill embutida (`skill`, `--install`) | back | **10/10** → paralelo direto | 3 |

Trilhas `front` e `devops`: **ausentes**. O produto é um binário de linha de comando sem superfície web e sem infraestrutura de implantação — não existiram, não foram inventadas.

O que fez F2 cair para 6 e não 8: ela compartilha `internal/schema` com F1 e depende do formato de árvore do `inspect`, que é de F1. Congelei o §2 (schema YAML) e o §4 (formato de árvore) e entreguei stub, mas a fronteira de arquivos nunca ficou disjunta de verdade — e foi exatamente onde a integração doeu.

Todas as cinco couberam no limite de quatro por árvore? **Não** — cinco funcionalidades passam do limite. Não agrupei em worktrees separados por grupo, e sim dei **um worktree por agente** (`isolation: "worktree"`), com integração centralizada em `prawn`. Funcionou, mas ver "Sinais de épico gordo".

## Tempo

- **Tempo-agente**: 38.146.702 ms ≈ **10,6 h**
- **Caminho crítico**: F1 impl1 → F2 impl1 → F2 val1 → F2 impl2 → F2 val2 → F2 impl3 = 13.683.503 ms ≈ **3,8 h**
- **Fator de paralelismo**: **2,79**
- **Turnos do orquestrador**: 30, **não medidos** (o Agent não devolve duração dos meus próprios turnos; não estimo sem base)
- **Agentes levantados**: 25 (15 implementadores, 10 validadores) · **1.340** chamadas de ferramenta · **3.662.133** tokens de subagente

### Por funcionalidade

| func | agentes | tokens | tempo-agente |
| --- | --- | --- | --- |
| F2 | 5 | 773.770 | 3,6 h |
| F4 | 4 | 705.085 | 2,8 h |
| F3 | 5 | 851.044 | 2,5 h |
| F1 | 6 | 887.539 | 1,3 h |
| F5 | 5 | 444.695 | 0,5 h |

### Por papel

| papel | agentes | tokens | fração de tokens | tempo-agente | fração de tempo |
| --- | --- | --- | --- | --- | --- |
| implementador | 15 | 2.398.620 | 65,5% | 6,0 h | 57,0% |
| validador | 10 | 1.263.513 | 34,5% | 4,6 h | 43,0% |

Validação em 43% do tempo-agente: **abaixo da metade**, dentro do que a régua pede. Mas está no teto do confortável, e o motivo é identificável — validadores rodaram mutação de verdade já na primeira rodada, que é caro e é exatamente o que pagou por si (ver Achados).

### Por rodada

| | tempo-agente |
| --- | --- |
| implementação, rodada 1 | 2,0 h |
| implementação, retificações (r2, r3, r4) | 4,1 h |

**Retificação custou o dobro da primeira entrega.** Pela régua da MEDICAO.md isso é sinal de contrato frouxo, e é honesto reconhecer: das quatro seções que precisei emendar em pleno voo (§5b, §8b, §8c, §4b), nenhuma era imprevisível — todas são limites de recurso ou de unicidade que um contrato mais maduro teria trazido escritos.

### Por frente

`back`: 25 agentes, 100% do tempo-agente. `front`: ausente. `devops`: ausente.

## Custo das rotinas de verificação

Árvore parada, sem contenção, medido no fim:

| comando | tempo |
| --- | --- |
| `gofmt -l .` | 0,04 s |
| `go build ./...` | 0,37 s |
| `go vet ./...` | 0,32 s |
| `go test -count=1 ./...` | 3,25 s |
| **portão** | **3,98 s** |
| `go test -count=1 -race ./...` | 41,65 s |

Rodadas: 15 de implementação + 10 de validação = 25 execuções do portão ≈ 100 s, mais **267 mutantes** aplicados ao longo da execução (soma dos vereditos), cada um pagando a suíte do pacote.

**Fator de contenção medido**: o validador de F4 relatou 70,3 s / 22,1 s / 412,6 s para os mesmos comandos que, na máquina ociosa, deram 25,2 s / 3,4 s / 42,7 s — **fator de 3 a 10×** quando três ou quatro agentes disputam a mesma máquina. Isso sozinho justifica o worktree por agente.

**O caso extremo mereceu correção, não tolerância**: um teste de F1 (`TestRenderAceitaTelaExatamenteNoLimite`) rasterizava e codificava 67 Mpx em WebP lossless. Contra o stub de compilação era instantâneo; contra o rasterizador real de F3 levava **25 a 31 min sob `-race`** e a suíte simplesmente não terminava. Só apareceu na integração — nenhuma trilha isolada podia vê-lo. Trocado por um predicado (`cabeNaTela`), o pacote raiz caiu de **504 s para 0,96 s** sob `-race`.

## Contexto

- **Fase de contexto**: do início até o contrato congelado. Li `CONTEXT.md` (2.912 B), `docs/PRD.md` (26.579 B) e escrevi `internal/scene/scene.go` + `CONTRACT.md`. Volume lido ≈ 29,5 KB ≈ **7,4k tokens** — bem abaixo dos ~50k da régua.
- **Contexto de skill** (a que o binário embute e instala): `SKILL.md`, 9.295 B, **1.513 palavras**.
- **Contexto de contrato**: `CONTRACT.md`, 14.676 B, **2.170 palavras**, lido **25 vezes** (uma por agente levantado) ≈ 54k palavras de leitura acumulada.

## Achados

| rodada | bloqueante | quickwin | estrutural |
| --- | --- | --- | --- |
| F5 r1 | 0 | 6 | 1 |
| F5 r2 | 0 | 5 | 0 |
| F1 r1 | 0 | 4 | 2 |
| F1 r2 | 0 | 6 | 2 |
| F3 r1 | 0 | 9 | 1 |
| F3 r2 | **1** | 5 | 2 |
| F2 r1 | **1** | 3 | 1 |
| F2 r2 | **1** | 6 | 3 |
| F4 r1 | 0 | 10 | 3 |
| F4 r2 | 0 | 6 | 3 |
| **total** | **3** | **60** | **18** |

### Os três bloqueantes, e por que os três eram invisíveis por leitura

1. **F3 — `NewCanvas` entrava em pânico com `escala = +Inf`** (`image: NewRGBA Rectangle has huge or negative dimensions`), enquanto `0`, `-1`, `NaN` e `-Inf` produziam uma tela 1×1 usável. Causa: uma função conflava arredondamento de coordenada com saturação no teto de área. **Verifiquei por conta própria** antes de aceitar, com teste descartável sobre `{+Inf, -Inf, NaN, 0, -1, 1e300}`.
2. **F2 r1 — `repeat` sem teto**: `n: 1e19` e `n: 1e30` eram *aceitos* e travavam, com overflow de `int64` **dependente de plataforma** (arm64 satura em maxint64 e passa; amd64 vira negativo e recusa). Mais amplificação multiplicativa: 8 Componentes encadeados com `n: 10`, dentro do limite sancionado de 16 níveis, pediam 10⁸ Elementos a partir de ~1 KB de YAML.
3. **F2 r2 — o conserto do (2) não fechou a bomba.** O teto passou a contar *nascimentos* de Elemento, e o custo da bomba é *trabalho*: uma cadeia de Repetições cuja folha é `elements: []` expande milhões de vezes sem materializar nada e atravessa o teto sem encostar nele. **591 bytes** de YAML e o processo não terminava em 45 s, com RSS plano em 13 MB — não OOM, **livelock**. A fixture original não pegava porque terminava em `rect`, o único formato da família que o teto alcançava.

Densidade de bloqueante: **0,30 por rodada de validação**. Todos os três nasceram de mutação ou de sondagem numérica adversarial; nenhum sairia de revisão por leitura.

### Exceção do teto de rodadas, acionada duas vezes

A skill permite uma única exceção: defeito capaz de deixar a máquina do operador quebrada volta mesmo na segunda rodada, mesmo classificado como quickwin.

- **F5** — `Instala` seguia symlink no destino e **sobrescrevia um arquivo arbitrário** (demonstrado: vítima passou de 8 para 9.295 bytes). Estava na segunda rodada e voltou. O conserto final (`CreateTemp` + `Rename` atômico) fechou também uma janela destrutiva entre `Remove` e `OpenFile`, e apagou dois arquivos de build-tag que a solução anterior exigia.
- **F1** — `NaN`/`Inf` em números do YAML saturavam para `9223372036854775807`, imprimindo `frame home 9223372036854775807x10` com código **0**.

**Uma terceira rodada de validação foi negada e substituída por verificação minha.** F2 chegou à segunda rodada com bloqueante. O teto limita *rodadas de validação*, não a correção de bloqueante — então mandei a retificação ao mesmo implementador e verifiquei eu mesmo, escrevendo **do zero** três bombas que os testes dele não usavam: folha vazia 5×1000, Slot com `default` 5×1000, e Slot sem `default` com 1000×1000 expansões e zero materialização. As três saem com código 1 em **menos de 50 ms**, com a cadeia inteira de arquivos na mensagem.

### Issues abertas (estruturais, nunca retificados)

| # | funcionalidade | issue |
| --- | --- | --- |
| 1 | F4 | **A família de helpers de teste de tinta é frágil por construção.** Texto (`TomFrame`) e fundo do Frame são o *mesmo* cinza; se a região estiver errada, o limiar casa com o fundo e a asserção fica vazia — verde, sempre. Já errou **três vezes** nesta suíte. A saída existe na própria suíte: generalizar `TestModoDesligadoNaoDesenhaNada` para "tinta da Nota = pixels que diferem da cena sem Notas" apaga a categoria inteira. |
| 2 | F2 | O dono da invariante `n` está partido entre `decode` e `clones`, com **duas frases diferentes para a mesma regra**. O `float64` comprou aparência de rigor: o tipo deixou de modelar "quantidade de clones". |
| 3 | F2 | O `default` de `materializa` **é** o caso `TipoInstancia` disfarçado. Um 5º tipo cairia num deref de ponteiro — **pânico** onde o §7 manda código 1. |
| 4 | F2 | `resolucao` é objeto mutável com reset manual por Frame; o 6º campo esquecido vaza estado entre Frames. |
| 5 | F3 | Mistura de sistemas de unidade — uma função com 9 parâmetros misturando espaço do Frame e px de dispositivo — e um limiar mágico (`limiteDeTracado()`). |
| 6 | F4 | `notes_test.go` carrega ~150 linhas de análise de raster genérica que `internal/render` também precisa; extrair antes que virem duas cópias divergentes. |
| 7 | F5 | 19 âncoras faltando no teste anti-dessincronização da skill, e a skill não explica como Elevação vira Tom. |
| 8 | F1 | Varredura inerte entre Camadas em `atribuiElevacao`; seis lacunas de teste em bordas da CLI. |
| 9 | F4 | `Plano.modo` sobrevive para um único ramo; virar `nota.balao bool` tornaria estrutural a garantia de que Desligado não pinta. |

## Métricas derivadas

| métrica | valor | leitura |
| --- | --- | --- |
| Fator de paralelismo | **2,79** | dentro da faixa boa (1,5–3) |
| Fração de validação | **43%** do tempo, 34,5% dos tokens | abaixo da metade; validadores não fizeram trabalho de implementador |
| Tokens por linha entregue | 3.662.133 ÷ 9.049 = **405** | caro, e o porquê está em "Atividades mais custosas" |
| Custo por achado | 1.263.513 ÷ 81 = **15.599 tokens/achado** | ~5.500 tokens por quickwin, ~421k por bloqueante |
| Densidade de bloqueante | **0,30** por rodada de validação | instruções positivas funcionaram; o que sobrou foi limite de recurso |
| Achados por rodada | r1: 40 · r2: 41 | **a segunda rodada não parou de achar** |

O último número é o que decide uma regra: em cinco funcionalidades, a segunda rodada achou **tanto quanto** a primeira, e **dois dos três bloqueantes** apareceram nela. **O teto de duas rodadas está no lugar certo e não deve cair para uma.**

## Desafios

- **Contrato como único ponto de sincronização.** Cinco implementadores em worktrees disjuntos, sem canal entre eles. Escrevi `internal/scene` eu mesmo e declarei todas as dependências no `go.mod` antes de levantar o primeiro agente, para que ninguém precisasse tocar arquivo compartilhado.
- **Padrão de stub de compilação.** `stub_render.go`, `stub_notes.go` e `stub_skill.go` com assinaturas exatas do contrato permitiram que F1 escrevesse a fiação completa da CLI antes de F3, F4 e F5 existirem. Apagados na integração. Foi o que fez o fator de paralelismo chegar a 2,79 — e foi também o que escondeu o teste de 67 Mpx.
- **Verificar pessoalmente os bloqueantes de classe "máquina do operador"**, em vez de aceitar o relato do implementador. Três vezes; nenhuma desmentiu o relato, e a terceira (bombas de F2) exigiu escrever fixtures que os testes dele não tinham.

## Problemas não previstos

- **Worktrees nascendo no commit inicial vazio.** F5, F1, F2 e F4 detectaram e se corrigiram com `git reset --hard prawn`. Custo: um ciclo de detecção por agente. Não é problema da skill, é do harness — mas custa e precisa aparecer.
- **Violação de arquivo congelado que estava certa.** F3 corrigiu unilateralmente o `go.mod` (eu havia fixado `golang.org/x/image` v0.23.0, abaixo do mínimo de `nativewebp` v1.2.1). Voltou para mim pela regra do contrato; verifiquei com pacote-sonda, **aceitei e emendei o §0**. A regra funcionou: a mudança voltou ao dono em vez de ser negociada entre implementadores.
- **Achado que cruza fronteira de arquivo.** F4 precisou tocar `main_test.go` e goldens de F1: com Nota real em `basico.yaml` o Chrome passa a existir e a área recusada muda, e o caso `--layers` (margens zero) deixa de poder dividir o mesmo golden. Autorizei pontualmente e mandei reaplicar **sobre a versão nova de F1**, em vez de fazer merge otimista.
- **O hook `rtk` filtra a saída de `git` e `go`.** Escondeu um merge commit do validador de F4, que diffou contra a base errada e fabricou um achado falso ("CONTRACT.md +26"). Ele mesmo detectou e refez com `rtk proxy`. **Qualquer medição byte-exata precisa de `rtk proxy`** — passei a usar isso em toda a integração.
- **Falha pré-existente corrigida, não reportada**: `w: 1e30` saturava em `int64` e imprimia `9223372036854775807` com código 0 — reproduzido no `prawn` puro, território originalmente de F1. Roteado ao dono atual do arquivo e corrigido com teto de dimensão.

## Atividades mais custosas

| # | atividade | tempo-agente | por quê |
| --- | --- | --- | --- |
| 1 | F4 impl r1 (Notas do zero) | 4,1 h | três modos de layout, colisão 1-D e goldens de imagem numa tacada |
| 2 | F2 val r2 | 3,6 h | reproduzir a tabela inteira com `time -l` e **construir a bomba que sobreviveu** |
| 3 | F2 impl r2 | 3,4 h | trocar o tipo de `n`, dois tetos novos e unicidade de caminho |
| 4 | F3 val r2 | 2,8 h | 31 mutantes + medição pixel a pixel de 30 combinações de Círculo |
| 5 | F2 impl r3 | 2,7 h | mover o orçamento de nascimento para tentativa |

As duas mais caras são de F2, e ambas giram em torno do **mesmo** defeito: limite de recurso numa linguagem declarativa que se expande. Foi o item mais subestimado do contrato.

## Sinais de épico gordo

Conferido contra a régua — **cinco dos sete critérios são verdade**, e a régua pede dois:

| critério | resultado |
| --- | --- |
| mais de quatro funcionalidades | **sim** — cinco |
| contrato acima de ~1000 palavras | **sim** — 2.170 palavras, ~30% acima da régua proporcional para cinco |
| funcionalidade serial bloqueando mais de duas | **sim** — F1 é raiz de F2, F3, F4 e F5 |
| fator de paralelismo abaixo de 1,5 | não — 2,79 |
| fase de contexto acima de ~50k tokens | não — ~7,4k |
| mais de um bloqueante por rodada | não — 0,30 |
| estrutural cruzando fronteira entre trilhas | **sim** — o `preso` duplicado entre `render` e `notes`, e os helpers de raster que `render` e `notes` precisam em comum |

**Veredito: o épico estava gordo.** O corte correto seriam dois épicos com contrato congelado entre eles — **núcleo** (F1 + F2: schema, resolução, reuso, `inspect`) e **saída** (F3 + F4 + F5: rasterização, Notas, skill) —, cada um no seu worktree. A fronteira entre eles já existe e é limpa: `scene.Documento`. Os dois estruturais que cruzam trilha estão **exatamente** nessa fronteira, o que é a evidência de que ela foi desenhada no lugar certo mas atravessada por agentes demais.

O contrato lido 25 vezes × 2.170 palavras é o custo direto disso.

## O que mudou na skill por causa desta execução

Três coisas que a execução ensinou e que valem virar regra:

1. **O teto de duas rodadas fica.** Foi a primeira execução com dado suficiente para questioná-lo, e o dado defende: a segunda rodada achou tanto quanto a primeira (41 vs 40) e trouxe **dois dos três bloqueantes**. Cortar para uma rodada teria embarcado a bomba de amplificação de F2 e o pânico de `+Inf` de F3.
2. **Falta uma regra sobre bloqueante na segunda rodada.** A skill diz "na segunda, aceite se não houver bloqueante" e não diz o que fazer quando há. Resolvi retificando com verificação minha no lugar de um terceiro validador, e funcionou — mas foi interpretação, não regra. **Proposta**: bloqueante na rodada 2 volta ao implementador, e a verificação é do orquestrador, sem terceiro validador.
3. **Stub de compilação precisa de aviso.** Ele comprou o paralelismo (2,79) e escondeu um teste que passava de 25 min contra a implementação real. **Proposta**: toda integração de stub exige rodar a suíte cronometrada e comparar com o tempo contra o stub — a diferença é o que o stub estava escondendo.

O que **não** virou regra, e por quê: a exceção de "máquina do operador" foi acionada duas vezes em cinco funcionalidades, o que é frequência alta para uma exceção. Não proponho alargá-la — os dois casos (escrita fora do destino, número saturado com código de sucesso) são exatamente o que ela descreve, e a alta frequência diz mais sobre a natureza de um CLI que instala arquivo e lê número do usuário do que sobre a regra.
