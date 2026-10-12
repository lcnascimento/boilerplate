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
- **Docker**, para a observabilidade e o Scalar.
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

Ele sobe só os serviços Go, o `web` e as dependências deles. Sem a variável, o `tilt up` sobe tudo; outro valor é erro. A observabilidade fica em `backend/docker-compose.observability.yml`, que o Tilt só carrega fora do modo mínimo, e o `backend/docker-compose.yml` guarda o que todo stack sobe. Sem o Alloy no ar, os logs dos serviços que exportam por OTLP não aparecem em lugar nenhum.

Os logs dos serviços Go vão por OTLP para o Loki (`OTEL_LOGS_EXPORTER=otlp` no `backend/.env`), então eles aparecem no dashboard `logs` do Grafana, filtrados por `service_name`, e não no painel do Tilt.

O app só fala com o backend pelo servidor: o Tilt passa a `<NAME>_API_URL` de cada serviço para o `next dev`. Para rodar o `next dev` sem o Tilt, copie `frontend/apps/web/.env.example` para `frontend/apps/web/.env.local`. Sem essas variáveis, o servidor recusa subir.

### Lint e testes

**Backend** (em `backend/`, cada alvo roda em todos os módulos do `go.work`):

```sh
make lint          # golangci-lint (make lint.fix corrige o que der)
make test          # go test -race com cobertura
make mocks         # regenera internal/mocks/mocks.go a partir de cada contracts.go
make tidy          # go mod tidy
make vuln-check    # govulncheck (só local, fora do CI)
```

**Frontend** (em `frontend/`, via Turborepo):

```sh
pnpm lint
pnpm typecheck
pnpm test                # unit, componentes e as stories do ui como testes
pnpm build
pnpm test:e2e            # Playwright contra o stack real
pnpm format:check        # prettier (pnpm format corrige)
pnpm storybook           # Storybook do ui em http://localhost:6006
```

As stories do `ui` e o e2e rodam no Chromium do Playwright. Instale uma vez:

```sh
pnpm -C frontend --filter @boilerplate/web exec playwright install chromium
```

O `pnpm test:e2e` sobe um `next build && next start` na porta `3001` e o binário de cada serviço Go (build com o Go local), então não depende do Tilt e não conflita com ele.

Para depurar o e2e, o recurso `web-e2e-ui` do Tilt (manual: dispare-o na UI do Tilt) abre o [UI Mode](https://playwright.dev/docs/test-ui-mode) do Playwright em <http://127.0.0.1:9330>, com os mesmos servidores do `pnpm test:e2e`.

<!-- example:begin -->

O do `notes` sobe na porta `8081`, vazio a cada execução.

<!-- example:end -->

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

1. Crie `backend/services/<name>/` com dois módulos Go: `go.mod` (`github.com/lcnascimento/boilerplate/backend/services/<name>`) e `pkg/go.mod` (`.../<name>/pkg`, com o `replace` para ele no primeiro). O domínio vive em `pkg/domain`, `pkg/models`, `internal/application`, `internal/repository` e `internal/transport`, e cada construtor é registrado no fx uma vez em `bundle/`. O `Makefile` do serviço define o `OTEL_SERVICE_NAME`, o binário (`build.api.frontend` / `run.api.frontend`) e o `FRONTEND_PORT`; o `Tiltfile` dele, o recurso do Tilt.
   <!-- example:begin -->
   O caminho mais curto é copiar `backend/services/notes/` e trocar `notes` pelo novo nome nesses lugares e nos imports.
   <!-- example:end -->
2. Escolha uma porta livre (ex.: `8082`) e use-a no `.env.example` do serviço (`FRONTEND_PORT`), no `FRONTEND_PORT ?=` do `Makefile` e na `readiness_probe` e no link do `Tiltfile` do serviço.
   <!-- example:begin -->
   A `8080` é do `notes` e a `8081`, do e2e dele.
   <!-- example:end -->
3. Registre os dois módulos no `backend/go.work`:

   ```text
   use (
   	./services/<name>
   	./services/<name>/pkg
   )
   ```

4. Inclua o serviço no `backend/Tiltfile`: `include('./services/<name>/Tiltfile')`.
5. Rode `make tidy mocks lint test` em `backend/`. Um teste com `fx.ValidateApp` em `bundle/` quebra se o grafo do fx estiver incompleto.
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
11. e2e: em `frontend/apps/web/playwright.config.ts`, adicione o serviço ao array `webServer` (outra porta livre, `reuseExistingServer: false`) e passe `<NAME>_API_URL` no `env` do servidor web. No `.github/workflows/ci-e2e.yml`, adicione o build do binário antes do passo de e2e.

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

O estado com zero serviços também é válido e fica verde: o `go.work` fica só com `quality` (`make lint` e `make test` passam), o `tilt up` sobe a observabilidade e o `web`, o `openapi/` mantém a infra sem subir o Scalar, o app fica com `/`, `/profile/preferences` e o `not-found`, o e2e roda só o teste do shell e o `@boilerplate/api` fica só com o `core` (`request`, `Result` e `APIError`).
