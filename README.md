# Boilerplate

Template de projetos pessoais: um monorepo com backend em Go, frontend em Next.js, specs OpenAPI e observabilidade local, tudo de pé com um `tilt up`.

<!-- example:begin -->

Ele traz um serviço de exemplo, o `notes`, que atravessa todas as camadas: uma nota criada na tela passa pela Server Action, pelo cliente tipado e pelo serviço Go, e aparece com trace, log e métrica no Grafana.

<!-- example:end -->

## Stack

| Área | O que tem |
| --- | --- |
| Backend (`backend/`) | Go (versão no `go.work`) com [uber/fx](https://github.com/uber-go/fx) e [go-kit](https://github.com/lcnascimento/go-kit) (`errors`, `http`, `o11y`, `env`, `util`, `validator`), `golangci-lint`, `mockgen`, `govulncheck` e coleção [Bruno](https://www.usebruno.com/) |
| Observabilidade local (`backend/`) | Alloy, Prometheus, Loki, Tempo, Pyroscope e Grafana via Docker Compose, com os dashboards `overview` (home), `http`, `logs` e `traces` |
| Banco local (`backend/`) | Postgres 18 via Docker Compose, compartilhado: um database e um role por serviço. Acesso com [pgx](https://github.com/jackc/pgx) (`pgxpool`, instrumentado com [otelpgx](https://github.com/exaring/otelpgx)) e queries tipadas pelo [sqlc](https://sqlc.dev/). Migrations com [goose](https://github.com/pressly/goose) (como biblioteca) e lint delas com [squawk](https://squawkhq.com/) |
| Frontend (`frontend/`) | Next.js 16 (App Router, React Compiler), next-intl 4 (pt-BR e en), shadcn + Tailwind 4 + Storybook, Zod, Vitest, Testing Library e Playwright |
| Monorepo do frontend | pnpm + Turborepo: app `apps/web`, pacotes `api`, `i18n`, `types`, `ui` e `utils`, configs compartilhadas em `tooling/{eslint,typescript}` |
| Contrato (`openapi/`) | Uma spec OpenAPI 3.1 por serviço, servida pelo [Scalar](https://scalar.com/) e validada pelo [vacuum](https://quobix.com/vacuum/) |
| Tooling raiz | [Tilt](https://tilt.dev/), lefthook (lint no pre-commit) e commitlint (Conventional Commits no commit-msg) |
| CI (`.github/workflows/`) | `ci-backend`, `ci-frontend`, `ci-e2e` e `ci-openapi`, cada um disparado pelos caminhos da sua área |

```text
backend/
  services/<name>/     um módulo Go por serviço (bundle, cmd, internal, pkg)
  config/              alloy, grafana, tempo e a coleção Bruno
  quality/             placeholder para testes de integração e performance
frontend/
  apps/web/            app Next.js (shell + uma pasta de rotas por funcionalidade)
  packages/            api, i18n, types, ui, utils
  tooling/             eslint e typescript compartilhados
openapi/               uma spec por serviço + Scalar
```

## Como rodar

### Pré-requisitos

- **Go** na versão do `backend/go.work`. Com um Go mais antigo instalado, o `GOTOOLCHAIN=auto` (padrão) baixa a versão certa sozinho.
- **Node** na versão do `.nvmrc` (`nvm use`, `fnm use` ou equivalente).
- **pnpm**. Qualquer versão recente serve: ela troca para a versão do `packageManager` dos `package.json` sozinha.
- **Docker**, para o Postgres, a observabilidade e o Scalar. O `psql` roda dentro do container, então não precisa dele no host.
- **Tilt**.

### Primeira vez

```sh
pnpm install                 # raiz: commitlint + lefthook (instala os git hooks)
pnpm -C frontend install     # workspace do frontend
make -C backend setup        # cria os .env a partir dos .env.example e instala as ferramentas Go
```

O `make -C backend setup` chama o `make -C backend env`, que cria `backend/.env` e o `.env` de cada serviço (`backend/services/<name>/.env`) só quando eles ainda não existem; o `env` também roda sozinho, sem instalar as ferramentas. Para pegar um `.env.example` atualizado, apague o `.env` e rode de novo.

### Subindo tudo

```sh
tilt up
```

A UI do Tilt fica em <http://localhost:10350>. Ela sobe:

| O quê | Onde |
| --- | --- |
| App web (`next dev`) | <http://localhost:3000> |
| Postgres (superuser `postgres`/`postgres`) | `localhost:5432` |
| Grafana (home = dashboard `overview`) | <http://localhost:3010> |
| Scalar (referência das APIs) | <http://localhost:8090> |
| Prometheus, Loki, Tempo, Pyroscope, Alloy | `9090`, `3100`, `3200`, `4040`, `12345` (links na UI do Tilt) |

<!-- example:begin -->

E cada serviço Go na sua porta: o `notes` em <http://localhost:8080> (`/health`, `/ping`, `/notes/v1/notes`).

<!-- example:end -->

Para um stack leve, sem observabilidade, Scalar nem `web-e2e-ui`, suba o modo mínimo:

```sh
TILT_STACK=minimal tilt up
```

Ele sobe só o Postgres, os serviços Go (com os recursos de banco), o `web` e as dependências deles. Sem a variável, o `tilt up` sobe tudo; outro valor é erro. A observabilidade fica em `backend/docker-compose.observability.yml`, que o Tilt só carrega fora do modo mínimo, e o `backend/docker-compose.yml` guarda o que todo stack sobe. Sem o Alloy no ar, os logs dos serviços que exportam por OTLP não aparecem em lugar nenhum.

Os logs dos serviços Go vão por OTLP para o Loki (`OTEL_LOGS_EXPORTER=otlp` no `backend/.env`), então eles aparecem no dashboard `logs` do Grafana, filtrados por `service_name`, e não no painel do Tilt.

O app só fala com o backend pelo servidor: o Tilt passa a `<NAME>_API_URL` de cada serviço para o `next dev`. Para rodar o `next dev` sem o Tilt, copie `frontend/apps/web/.env.example` para `frontend/apps/web/.env.local`. Sem essas variáveis, o servidor recusa subir.

### Banco de dados

Um único container `postgres` (em `backend/docker-compose.yml`) atende todos os serviços; cada serviço tem um database e um role com o nome dele, e o database pertence ao role. No Tilt, cada serviço encadeia `postgres → <name>-db → <name>-migrate → <name>-api-*`, então a API só sobe com o banco criado e migrado. Os dados ficam no volume `postgres_data` e sobrevivem a `tilt down`/`tilt up`. Sem o Tilt, suba só o banco com `docker compose up -d --wait postgres` em `backend/`.

No diretório do serviço (`backend/services/<name>/`), com o `postgres` no ar:

```sh
make db.setup      # cria o role e o database, só se não existirem
make db.migrate    # aplica as migrations pendentes (binário cmd/migrate, mesmo DATABASE_URL da API)
make db.reset      # apaga o database, recria e migra de novo
```

As migrations ficam em `internal/repository/migrations/` de cada serviço, embutidas no binário. São só para frente (só a seção `-- +goose Up`; desfazer algo é uma migration nova), numeradas em sequência com 5 dígitos, cada uma na sua transação, e rodam sob advisory lock, então execuções concorrentes não se atropelam. A API não roda migrations no boot. Para criar a próxima, em `backend/`:

```sh
make migration.new name=add_tags                  # basta enquanto só um serviço tem migrations
make migration.new name=add_tags service=<name>   # com mais de um
```

A API abre o pool no boot e pinga o banco: com o banco inacessível, ela falha em até 5s com um erro dizendo o host e a causa. As queries aparecem nos traces do Grafana.

Cada repositório escreve suas queries em SQL num `queries.sql` ao lado dele, e o [sqlc](https://sqlc.dev/) gera o código Go tipado num pacote `internal/db` do repositório (commitado). O `sqlc.yaml` na raiz do serviço tem um bloco `sql:` por repositório, com o `schema` apontando para as migrations. Depois de mudar uma query ou uma migration, rode `make sqlc` em `backend/`; o `ci-backend` falha se o código gerado estiver desatualizado.

Se o banco tiver uma migration aplicada que a branch atual não conhece (por exemplo, depois de trocar de branch), o `<name>-migrate` e o `make db.migrate` falham com uma mensagem pedindo `make db.reset`. Para zerar todos os bancos, apague o volume: `docker compose down -v` em `backend/`.

### Lint e testes

**Backend** (em `backend/`, cada alvo roda em todos os módulos do `go.work`):

```sh
make lint             # golangci-lint (make lint.fix corrige o que der) e squawk nas migrations
make test             # go test -race com cobertura: unit + componente (exige o Postgres no ar)
make test.unit        # só os de unidade (-short), sem banco
make test.component   # só os de componente (-run Component)
make sqlc             # regenera e confere (sqlc vet) o código das queries de cada serviço
make mocks            # regenera internal/mocks/mocks.go a partir de cada contracts.go
make tidy             # go mod tidy
make vuln-check       # govulncheck (só local, fora do CI)
```

Os testes de componente (`TestComponent<Unit>_<Scenario>`, pulados com `-short`) rodam contra o Postgres de verdade, sem precisar do `<name>-db` nem do `<name>-migrate`: basta o container (`tilt up` ou `docker compose up -d postgres` em `backend/`). Cada teste ganha um database próprio, clonado de um template já migrado pelo [pgtestdb](https://github.com/peterldowns/pgtestdb), então eles rodam em paralelo e nunca tocam o database de dev. O template é reaproveitado entre execuções e só é recriado quando as migrations mudam; o database de um teste que falha fica no servidor, com o nome no log do teste. A conexão de admin vem de `POSTGRES_ADMIN_URL` (padrão: o superuser do compose, `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`). Com o Postgres fora do ar, eles falham (não pulam) com uma mensagem dizendo o que rodar.

**Frontend** (em `frontend/`, via Turborepo):

```sh
pnpm lint
pnpm typecheck
pnpm test                # unit, componentes e as stories do ui como testes
pnpm build
pnpm test:e2e            # Playwright contra o stack do Tilt (suba-o antes)
pnpm format:check        # prettier (pnpm format corrige)
pnpm storybook           # Storybook do ui em http://localhost:6006
```

As stories do `ui` e o e2e rodam no Chromium do Playwright. Instale uma vez:

```sh
pnpm -C frontend --filter @boilerplate/web exec playwright install chromium
```

O e2e depende do Tilt: o `pnpm test:e2e` roda contra o stack já de pé (`tilt up` ou `TILT_STACK=minimal tilt up`), o mesmo que você usa no dia a dia, e não sobe servidor nenhum. Antes do primeiro teste, ele confere o `web` e o `/health` de cada serviço e, se algum não responder, aborta em segundos pedindo `tilt up`. As URLs vêm de env, com as portas do Tilt como padrão: `WEB_URL` (`http://localhost:3000`) e uma `<NAME>_URL` por serviço, todas em `frontend/apps/web/e2e/stack.ts`.

O e2e divide os dados com o seu dev: cada teste cria os próprios registros com um marcador único `e2e-` (a fixture `marker`, em `e2e/fixtures.ts`), só faz asserções sobre eles e, no fim do teste, mesmo em falha, a fixture apaga pela API de cada serviço os registros com esses marcadores.

No CI, o `ci-e2e` faz o mesmo caminho: sobe `TILT_STACK=minimal tilt up` em background, espera todos os recursos ficarem prontos, roda o `pnpm test:e2e` e sempre faz `tilt down`. Em falha, os logs do Tilt vão para o log do passo e para o artefato `tilt-logs`.

Para depurar o e2e, o recurso `web-e2e-ui` do Tilt (manual: dispare-o na UI do Tilt) abre o [UI Mode](https://playwright.dev/docs/test-ui-mode) do Playwright em <http://127.0.0.1:9330>, contra o mesmo stack.

**OpenAPI** (na raiz):

```sh
docker run --rm -v "$PWD:/work:ro" -w /work dshanley/vacuum:v0.33.0 lint --details openapi/*.yaml
```

O pre-commit (lefthook) roda `make lint` quando o commit tem `.go` e `pnpm lint` quando tem TS/JS. O commit-msg exige [Conventional Commits](https://www.conventionalcommits.org/).

### Explorando a API e depurando

- **Bruno**: abra `backend/config/bruno` no Bruno, ou rode uma coleção pela CLI a partir da pasta dela (`backend/config/bruno/collections/<Name>`): `npx @usebruno/cli run --env local`.
- **Debug**: o build dos serviços é de debug (`-gcflags="all=-N -l"`). Com o serviço rodando pelo Tilt, use a configuração `Attach` do `.vscode/launch.json` e escolha o processo (o binário do serviço, `<name>_api_frontend`).

<!-- example:begin -->

A coleção do `notes` fica em `backend/config/bruno/collections/Notes`, e o `Create` dela guarda o id da nota para os requests seguintes.

<!-- example:end -->

## Como adicionar um serviço

Nos passos abaixo, `<name>` é o nome do serviço em kebab-case (ex.: `billing`) e `<NAME>` é o mesmo nome em maiúsculas com `_` (ex.: `BILLING`).

<!-- example:begin -->

O `notes` é o molde. Em arquivos compartilhados, tudo o que é dele está entre cercas `example:begin` / `example:end`, então cada cerca indica um lugar onde um serviço novo também entra.

<!-- example:end -->

**Backend**

1. Crie `backend/services/<name>/` com dois módulos Go: `go.mod` (`github.com/lcnascimento/boilerplate/backend/services/<name>`) e `pkg/go.mod` (`.../<name>/pkg`, com o `replace` para ele no primeiro). O domínio vive em `pkg/domain`, `pkg/models`, `internal/application`, `internal/repository` e `internal/transport`, e cada construtor é registrado no fx uma vez em `bundle/`. O `Makefile` do serviço define o `OTEL_SERVICE_NAME`, o binário (`build.api.frontend` / `run.api.frontend`) e o `FRONTEND_PORT`; o `Tiltfile` dele, os recursos do Tilt. O banco vem junto: `db.setup`/`db.migrate`/`db.reset` no `Makefile` (database e role com o nome do serviço), os recursos `<name>-db` e `<name>-migrate` no `Tiltfile`, as migrations em `internal/repository/migrations/` com o `cmd/migrate`, o `sqlc.yaml` com as `queries.sql` dos repositórios, e o `DATABASE_URL` no `.env.example`.
   <!-- example:begin -->
   O caminho mais curto é copiar `backend/services/notes/` e trocar `notes` pelo novo nome nesses lugares e nos imports.
   <!-- example:end -->
2. Escolha uma porta livre (ex.: `8082`) e use-a no `.env.example` do serviço (`FRONTEND_PORT`), no `FRONTEND_PORT ?=` do `Makefile` e na `readiness_probe` e no link do `Tiltfile` do serviço.
   <!-- example:begin -->
   A `8080` é do `notes`.
   <!-- example:end -->
3. Registre os dois módulos no `backend/go.work`:

   ```text
   use (
   	./services/<name>
   	./services/<name>/pkg
   )
   ```

4. Inclua o serviço no `backend/Tiltfile`: `include('./services/<name>/Tiltfile')`.
5. Rode `make tidy mocks sqlc lint test` em `backend/`, com o `postgres` no ar. Um teste com `fx.ValidateApp` em `bundle/` quebra se o grafo do fx estiver incompleto.
6. (Opcional) Crie a coleção Bruno em `backend/config/bruno/collections/<Name>` e liste-a em `collections:` no `workspace.yml`.

**OpenAPI**

7. Crie `openapi/<name>.yaml` (OpenAPI 3.1, autocontido) e monte-o no Scalar com uma linha em `openapi/docker-compose.yml`: `- ./<name>.yaml:/sources/<name>.yaml:ro`.

**Frontend**

8. Crie `frontend/packages/api/src/<name>/index.ts`. O `exports` com curinga do pacote já publica o subpath `@boilerplate/api/<name>`, sem editar `package.json`. Ele traz os schemas Zod em `schemas/index.ts` (o entry isomórfico que client components importam), o cliente sobre `request()` de `@boilerplate/api/core` e um `parse<Name>Env` que valida `<NAME>_API_URL`.
   <!-- example:begin -->
   Siga `src/notes/`.
   <!-- example:end -->
9. Declare `<NAME>_API_URL`:
   - em `frontend/Tiltfile`, no `serve_env` do recurso `web`;
   - em `frontend/apps/web/.env.example`;
   - em `frontend/apps/web/instrumentation.ts`, chamando o `parse<Name>Env` dentro do `Promise.all`, para o servidor falhar na subida sem ela.

   O `turbo.json` já repassa `*_API_URL` para as tasks.

10. Telas: as rotas em `frontend/apps/web/app/(dashboard)/<name>/`, as funções `"use server"` em `frontend/apps/web/actions/<name>.ts`, as mensagens em `frontend/apps/web/shared/_messages/<name>.json` (pt-BR e en no mesmo arquivo) registradas em `shared/_messages/index.ts`, e o item do menu em `navRoutes` (`shared/navigation.ts`).
11. e2e: em `frontend/apps/web/e2e/stack.ts`, declare a `<NAME>_URL` (padrão, a porta do serviço) e acrescente o `/health` dela em `stackProbes`; se o e2e criar dados no serviço, acrescente em `e2e/fixtures.ts` uma limpeza em `cleanups` que apague pela API os registros com os marcadores. Em `frontend/Tiltfile`, acrescente o recurso do serviço às `resource_deps` do `web-e2e-ui`. O `ci-e2e` não muda: ele sobe o stack do Tilt sem citar recurso nenhum.

Nenhum JSON compartilhado (`package.json`, `turbo.json`, `tsconfig`) muda para adicionar um serviço.

## Como instanciar

1. Crie o repositório a partir do template, pelo botão **Use this template** no GitHub ou pela CLI:

   ```sh
   gh repo create <name> --template lcnascimento/boilerplate --private --clone
   cd <name>
   ```

2. Troque o marcador `boilerplate` pelo nome do projeto:

   ```sh
   make init NAME=<name>
   ```

   - `NAME` precisa casar com `^[a-z][a-z0-9-]*$` (ex.: `my-app`). Um nome inválido é rejeitado antes de qualquer arquivo ser tocado.
   - `boilerplate` vira `<name>` em todo arquivo versionado (módulos Go, escopo `@<name>/*`, rede docker, dashboards, `service.namespace`, cookie de idioma, servidor MCP) e `Boilerplate` vira o título derivado (`my-app` → `My App`).
   - Sai do `CLAUDE.md` a seção "Agent skills" e sai a pasta `docs/agents/` inteira.
   - O script exige a árvore git limpa, não faz commit nem instala nada e, depois de rodar com sucesso, se apaga junto com o alvo `init`, o workflow `ci-template` e esta seção do README.
   - Se algo der errado, `git checkout . && git clean -fd` volta ao estado anterior.

3. Revise o diff e siga os próximos passos que o script imprime: `pnpm install` (na raiz e em `frontend/`), `pnpm -C frontend format` (o nome novo muda o comprimento de algumas linhas, e o prettier quebra ou junta imports de novo), o setup do ai-foundation e o commit.

## Removendo o exemplo

Para começar sem o serviço `notes`:

```sh
make clean-example
```

- É opcional e independente do `make init`: rode antes, depois ou nunca.
- Apaga os arquivos do exemplo (`backend/services/notes/`, a coleção Bruno `Notes`, `openapi/notes.yaml`, as telas e a action do `notes`, `frontend/packages/api/src/notes/`, as mensagens `notes` e os cenários e2e dele) e todo trecho entre cercas `example:begin` / `example:end` nos arquivos compartilhados, este README incluído (cercas em comentário HTML). Falha se sobrar alguma cerca.
- Exige a árvore git limpa, não faz commit nem instala nada e, depois de rodar com sucesso, se apaga junto com o alvo `clean-example` e esta seção do README. Para desfazer antes do commit: `git checkout . && git clean -fd`.
- Tirar as cercas pode deixar código fora do formato do prettier (um import que passa a caber numa linha, um array vazio). Por isso os próximos passos que o script imprime são `pnpm -C frontend install && pnpm -C frontend format` e o commit.

O estado com zero serviços também é válido e fica verde: o `go.work` fica só com `quality` (`make lint` e `make test` passam, sem migrations para o squawk), o `tilt up` sobe o Postgres, a observabilidade e o `web`, o `openapi/` mantém a infra sem subir o Scalar, o app fica com `/`, `/profile/preferences` e o `not-found`, o e2e roda só o teste do shell e o `@boilerplate/api` fica só com o `core` (`request`, `Result` e `APIError`).
