# Draftboard — o que o Executor precisa saber

Go 1.26, sem dependências novas permitidas. Binário CLI: `main.go`, `comandos.go`,
`opcoes.go` na raiz; tudo mais em `internal/`.

## Gate

Não existe workflow de CI de teste — `.github/workflows/release.yml` só publica
Lançamento por tag. O gate é local:

    go build ./... && go vet ./... && gofmt -l . && go test ./...

Baseline em `main` (cb909b9): verde, 298 testes em 11 pacotes.

Goldens WebP ficam em `testdata/f4` e são regravados com
`go test ./internal/notes/ -atualizar`.

## Trilhas presentes

Só uma: `back` (Go). Não há front, devops nem data. `docs` existe como trilha de
documentação (SKILL.md embutida, PRD, ADRs) mas anda junto do código.

## Como observar

    go run . inspect  arquivo.yaml     # árvore textual
    go run . render   arquivo.yaml --out DIR
    go run . board    arquivo.yaml --out DIR

A superfície observável primária é textual (`inspect`) — é ela que o validador
usa. As imagens WebP se conferem abrindo o arquivo.

## Vocabulário e decisões

- `CONTEXT.md` — glossário obrigatório. Os `_Avoid_` são proibições reais.
- `docs/adr/` — 0001 (Rótulo de Controle), 0002 (posição derivada da contenção),
  0003 (categoria por corrigibilidade), 0004 (Margem aposentado).
- `CONTRACT.md` — contrato da CLI (códigos de saída, stdout/stderr).
- `docs/PRD.md`.

Código, comentários, identificadores e mensagens em **português**.

## Issues

Não há tracker externo. Achado estrutural vira seção em `docs/`.
