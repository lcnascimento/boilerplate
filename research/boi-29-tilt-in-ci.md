# Rodar o stack do Tilt no CI (BOI-29)

Ticket: [BOI-29](https://linear.app/lcnascimento-boilerplate/issue/BOI-29), filho do mapa [BOI-25](https://linear.app/lcnascimento-boilerplate/issue/BOI-25). Este documento traz **fatos e trade-offs, sem escolha**: a escolha é do ticket de grilling "O e2e no CI" (BOI-36).

Versões de referência: Tilt v0.37.4 (a instalada localmente; o código-fonte citado é da tag `v0.37.4`), Next.js 16.4.0, Playwright 1.64.0.

## TL;DR

1. **`tilt ci` sozinho não serve para "subir o stack e depois rodar o Playwright"**: ele sai assim que todos os recursos ficam prontos, e ao sair **mata todo `serve_cmd`** (o `web` e o `notes-api-frontend`). Containers do docker compose sobrevivem. Há dois desenhos que funcionam com Tilt:
   - **(T1)** o próprio e2e vira um recurso do Tilt (um `local_resource` com `cmd` que roda o Playwright e `resource_deps` no `web` e nos serviços), ligado só no CI; o `tilt ci` só termina quando ele termina, e falha se ele falhar;
   - **(T2)** `tilt up` em background + `tilt wait --for=condition=Ready uiresource/<r> ...` + `pnpm test:e2e` num passo seguinte.
2. **Recursos com `auto_init=False` (o `web-e2e-ui`) são ignorados pelo `tilt ci`**, e recursos desabilitados também. Recursos que não interessam ao e2e (observabilidade, Scalar) **rodam** a menos que se restrinja a seleção.
3. Restringir: `tilt ci <nomes>` (habilita os nomes **e as `resource_deps` transitivas**), mas **um nome inexistente é erro** — o que quebra o caminho B do `ci-template` (`make clean-example` remove o `notes-api-frontend`). `config.parse`/`config.set_enabled_resources` no Tiltfile (com `config.tilt_subcommand == 'ci'` ou um flag) resolve isso dentro do Tiltfile. Não há seleção por label na linha de comando nesta versão.
4. **Readiness**: `local_resource` com `readiness_probe` só conta como pronto quando o probe passa; **docker compose conta como pronto quando o container sobe, ignorando healthcheck**, a menos que se use `docker_compose(..., wait=True)` (vira `docker compose up --wait`). Isso importa para o Postgres que o BOI-25 vai adicionar.
5. Custo: o job atual leva ~1m35s (passo `e2e` ~28s). Com Tilt entram a instalação do Tilt e, se o stack de observabilidade não for excluído, ~914 MB comprimidos de imagens (Grafana sozinho ~459 MB). Não há imagem construída localmente: todo o Docker é pull.
6. **`next dev` sob Playwright**: o Next compila rotas sob demanda, e o próprio Next recomenda testar contra build de produção. O probe TCP do `web` fica verde antes de qualquer rota compilar, então o primeiro acesso de cada rota paga a compilação dentro do timeout do teste.

## 1. O que o stack do Tilt sobe hoje

Fonte: `Tiltfile`, `backend/Tiltfile`, `backend/services/notes/Tiltfile`, `frontend/Tiltfile`, `openapi/Tiltfile`, `backend/docker-compose.yml`, `openapi/docker-compose.yml`.

| Recurso | Tipo | Readiness | Relevante para o e2e? |
| --- | --- | --- | --- |
| `root-deps` | `local_resource` `cmd='pnpm install'` (raiz) | job: precisa terminar | não (mas roda) |
| `alloy`, `prometheus`, `loki`, `tempo`, `pyroscope`, `grafana` | `dc_resource` (`backend/docker-compose.yml`) | container no ar | não |
| `notes-api-frontend` | `local_resource` `cmd=make build...`, `serve_cmd=make run...`, `allow_parallel=True` | HTTP `GET :8080/health`, `period_secs=5`, `failure_threshold=10` | sim (dentro de `example:begin/end`) |
| `scalar` | `dc_resource` (só se houver `openapi/*.yaml`) | container no ar | não |
| `frontend-deps` | `local_resource` `cmd='pnpm install'` (frontend) | job | sim (dep do `web`) |
| `web` | `local_resource` `serve_cmd='pnpm --filter @boilerplate/web dev'`, `serve_env` com `NOTES_API_URL=http://localhost:8080` | TCP `:3000`, `period_secs=5` | sim |
| `web-e2e-ui` | `local_resource`, `auto_init=False`, `TRIGGER_MODE_MANUAL` | TCP `:9330` | não |

Observações do repo:

- O e2e atual **não usa o Tilt**: `frontend/apps/web/playwright.config.ts` sobe `next build && next start --port 3001` e o binário do `notes` na `8081` (com `OTEL_SDK_DISABLED=true`) via `webServer`. Para usar o stack do Tilt, a config muda para `baseURL` `:3000`, `NOTES_URL` `:8080` e sem `webServer` (ou com `reuseExistingServer`).
- O `notes-api-frontend` do Tilt **não** desliga o OTel. Sem `.env` no CI (`backend/.env` é cópia local do `.env.example`, que liga `OTEL_LOGS_EXPORTER=otlp`), o SDK usa os defaults e tenta exportar para `localhost:4317`. Se o Alloy não subir no CI, o serviço exporta para o nada; **não verifiquei** se isso só gera log de erro ou atrasa/derruba algo (depende do go-kit) — precisa ser checado num spike.
- `local_resource` sem `allow_parallel=True` é serializado: o Tilt põe hold `HoldReasonIsUnparallelizableTarget` ([build_control.go](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/engine/buildcontrol/build_control.go#L352-L358)). `root-deps`, `frontend-deps` e `web` rodam um de cada vez entre si; só o `notes-api-frontend` tem `allow_parallel=True`.
- `notes-empty.spec.ts` apaga **todas** as notas do serviço antes de rodar. Hoje isso é inofensivo (o e2e tem seu próprio `notes` na 8081). Rodando contra o `notes` do Tilt (8080), localmente isso apaga os dados de dev. É um efeito do "e2e usa o stack do Tilt" que o BOI-36 / o ticket de isolamento precisam considerar.

## 2. `tilt ci` no GitHub Actions

### 2.1 Instalação

- Script oficial: `curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash`; ou o binário da release (`https://github.com/tilt-dev/tilt/releases/download/v<ver>/tilt.<ver>.linux.x86_64.tar.gz`) extraído em `/usr/local/bin` ([docs.tilt.dev/install](https://docs.tilt.dev/install.html)). O script pega o `master`; o tarball permite fixar versão. A doc não traz guia específico de CI nem action oficial.
- O repo não fixa a versão do Tilt em lugar nenhum (`.tool-versions` tem só `pnpm` e `golang`). Local: v0.37.4; a doc atual aponta v0.37.8.
- O runner `ubuntu-latest` já tem Docker com compose v2 (o `tilt` usa `docker compose` para os `dc_resource`). Sem recursos Kubernetes no Tiltfile, o `tilt ci` não exige cluster: o erro de cluster só derruba a sessão quando há recurso k8s habilitado ([status.go `hasClusterError`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/controllers/core/session/status.go)).

### 2.2 Como o `tilt ci` decide sair

`tilt ci --help`: "Exits with failure if any build fails or any server crashes. Exits with success if all tasks have completed successfully and all servers are healthy." Timeout padrão `--timeout 30m`.

No código ([`internal/controllers/core/session/status.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/controllers/core/session/status.go), `processExitCondition`, e [`conv.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/controllers/core/session/conv.go)):

- Cada recurso vira targets: um **job** (o `cmd`/build) e, se houver, um **server** (`serve_cmd` ou runtime do compose).
- Um target **sem estado** (nem waiting, nem active, nem terminated) é **pulado** — comentário no código: "the target has not been requested to run, e.g. auto_init=False". Recursos desabilitados também ficam sem esses estados. ⇒ o `web-e2e-ui` (`auto_init=False`) **não bloqueia nem é executado** pelo `tilt ci`.
- Qualquer target terminado com erro ⇒ sai com erro na hora.
- Sai com sucesso quando não há nada `waiting`, `not ready` ou `retrying`.
- `local_resource` com `serve_cmd`: `Active.Ready = lrs.Ready`, ou seja, **respeita o `readiness_probe`**.
- Docker compose: target genérico com `Ready: true` assim que o runtime está OK — comentário: "this also applies to Docker Compose, since we don't support its health checks". **Healthcheck do compose não segura o `tilt ci`**. O único jeito de esperar healthcheck é `docker_compose(..., wait=True)`, que acrescenta `--wait` ao `docker compose up` ([`internal/tiltfile/docker_compose.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/tiltfile/docker_compose.go), [`internal/dockercompose/client.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/dockercompose/client.go#L135-L139)). O mesmo vale para `resource_deps` em cima de um `dc_resource` (ex.: serviço Go dependendo do Postgres): "pronto" = container de pé, não "aceita conexões".
- Timeouts configuráveis no Tiltfile com `ci_settings(timeout=..., readiness_timeout=...)`; `readiness_timeout` padrão **5 min** (um server ativo e não pronto por 5 min derruba o CI) e `timeout` padrão 30 min ([`pkg/model/ci.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/pkg/model/ci.go), [`cisettings`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/tiltfile/cisettings/ci_settings.go)).

### 2.3 O que acontece ao sair — e por que isso decide o desenho

- `tilt up --help`: "When you exit Tilt (using Ctrl+C), Kubernetes resources and Docker Compose resources continue running ... Any long-running local resources--i.e. those using serve_cmd--are terminated when you exit Tilt." A doc de CI acrescenta que port-forwards somem após o `tilt ci` ([docs.tilt.dev/ci](https://docs.tilt.dev/ci.html)).
- Logo, `tilt ci && pnpm test:e2e` **não funciona**: quando o Playwright começa, `web` (3000) e `notes` (8080) já morreram; só o compose (observabilidade, Scalar, futuro Postgres) continua.

Os dois desenhos com Tilt:

**(T1) O e2e como recurso do Tilt**, rodado por `tilt ci`:
- um `local_resource('web-e2e', cmd='pnpm --filter @boilerplate/web test:e2e', resource_deps=['web', <serviços>])`; job que precisa terminar, e erro no job derruba o `tilt ci` com código ≠ 0.
- Para não rodar a cada `tilt up` local, liga-se só no CI: `auto_init = config.tilt_subcommand == 'ci'` (o Tiltfile expõe `config.tilt_subcommand`, valores como `up`, `ci`, `down` — [`internal/tiltfile/config/config.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/tiltfile/config/config.go), [`pkg/model/tilt_subcommand.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/pkg/model/tilt_subcommand.go)) ou por `os.getenv('CI')`.
- Prós: um comando (`tilt ci`) para local e CI; o Tilt garante a ordem e os probes. Contras: saída do Playwright vira log do Tilt (o reporter `github` e o upload do `playwright-report/` continuam funcionando porque o relatório é escrito em disco); o timeout global do `tilt ci` (30 min) passa a cobrir os testes; o `readiness_timeout` não se aplica a jobs.
- O `web-e2e-ui` já existente segue o mesmo padrão (recurso manual); os dois convivem.

**(T2) `tilt up` em background + `tilt wait`**:
- `tilt up` sem TTY cai sozinho no modo stream (`initialTermMode`: `!isTerminal ⇒ TerminalModeStream`, [`internal/cli/up.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/cli/up.go#L103-L116)); em Actions roda como `tilt up <recursos> &` ou passo com `nohup`.
- `tilt wait --for=condition=Ready uiresource/web uiresource/notes-api-frontend --timeout=<d>` (comando marcado "Experimental" no help; timeout padrão 30s, precisa ser aumentado).
- Depois `pnpm test:e2e` num passo normal (logs e reporter do Playwright como hoje) e, opcionalmente, `tilt down` (desnecessário num runner efêmero).
- Prós: Playwright continua um passo próprio do workflow. Contras: o workflow precisa listar os recursos a esperar (o mesmo problema de nomes do caminho B, §3); falha de servidor depois do `wait` não interrompe nada sozinho; o `tilt up` não sai com erro, então o log do Tilt precisa ir para artefato para diagnóstico.

### 2.4 Tempo de boot (estimativa, não medido)

Não rodei `tilt ci` (o Tilt do usuário pode estar na 10350). Base medida: a última execução do `ci-e2e` em `main` ([run 38156518188](https://github.com/lcnascimento/boilerplate/actions/runs/38156518188)) levou ~1m33s de job: setup pnpm/node ~12s, `pnpm install` 4s, setup-go 15s, build do `notes` 1s, `playwright install --with-deps chromium` 29s, passo `e2e` 28s.

O que o Tilt acrescenta:
- instalação do Tilt (download de um tarball);
- `root-deps` e `frontend-deps` rodam `pnpm install` de novo (rápidos com store aquecido, mas **serializados** com o `web`, §1);
- pull das imagens compose habilitadas. Tamanhos comprimidos amd64 no Docker Hub (API `hub.docker.com/v2/repositories/<repo>/tags/<tag>`, consultada em 2026-10-11): `grafana/grafana:13.2.3` 459 MB, `grafana/alloy:v1.20.1` 211 MB, `prom/prometheus:v3.15.0` 108 MB, `grafana/pyroscope:2.4.0` 56 MB, `grafana/loki:3.7.8` 45 MB, `grafana/tempo:2.10.8` 35 MB — **~914 MB** só para a observabilidade. Para comparação, `postgres:18-alpine` 114 MB;
- o `web` fica pronto (TCP 3000) em segundos, mas cada rota compila no primeiro acesso (§5).

O número real só sai de um spike no Actions; vale medir com e sem a observabilidade.

## 3. Restringir aos recursos necessários

Fonte: [`internal/tiltfile/config/resources.go`](https://github.com/tilt-dev/tilt/blob/v0.37.4/internal/tiltfile/config/resources.go), [docs.tilt.dev/tiltfile_config](https://docs.tilt.dev/tiltfile_config.html), `tilt up --help`.

- **Args posicionais** (`tilt ci web notes-api-frontend`): sem `config.parse` no Tiltfile, cada arg é um nome de recurso. O Tilt habilita esses recursos **mais todas as `resource_deps` transitivas** (`addManifestAndDeps`). Ex.: `tilt ci web` traz o `frontend-deps`; o `root-deps` não entra (ninguém depende dele).
- **Nome inexistente é erro**: "You specified some resources that could not be found: ... Is this a typo?". Consequência direta: o workflow chamado pelo `ci-template` no caminho **B** (`make clean-example` → zero serviços) não pode passar `notes-api-frontend`. O `ci-e2e.yml` hoje resolve isso com `if: hashFiles('backend/services/notes/Makefile') != ''` e com cercas `example:begin/end` que o `clean-example.sh` apaga também em YAML; o mesmo mecanismo serve para uma lista de recursos no workflow, ou a lista vai para dentro do Tiltfile.
- **`config.set_enabled_resources([...])`** no Tiltfile "trumps everything" (ignora os args). Com `config.parse()` + `config.define_string_list(..., args=True)`, os args deixam de ser nomes de recurso e passam a ser o que o Tiltfile definir (ex.: um flag `--e2e`).
- **Labels**: o matcher só compara nomes; não há `tilt ci -l <label>` nesta versão. Para "tudo com label X" o Tiltfile teria que montar a lista (não há API pública no Starlark para ler os labels dos recursos já registrados; seria preciso manter a lista em paralelo, ex.: cada `include` de serviço anexando seu recurso a uma lista do e2e).
- Ideia que respeita a estrutura atual: cada Tiltfile de serviço já é incluído dentro de `example:begin/end`; se cada serviço declarar seus recursos de e2e numa lista compartilhada (ou o `web` tiver `resource_deps` nos serviços), `tilt ci web` puxaria tudo por transitividade, e o caminho B continuaria válido sem nomes no workflow. Hoje o `web` **não** tem `resource_deps` nos serviços (só no `frontend-deps`).
- Recursos manuais e `auto_init=False` já ficam fora do `tilt ci` (§2.2), sem precisar restringir.

## 4. Cache para manter o job rápido

- **Go**: `actions/setup-go` já cacheia módulos e build cache (`cache-dependency-path: backend/**/go.sum`). O `make build.api.frontend` do Tilt usa o mesmo `go build`, então aproveita o mesmo cache.
- **pnpm**: `actions/setup-node` com `cache: pnpm` já cacheia o store do frontend. Com o Tilt, o `root-deps` instala também a raiz (`package.json` raiz; o `prepare` roda `lefthook install`) — o `cache-dependency-path` teria de incluir o `pnpm-lock.yaml` da raiz.
- **Next.js (`next dev`)**: desde o 16.1 o cache de filesystem do Turbopack para dev é padrão e fica em `.next/dev/cache/turbopack`; "Restarting the dev server reuses the previous compilation" ([turbopackFileSystemCache](https://nextjs.org/docs/app/api-reference/config/next-config-js/turbopackFileSystemCache)). Cachear esse diretório com `actions/cache` é a alavanca para reduzir a compilação sob demanda no CI (efeito não medido). O cache do `next build` (`.next/cache`) deixa de importar se o e2e não fizer build — mas hoje o turbo roda `build` antes de `test:e2e` (`turbo.json`: `test:e2e.dependsOn: ["^build", "build"]`), o que precisaria ser revisto para não pagar um `next build` inútil.
- **Imagens Docker**: não há imagem construída localmente (todas são pull), então não há cache de build. Opções: não subir o que o e2e não usa (§3) ou `docker save`/`actions/cache`/`docker load` — esta última costuma empatar com o pull do registry para imagens grandes; não medido aqui.
- **Browsers do Playwright**: a doc do Playwright **não recomenda** cachear: "the amount of time it takes to restore the cache is comparable to the time it takes to download the binaries" ([playwright.dev/docs/ci](https://playwright.dev/docs/ci)).

## 5. Alternativa sem Tilt e o custo de dois caminhos

Alternativa: `docker compose up -d <deps>` (ex.: Postgres, com `--wait` se tiver healthcheck) + processos em background (o `webServer` do Playwright, como hoje, ou `&` no workflow). É o padrão citado no BOI-25 a partir do projeto `remarks` para os testes de componente.

| | Stack do Tilt (T1/T2) | Compose + processos (sem Tilt) |
| --- | --- | --- |
| Fonte da verdade de "como sobe" | Tiltfiles (uma só, local = CI) | Tiltfiles para dev **e** `playwright.config.ts`/workflow para e2e |
| Ao adicionar um serviço | o `include` do Tiltfile já cobre (se a seleção de recursos for derivada, §3) | o README "Como adicionar um serviço" ganha mais passos: porta de e2e, `webServer`, passo de build no workflow (como hoje para o `notes`) |
| Drift | baixo: env, portas e comandos são os mesmos | médio: duas listas de env/portas (hoje `serve_env` do Tilt vs. `env` do `webServer`; 8080 vs. 8081; 3000 vs. 3001) |
| Dependência extra no CI | instalar o Tilt | nenhuma |
| Diagnóstico | logs do Tilt (stream) | logs de cada processo no Playwright/Actions |
| Isolamento dos dados de dev | e2e compartilha o `notes`/Postgres de dev localmente | e2e tem processos próprios (hoje: `notes` na 8081 com store próprio) |
| Requisito do BOI-25 ("local e CI usam o mesmo caminho", web do Tilt) | atende | não atende (é o modelo atual) |

O próprio mapa BOI-25 fixou que o e2e usa o web do Tilt na 3000 e o `notes` na 8080, local e CI; a alternativa sem Tilt contraria isso e mantém dois caminhos.

## 6. Riscos de `next dev` em vez de `next build && next start`

Fatos:
- "`next dev` compiles routes in your application as you open or navigate to them" ([Local development](https://nextjs.org/docs/app/guides/local-development)). Turbopack é o bundler padrão do dev.
- O guia oficial de Playwright do Next: "We recommend running your tests against your production code to more closely resemble how your application will behave" ([Testing: Playwright](https://nextjs.org/docs/app/guides/testing/playwright)).
- O probe do `web` é TCP na 3000 (`frontend/Tiltfile`, escolhido para não logar a página a cada sucesso): fica verde quando o servidor escuta, **antes** de qualquer rota compilar.
- A config do Playwright usa `fullyParallel: true`, `retries: 2` no CI, `trace: "on-first-retry"`, sem `workers` explícito; a doc do Playwright recomenda `workers: 1` no CI para estabilidade ([playwright.dev/docs/ci](https://playwright.dev/docs/ci)).

Riscos que decorrem desses fatos (não medidos):
- **Primeiro acesso lento**: o primeiro `page.goto` de cada rota inclui a compilação; com o timeout padrão de teste (30s) e de `expect` (5s), uma rota pesada no runner pode estourar, e os `retries: 2` mascaram como "flaky". Mitigações possíveis: aquecer as rotas antes (requisições a cada rota após o `web` ficar pronto), probe HTTP numa rota real, cache do Turbopack dev (§4), timeouts maiores só no CI.
- **Compilação concorrente**: com `fullyParallel`, vários workers batem em rotas ainda não compiladas ao mesmo tempo.
- **Comportamento diferente de produção**: o e2e deixa de exercitar o build de produção (minificação, otimizações, o que o `next build` valida). O `ci-frontend` roda `pnpm turbo run build`, então o build de produção continua compilando no CI; mas o e2e atual é o único lugar que **executa** o `next start`. Com `next dev`, um erro que só aparece em runtime de produção deixa de ser pego.
- **Ganho**: some o `next build` do caminho crítico do e2e (hoje ele roda dentro do `webServer`, e o turbo ainda roda outro `build` antes por causa do `dependsOn`).

## 7. Interação com o `ci-template` (caminhos A e B)

Fonte: `.github/workflows/ci-template.yml`, `scripts/init.sh`, `scripts/clean-example.sh`.

- O `ci-e2e.yml` é chamado pelo `ci-template` com `bootstrap: make init ...` (A) e `make clean-example && ... && make init ...` (B); o workflow tem de funcionar nos dois.
- Caminho A: nomes de recursos não mudam com o `make init` (ele troca o marcador "boilerplate", não os nomes `web`/`notes-api-frontend`).
- Caminho B: `notes-api-frontend` e `scalar` deixam de existir (o `openapi/Tiltfile` não adiciona recurso sem `*.yaml`); sobram `root-deps`, a observabilidade, `frontend-deps`, `web`, `web-e2e-ui`. Qualquer nome fixo de serviço no workflow precisa estar dentro de `example:begin/end` ou ser derivado do Tiltfile (§3). `tilt ci` sem args funciona nos dois caminhos, mas sobe a observabilidade.
- Após o B, `Makefile` raiz e `scripts/` não existem; nada no desenho com Tilt depende deles.

## Fontes

- Repo (em `main`, e12b42f): `Tiltfile`, `backend/Tiltfile`, `backend/services/notes/{Tiltfile,Makefile,.env.example}`, `backend/.env.example`, `backend/docker-compose.yml`, `frontend/Tiltfile`, `frontend/turbo.json`, `frontend/apps/web/{playwright.config.ts,next.config.ts,package.json,e2e/}`, `openapi/{Tiltfile,docker-compose.yml}`, `.github/workflows/{ci-e2e,ci-template}.yml`, `scripts/{init,clean-example}.sh`, `.tool-versions`.
- Tilt v0.37.4: `tilt ci --help`, `tilt up --help`, `tilt wait --help`; código em <https://github.com/tilt-dev/tilt/tree/v0.37.4>: `internal/controllers/core/session/{status,conv}.go`, `internal/tiltfile/config/{resources,config}.go`, `internal/tiltfile/cisettings/ci_settings.go`, `pkg/model/ci.go`, `internal/tiltfile/docker_compose.go`, `internal/dockercompose/client.go`, `internal/engine/buildcontrol/build_control.go`, `internal/tiltfile/local_resource.go`, `internal/cli/up.go`.
- Docs Tilt: <https://docs.tilt.dev/ci.html>, <https://docs.tilt.dev/tiltfile_config.html>, <https://docs.tilt.dev/install.html>.
- Docs Next.js 16.4: <https://nextjs.org/docs/app/guides/testing/playwright>, <https://nextjs.org/docs/app/guides/local-development>, <https://nextjs.org/docs/app/api-reference/config/next-config-js/turbopackFileSystemCache>.
- Docs Playwright: <https://playwright.dev/docs/ci>.
- GitHub Actions: run 38156518188 do `ci-e2e` (tempos por passo via `gh run view`).
- Docker Hub API de tags (tamanhos comprimidos amd64), consultada em 2026-10-11.
