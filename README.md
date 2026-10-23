# Boilerplate

Template de projetos pessoais: um monorepo com backend em Go, frontend em Next.js, specs OpenAPI e observabilidade local, tudo de pé com um `tilt up`.

<!-- example:begin -->

Ele traz um serviço de exemplo, o `notes`, que atravessa todas as camadas: uma nota criada na tela passa pela Server Action, pelo cliente tipado e pelo serviço Go, e aparece com trace, log e métrica no Grafana.

<!-- example:end -->

## Stack

| Área | O que tem |
| --- | --- |
| Backend (`backend/`) | Go (versão no `go.work`) com [uber/fx](https://github.com/uber-go/fx) e [go-kit](https://github.com/lcnascimento/go-kit) (`errors`, `http`, `o11y`, `env`, `util`, `validator`), `golangci-lint`, `mockgen`, `govulncheck` e coleção [Bruno](https://www.usebruno.com/) |
| Observabilidade local (`backend/`) | Alloy, Prometheus, Loki, Tempo, Pyroscope e Grafana via Docker Compose, com os dashboards `overview` (home), `kafka` e `postgres`, um dashboard gerado por serviço, e os alertas como regras do Prometheus, com runbook em `docs/runbooks/` |
| Banco local (`backend/`) | Postgres 18 via Docker Compose, compartilhado: um database e um role por serviço. Acesso com [pgx](https://github.com/jackc/pgx) (`pgxpool`, instrumentado com [otelpgx](https://github.com/exaring/otelpgx)) e queries tipadas pelo [sqlc](https://sqlc.dev/). Migrations com [goose](https://github.com/pressly/goose) (como biblioteca) e lint delas com [squawk](https://squawkhq.com/) |
| Mensageria local (`backend/`) | Kafka 4.3 (KRaft, nó único, sem auto-create de tópicos), Confluent Schema Registry e [Redpanda Console](https://github.com/redpanda-data/console) via Docker Compose |
| Frontend (`frontend/`) | Next.js 16 (App Router, React Compiler), next-intl 4 (pt-BR e en), shadcn + Tailwind 4 + Storybook, Zod, Vitest, Testing Library e Playwright |
| Monorepo do frontend | pnpm + Turborepo: app `apps/web`, pacotes `api`, `i18n`, `types`, `ui` e `utils`, configs compartilhadas em `tooling/{eslint,typescript}` |
| Identidade (`iam`) | [Better Auth](https://www.better-auth.com/) no app web (email e senha, sessão, Organizations, Members e Roles) sobre o database `iam`, e o módulo Go `backend/services/iam/pkg`, que valida o JWT de Actor emitido pelo web e checa as Permissions. Faz parte fixa da stack: o `make clean-example` não o remove |
| Contrato (`openapi/`) | Uma spec OpenAPI 3.1 por serviço, servida pelo [Scalar](https://scalar.com/) e validada pelo [vacuum](https://quobix.com/vacuum/) |
| Tooling raiz | [Tilt](https://tilt.dev/), lefthook (lint no pre-commit, `format.check` e lint no pre-push) e commitlint (Conventional Commits no commit-msg) |
| CI (`.github/workflows/`) | `quality_gate` (jobs `format`, `lint`, `test` e `e2e`, cada um num alvo do Makefile da raiz), chamado pelo `pull_request` (com o `lint.breaking` contra a base do PR) e pelo `release` (push na `main`), sem filtro de caminhos |

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

O `make -C backend setup` chama o `make -C backend env`, que cria `backend/.env` e o `.env` de cada serviço (`backend/services/<name>/.env`) quando eles ainda não existem e, quando já existem, acrescenta só as chaves novas do `.env.example`, sem mexer nos valores que você mudou nem nas chaves que você comentou. O `env` também roda sozinho, sem instalar as ferramentas, e o `tilt up` o chama a cada carga do Tiltfile.

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
| Kafka (bootstrap server para os serviços no host) | `localhost:9092` |
| Schema Registry | <http://localhost:8081> |
| Redpanda Console (tópicos, mensagens e schemas) | <http://localhost:8085> |
| Prometheus, Loki, Tempo, Pyroscope, Alloy | `9090`, `3100`, `3200`, `4040`, `12345` (links na UI do Tilt) |

<!-- example:begin -->

E cada serviço Go na sua porta: o `notes` em <http://localhost:8080> (`/health`, `/ping`, `/notes/v1/notes`), o forwarder do `notes` em <http://localhost:8091> (`/health`, `/ping`) e o consumer do `audit`, que grava cada Integration Event de `notes.note.v1` como Audit Entry no banco `audit`, com health em <http://localhost:8092> (`/health`, `/ping`).

<!-- example:end -->

Para um stack leve, sem observabilidade, Redpanda Console, Scalar nem `playwright`, suba o modo mínimo:

```sh
TILT_STACK=minimal tilt up
```

Ele sobe só o Postgres, o Kafka, o Schema Registry, os serviços Go (com os recursos de banco), o `web` e as dependências deles. Sem a variável, o `tilt up` sobe tudo; outro valor é erro. A observabilidade fica em `backend/docker-compose.observability.yml`, que o Tilt só carrega fora do modo mínimo, e o `backend/docker-compose.yml` guarda o que todo stack sobe, exceto o Redpanda Console, que fica no profile `console` do compose. Sem o Alloy no ar, os logs dos serviços que exportam por OTLP não aparecem em lugar nenhum.

O Alloy também lê os logs dos containers da rede `boilerplate` do compose (containers fora dela, como os de outros projetos, ficam de fora) e os grava no Loki no mesmo formato dos logs OTLP dos serviços: `service_name` é o serviço do compose, a linha é só a mensagem, o nível vira `severity_text` (e o `detected_level` do Grafana) e os outros campos viram structured metadata. O `otelcol.processor.transform "containers"` de `backend/config/alloy/config.alloy` reconhece JSON, logfmt, o formato log4j do Kafka e do Schema Registry e o do Postgres; uma linha em outro formato, como a continuação de um stack trace, chega como está.

O servidor do `web` loga com o pino, que o `PinoInstrumentation` manda por OTLP com `trace_id` e `span_id`: o `onRequestError` loga cada request que falha, e o `request()` de `@boilerplate/api` loga cada chamada à API que falha do lado do servidor (erro de rede, 5xx ou resposta fora do contrato); um 4xx é resposta esperada e não vira log. O Tilt só sobe o `web` depois do Alloy, porque o exporter OTLP desiste em cerca de 10s e um log escrito antes disso se perde.

Os logs dos serviços Go vão por OTLP para o Loki (`OTEL_LOGS_EXPORTER=otlp` no `backend/.env`), então eles aparecem na seção Logs do dashboard de cada serviço no Grafana e no Logs Drilldown, filtrados por `service_name`, e não no painel do Tilt.

No browser, o `web` manda RUM pelo [Grafana Faro](https://grafana.com/docs/grafana-cloud/monitor-applications/frontend-observability/), iniciado em `frontend/apps/web/instrumentation-client.ts`: Web Vitals, erros de JS, `console` a partir de `warn`, page loads e os traces de `fetch`. Fora do modo mínimo, o Tilt passa `NEXT_PUBLIC_FARO_URL=http://localhost:12347/collect`, o `faro.receiver` do Alloy; sem essa variável, o Faro não inicializa. `NEXT_PUBLIC_FARO_SAMPLE_RATE` (de 0 a 1, padrão 1) define a fração de sessões amostradas, e apontar para o Grafana Cloud é só trocar a URL. O Alloy grava os sinais do Faro no Loki com `service_name="web"`, cada campo como structured metadata (`kind`, `type`, `value_lcp`...), e manda os traces ao Tempo. Como nos logs dos serviços, a linha é um resumo curto (`fetch GET /notes 200 24ms`, `web-vitals INP 56ms good /notes`, `navigation /notes 120ms`, `session_start`, `console.error: <mensagem>`, `TypeError: <mensagem>`) e o nível vira `severity_text`: o do `console` para os logs, `ERROR` para exceções e `fetch` com status 5xx, `WARN` para um Web Vital `poor` e `INFO` para o resto. O `fetch` leva o `trace_id` do span do browser, então o Grafana abre o trace a partir da linha. O `otelcol.processor.filter "faro"` descarta os `faro.performance.resource` de `fetch` e XHR, que só repetem o `faro.tracing.fetch`; um sinal num formato que o `otelcol.processor.transform "faro"` não conhece chega com a linha logfmt original. Com o `experimental.clientTraceMetadata` do `next.config.ts`, o page load entra no trace do SSR, e uma Server Action continua o trace do browser até a API Go e o Kafka. A seção `browser` do dashboard `web` mostra o p75 de LCP, INP, CLS, FCP e TTFB, os erros de JS e os page loads; o `error.tsx` e o `global-error.tsx` reportam o erro ao Faro pelo hook `useReportBoundaryError` (`shared/faro/`).

O app só fala com o backend pelo servidor: o Tilt passa a `<NAME>_API_URL` de cada serviço para o `next dev`. O `make -C frontend env` (chamado pelo `make -C frontend setup` e a cada carga do Tiltfile) cria `frontend/apps/web/.env.local` a partir do `.env.example` e, quando ele já existe, acrescenta só as chaves novas, como o `make -C backend env`. É dele que o `next dev` sem o Tilt, o `iam:token` e o e2e leem as variáveis; o `env` do Tilt tem precedência. Sem essas variáveis, o servidor recusa subir.

### Mensageria

O `backend/docker-compose.yml` sobe um broker Kafka, o Schema Registry e o Redpanda Console, nessa ordem; no Tilt, `kafka → schema-registry → kafka-console`. O auto-create de tópicos está desligado: produzir num tópico que não existe falha, em vez de criar um tópico de uma partição. Cada serviço declara os próprios tópicos num `make kafka.topics` idempotente, que o Tilt roda como o recurso `<name>-topics` depois do `schema-registry`. Os nomes de tópico usam só pontos (`<serviço>.<agregado>.v<N>`), nunca `_`. Os serviços Go no host falam com o broker em `localhost:9092`; containers e ferramentas dentro da rede `boilerplate` usam `kafka:19092`. Tópicos e schemas ficam no volume `kafka_data` e sobrevivem a `tilt down`/`tilt up`. Sem o Tilt, em `backend/`: `docker compose up -d --wait kafka schema-registry` (mais `--profile console` para o Console).

<!-- example:begin -->

O `notes` grava cada Integration Event num outbox, na mesma transação da Note, e o runtime `notes-worker-forwarder` (`cmd/worker/forwarder`) leva o outbox ao tópico `notes.note.v1`, uma mensagem por vez, em ordem de commit, com o `NoteID` como key. Ele tenta publicar para sempre e nunca pula uma mensagem: com o Kafka fora do ar, a API continua aceitando escritas e o outbox cresce até o broker voltar. Várias réplicas podem rodar juntas, com uma ativa e as outras esperando. O outbox, o forwarder e a limpeza das mensagens já publicadas há mais de 7 dias vêm do pacote [`outbox`](https://github.com/lcnascimento/go-kit/tree/main/outbox) do go-kit, que todo serviço com Integration Events usa do mesmo jeito. O start falha se o broker não responder ou se o tópico não existir (`TOPIC_MISSING`: rode `make kafka.topics`). O `/health` na `8091` olha o Postgres e o forwarder, nunca o broker.

<!-- example:end -->

Os schemas Protobuf dos Integration Events ficam no Schema Registry, um subject por mensagem (`RecordNameStrategy`: o nome completo da mensagem, igual ao `ce_type`). Cada serviço que publica eventos registra os próprios num `make schemas.register` idempotente, que aplica `FULL_TRANSITIVE` em cada subject e registra o `.proto` normalizado; o Tilt o roda como o recurso `<name>-schemas` depois do `schema-registry`, e de novo quando um `.proto` muda. Uma mudança incompatível faz o registro falhar. Os schema IDs mudam entre ambientes e a cada `docker compose down -v`, então nada no código depende do valor deles.

Fora do modo mínimo, o Alloy lê do broker o lag de cada consumer group e os high-water marks de cada tópico (`prometheus.exporter.kafka`), sem instrumentar o Kafka, e só sobe depois do `kafka` saudável. O dashboard `kafka` do Grafana junta esses números às métricas `messaging.*` dos serviços, em seções Produce, Consume e DLQ, com o estado dos alertas de mensageria e os brokers no ar no topo.

Os alertas são regras do Prometheus geradas pelo `o11ygen` em `backend/config/prometheus/rules/`: um arquivo por serviço (um grupo por serviço) e o `infra.yaml` (grupos `kafka` e `postgres`). Cada processo herda os alertas dos componentes que declara: todo processo de longa duração herda `ServiceTelemetryAbsent` e `CriticalLogRecorded`, um processo com `WithDBPool()` herda `DBPoolSaturated`, um processo com `WithActorAuth()` herda `AuthenticationFailuresHigh` (a taxa de 401 sobre as requisições) e ganha a linha "Autenticação e autorização" no dashboard do serviço, um `consumer` herda `ConsumerLagHigh` e `DLQMessageReceived` do consumer group declarado em `WithConsumerGroup(...)`, e um `worker` herda `OutboxAgeAbsent`. A infraestrutura tem `KafkaBrokersDown`, `PostgresDown` e `PostgresConnectionsNearMax`. Todo alerta leva os labels `service` (o `service.name` do processo, ou `kafka`/`postgres`) e `severity` (`critical`: olhar já; `warning`: olhar em horas), e as anotações `summary`, `description`, `runbook_url` e `dashboard_url`. O runbook de cada alerta fica em `docs/runbooks/<Alerta>.md`.

Os logs também viram métricas: o `o11ygen` gera recording rules do ruler do Loki em `backend/config/loki/rules/fake/` (tenant `fake`, um arquivo e um grupo por serviço), que o Loki grava no Prometheus por `remote_write` a cada minuto: `service:log_records:rate1m` (logs/s por `service` e `level`, a partir do `detected_level`) e `service:log_records_critical:rate1m` (logs com `severity_number >= 21`, os níveis `Critical` e `Fatal` do go-kit). O `CriticalLogRecorded` dispara no primeiro log crítico de um processo. Um processo de longa duração pode declarar um alerta de log extra com `WithLogAlert` na declaração do serviço: `service.Daemon(...).WithLogAlert(&service.LogAlert{Name, Pipeline, MaxRecordsPer5m, Severity, Summary, Description})`, em que `Pipeline` são os estágios LogQL aplicados aos logs do processo (por exemplo `|= "payment declined"`) e o alerta dispara com mais de `MaxRecordsPer5m` logs em 5 minutos. Como todo alerta, ele precisa de um runbook em `docs/runbooks/<Name>.md` e de um teste `promtool` em `backend/config/prometheus/tests/`.

Não há Alertmanager nem contact point: os alertas aparecem só leitura em Alerting no Grafana (datasource Prometheus com `manageAlerts`) e nos painéis de alertas dos dashboards.

Cada processo também herda os SLOs padrão dos componentes que declara, todos com janela de 30 dias: um `api` herda `api-availability` (99,9% das requisições sem 5xx) e `api-latency` (99% em até 250ms), um `consumer` herda `consumer-success` (99,9% das tentativas de processamento sem erro) e `consumer-latency` (99% em até 1s), e um `worker` herda `outbox-freshness` (99% dos minutos com a pendente mais antiga do outbox em até 30s). As rotas `/ping` e `/health` ficam fora dos SLOs de API, e um `4xx` não queima budget. O `o11ygen` escreve uma spec do [Sloth](https://sloth.dev/) por serviço em `backend/config/sloth/` e roda o `sloth generate` (instalado pelo `make setup`), que escreve as recording rules e os alertas de burn-rate multi-janela em `backend/config/prometheus/slo/`: `SLOBurnRate` com `severity: critical` (page) ou `warning` (ticket), o label `service` do processo e o `sloth_slo` do SLO, e o runbook `docs/runbooks/SLOBurnRate.md`. Um SLO extra, de uma rota por exemplo, entra em `SLOs` na declaração do serviço, com o processo que o emite e um `service.EventsSLI{ErrorQuery: erro, TotalQuery: total}` ou `service.RawSLI{ErrorRatioQuery: razão}` cujas queries usam `{{.window}}`.

Cada serviço declarado ganha um dashboard gerado, com UID `boilerplate-<serviço>` (o mesmo do `dashboard_url` dos alertas dos processos dele), sem variável `$service`. No topo ficam o SLI atual, o budget restante de 30 dias e o burn rate de cada SLO, lidos das recording rules do Sloth, e a lista dos alertas do serviço. Abaixo vem uma seção por componente declarado: `api` (taxa, % de 5xx, latência p50, p90 e p99 total e por rota, um painel por percentil, 4xx/5xx por rota), `consumer` (mensagens/s com sucesso e falha, duração p50, p90 e p99, retentativas, lag do group, envios à DLQ por causa) e `worker` (idade da pendente mais antiga, envios/s, duração do envio p50, p90 e p99, limpeza, réplicas reportando), com o limite do SLO desenhado nos painéis de latência e frescor e exemplars nos de latência. Por fim, Runtime (goroutines, memória, GC) e Banco (pool pgx e conexões no servidor por `application_name`), colapsadas, e Logs (volume por nível e logs de erro), todas filtradas pela variável `$process`. Os links do topo abrem os apps Drilldown de Traces, Logs e Profiles já filtrados pelos processos do serviço e pelo intervalo, e cada série do painel de latência por rota abre os traces lentos daquela rota.

<!-- example:begin -->

No exemplo, o `audit-consumer` herda o alerta de lag e o de DLQ do group `audit`, e o `notes-worker-forwarder` herda o `OutboxAgeAbsent`, que dispara quando ele não reporta a idade do outbox do `notes` por 5 minutos, já que sem forwarder ninguém esvazia o outbox. A seção `worker` do dashboard `notes` mostra a idade da pendente mais antiga, os envios ao Kafka e a limpeza do outbox.

<!-- example:end -->

### Banco de dados

Um único container `postgres` (em `backend/docker-compose.yml`) atende todos os serviços; cada serviço tem um database e um role com o nome dele, e o database pertence ao role. No Tilt, cada serviço encadeia `postgres → <name>-db → <name>-migrate → <name>-api-*`, então a API só sobe com o banco criado e migrado. Os dados ficam no volume `postgres_data` e sobrevivem a `tilt down`/`tilt up`. Sem o Tilt, suba só o banco com `docker compose up -d --wait postgres` em `backend/`.

O Postgres sobe com `shared_preload_libraries=pg_stat_statements`, e o recurso `postgres-extensions` do Tilt roda, a cada `tilt up`, um `CREATE EXTENSION IF NOT EXISTS pg_stat_statements` no database `postgres` (sem o Tilt: `docker compose run --rm --no-deps postgres-extensions` em `backend/`). Fora do modo mínimo, o Alloy lê as métricas do servidor (`prometheus.exporter.postgres`, conectado só ao database `postgres`) e descarta as séries dos databases `testdb_*` dos testes de componente. Cada pool pgx se identifica pelo `application_name`, que é o `OTEL_SERVICE_NAME` do processo, e uma query própria do exporter (`backend/config/alloy/postgres-queries.yaml`) conta as conexões por database, role, `application_name` e estado em `pg_client_connections_count`, então as conexões de cada processo aparecem separadas. O dashboard `postgres` mostra conexões, locks, transações, tamanho e cache hit por database, as queries que mais ocupam o banco pelo `pg_stat_statements` e, na seção Cardinalidade, o top N de métricas por número de séries de cada `job`.

No diretório do serviço (`backend/services/<name>/`), com o `postgres` no ar:

```sh
make db.setup      # cria o role e o database, só se não existirem
make db.migrate    # aplica as migrations pendentes (binário cmd/migrate, mesmo DATABASE_URL da API)
make db.rollback   # desfaz só a última migration aplicada (a seção Down dela)
make db.reset      # apaga o database, recria e migra de novo
```

As migrations ficam em `internal/repository/migrations/` de cada serviço, embutidas no binário (`migrations.FS`). O runner do `cmd/migrate` (`migrate.Run`) e o `testdb` dos testes de componente vêm do módulo [`postgres` do go-kit](https://github.com/lcnascimento/go-kit/tree/main/postgres), parametrizados por esse `fs.FS`; o serviço não tem cópia deles. Cada uma tem a seção `-- +goose Up` e a `-- +goose Down`, que a desfaz; um teste de componente desfaz e reaplica todas, então uma Down quebrada falha no `make test`. São numeradas em sequência com 5 dígitos, cada uma roda na sua transação e todas rodam sob advisory lock, então execuções concorrentes não se atropelam. A exceção é a migration que começa com `-- +goose NO TRANSACTION`, para um comando que o Postgres não aceita numa transação, como o `CREATE INDEX CONCURRENTLY`; o `make lint` roda o squawk nela sem assumir transação. Uma coluna obrigatória numa tabela existente entra em duas migrations, sem `squawk-ignore` na Up: a primeira a adiciona nullable com um `CHECK (... IS NOT NULL) NOT VALID`, e a seguinte faz o `VALIDATE CONSTRAINT` e o `SET NOT NULL`. A API não roda migrations no boot. O `cmd/migrate` loga cada migration aplicada ou desfeita pelo logger do o11y, como serviço `notes-migrate` no Grafana. Uma Down que apaga dado (`DROP TABLE`, `DROP COLUMN`) precisa de um `-- squawk-ignore <regra>` explícito antes do comando, para o `make lint` passar. Para criar a próxima, em `backend/`:

```sh
make migration.new name=add_tags                  # basta enquanto só um serviço tem migrations
make migration.new name=add_tags service=<name>   # com mais de um
```

A API abre o pool no boot e pinga o banco: com o banco inacessível, ela falha em até 5s com um erro dizendo o host e a causa. As queries aparecem nos traces do Grafana.

Cada repositório escreve suas queries em SQL num `queries.sql` ao lado dele, e o [sqlc](https://sqlc.dev/) gera o código Go tipado num pacote `internal/db` do repositório (commitado). O `sqlc.yaml` na raiz do serviço tem um bloco `sql:` por repositório, com o `schema` apontando para as migrations. Depois de mudar uma query ou uma migration, rode `make sqlc` em `backend/`; o `quality_gate / format` falha se o código gerado estiver desatualizado.

Se o banco tiver uma migration aplicada que a branch atual não conhece (por exemplo, depois de trocar de branch), o `<name>-migrate` e o `make db.migrate` falham com `UNKNOWN_APPLIED_MIGRATION`, pedindo para zerar o banco: rode `make db.reset`. Para zerar todos os bancos, apague o volume: `docker compose down -v` em `backend/`.

### Autenticação e autorização

O `iam` é parte fixa da stack, e não do exemplo: o `make clean-example` o mantém inteiro. Ele tem três partes:

- o database `iam` e o serviço `backend/services/iam`, que só tem as migrations (`cmd/migrate` e os `make db.*`) e não tem API. No Tilt, a cadeia é `postgres → iam-db → iam-migrate → iam-seed → web`, e em runtime só o app web acessa esse database;
- o [Better Auth](https://www.better-auth.com/) dentro do app web (`frontend/apps/web/shared/auth/`): cadastro e login com email e senha, sessão deslizante de 7 dias num cookie HttpOnly, Secure e SameSite=Strict, rate limit de login e Organizations com as Roles `owner`, `admin` e `viewer`. As telas são o `(auth)` (`/sign-in` e `/sign-up`), o seletor de Organization no topo da barra lateral, `/settings/members` e a tela "sem Organization". O `proxy.ts` manda quem não tem sessão para `/sign-in?callbackUrl=...`;
- o módulo Go `backend/services/iam/pkg` (`domain`, `roles`, `actor` e `actor/actortest`), que os serviços Go usam para validar o token de Actor e checar as Permissions.

O cadastro em `/sign-up` cria o User e uma Organization nova, com o User como Owner. O endpoint público `POST /api/auth/sign-up/email` exige o `organizationName` e responde `400` (`INVALID_ORGANIZATION_NAME`) sem ele; só um User criado fora de qualquer endpoint, como os do seed, nasce sem Organization. Não há criação de Organization pela UI além dessa. Owner e Admin adicionam Users já cadastrados como Members em `/settings/members`, mudam a Role deles e os removem, e qualquer Member sai da Organization por lá.

**Users do seed.** O recurso `iam-seed` do Tilt (`pnpm --filter @boilerplate/web iam:seed`, idempotente) cria os Users direto pelo adapter do Better Auth, sem passar pelo cadastro, e as Organizations Acme e Globex com eles, todos com a senha de dev `dev-password` (`SEED_PASSWORD` em `frontend/apps/web/scripts/iam-seed/users.ts`):

| User | Organization e Role |
| --- | --- |
| `owner@acme.test` | Owner da Acme |
| `admin@acme.test` | Admin da Acme |
| `viewer@acme.test` | Viewer da Acme |
| `owner@globex.test` | Owner da Globex |
| `admin@multi.test` | Admin da Acme e da Globex, para trocar de Organization |

**Rate limit.** O Better Auth só liga o rate limit com `NODE_ENV=production`, então no `next dev` do Tilt e no e2e as tentativas de login não são limitadas (a história de força bruta não é exercitada localmente). O teste de unidade `login-failures.test.ts` liga o rate limit explicitamente para cobrir a métrica `rate_limited`.

O e2e entra com cada um deles num setup project (`e2e/auth.setup.ts`) e guarda um `storageState` por User em `frontend/apps/web/e2e/.auth/` (gitignored); o Owner da Acme é o padrão, e um spec troca de User com `test.use({ storageState: storageStateOf("<email>") })`.

**Variáveis.** O app web valida estas no boot (`parseIamEnv`, chamado no `instrumentation.ts`) e recusa subir sem elas. O Tilt as passa ao `web` e ao `iam-seed` (`iam_env` em `frontend/Tiltfile`), e o `.env.example` do app traz os mesmos valores para o `.env.local`:

| Variável | O que é |
| --- | --- |
| `IAM_DATABASE_URL` | Conexão com o database `iam` (local: `postgres://iam:iam@localhost:5432/iam?sslmode=disable`) |
| `BETTER_AUTH_SECRET` | Segredo que assina sessões e cifra as chaves do JWKS. O valor do repositório é só para dev: com `NODE_ENV=production`, o `parseIamEnv` o recusa; gere outro (`openssl rand -base64 32`) |
| `BETTER_AUTH_URL` | URL base do app web (local: `http://localhost:3000`). É também o `iss` do token de Actor, então o `AUTH_ISSUER` dos serviços Go precisa ser igual a ela |

**Token de Actor.** Cada chamada do servidor a um serviço Go leva `Authorization: Bearer` com um JWT de Actor de vida curta (5 minutos), emitido por `mintActorToken` (`frontend/apps/web/shared/auth/actor-token.ts`, o único módulo de emissão, com o guard `server-only`). Ele assina o Actor que recebe, `{ sub: "user_<uuid>", org: "org_<uuid>", role }` (EdDSA, `aud` = `boilerplate-api`), com o plugin `jwt` do Better Auth, sem reler o `member`. Quem confere o `member` é o Actor da requisição, `getRequestActor` (`frontend/apps/web/shared/auth/request-actor.ts`): ele lê a sessão, carrega numa só query os Members do User com a Role, do mais antigo ao mais novo, mantém a Organization ativa enquanto o User for Member dela (senão, passa para a mais antiga e grava isso na sessão) e falha com Role fora de `owner | admin | viewer`. O `iam:token` e o fixture do e2e, que não têm requisição, buscam o Actor com `findActor` (`shared/auth/actor.ts`), que recusa um User sem Member na Organization. O token nunca chega ao browser. O cliente de `@boilerplate/api` recebe um `getToken`, e as server actions emitem um token por chamada, sem cache, para a Organization ativa; só a resolução do Actor é reaproveitada dentro da mesma requisição. No Go, o middleware `actor.NewMiddleware` de `iam/pkg/actor` (com `Start` e `Stop` nos hooks do fx, que buscam o JWKS e encerram a atualização dele) valida o token contra o JWKS do web (`AUTH_JWKS_URL`, `AUTH_ISSUER` e `AUTH_AUDIENCE` no `.env.example` do serviço), responde `401` sem ele e põe o Actor no contexto; as rotas de health continuam públicas. Um Member removido ou com a Role trocada perde o acesso no próximo token, em até 5 minutos.

**Token para o Bruno.** Para chamar um serviço Go pela coleção Bruno, gere o token de um User do seed:

```sh
pnpm -C frontend --filter @boilerplate/web iam:token --as owner@acme.test [--org <slug>] [--ttl 1h]
```

Ele grava `IAM_TOKEN` no `.env` (gitignored) de cada coleção em `backend/config/bruno/collections/`, que a raiz da coleção usa em `request.auth` (`type: bearer`, `token: "{{process.env.IAM_TOKEN}}"`). Sem `--org`, vale a Organization mais antiga do User; sem `--ttl`, 5 minutos. Sem nenhuma coleção, ele imprime o token. O Bruno não tem auth no nível do workspace, então o bearer fica na raiz de cada coleção, e um teste do app web (`scripts/iam-token/bruno.test.ts`, no `pnpm test`) falha se alguma coleção em `backend/config/bruno/collections/` não o declarar. Depois do `make clean-example` não sobra coleção: a primeira que um serviço novo criar já nasce com essa regra.

**Permissions.** A matriz Role → Permissions é declarada uma vez em Go, em `backend/services/iam/pkg/roles/matrix.go`, e o `make iam.generate` gera dela o `frontend/apps/web/shared/auth/roles.gen.ts` (o `statement`, o `ac` e as `roles` do Better Auth), junto com as Permissions de membership do Better Auth (`ownerAc` para Owner, `adminAc` para Admin, nada para Viewer). A UI esconde o que a Role não permite com `roles[role].authorize(...)`, mas quem decide é o serviço Go: a application chama `actor.Can(permission)` e responde `403` quando a Role não a concede.

<!-- example:begin -->

No exemplo, as Notes pertencem a uma Organization: cada Member só vê as da Organization ativa (filtro no repositório e row-level security no database `notes`), o Owner lê, cria e apaga, o Admin lê e cria, e o Viewer só lê (`note:read`, `note:create` e `note:delete` na matriz).

<!-- example:end -->

A telemetria de identidade também é núcleo: o middleware e as server actions gravam `user.id` e `organization.id` nos spans e logs (nunca email nem nome), o Go conta `iam.actor.authentication.failures` por `reason` e `iam.actor.authorization.denials` por `permission` e `role`, e o web conta `iam.login.failures` por `reason`. Um processo declarado com `WithActorAuth()` no `o11ygen` ganha os painéis e o alerta de autenticação (ver Mensageria, acima).

### Lint e testes

**Tudo** (na raiz; cada job do CI roda o alvo de mesmo nome, então uma falha se reproduz com o mesmo comando):

```sh
make format          # formata backend e frontend e regenera o código gerado (sqlc, mocks, o11y, proto); commite o que mudar
make format.check    # com a árvore limpa: roda o make format e falha se ele mudar algum arquivo
make lint            # os linters, o typecheck e o build, no backend, no frontend e no OpenAPI
make lint.breaking   # buf breaking dos .proto contra AGAINST (padrão: origin/main); no CI, contra a base do PR
make test            # sobe o Postgres e roda os testes de unidade e de componente do backend e do frontend
make stack.up        # sobe o stack completo do Tilt em background (log em TILT_LOG) e espera os recursos que iniciam sozinhos
make test.e2e        # com o stack de pé: o e2e do Playwright e depois o make o11y.check-live
make stack.down      # tilt down
```

O job `format` é o `make format.check`, e o job `e2e` é `make stack.up && make test.e2e; make stack.down`.

**Backend** (em `backend/`, cada alvo roda em todos os módulos do `go.work`):

```sh
make format           # golangci-lint fmt (gci e gofmt)
make lint             # golangci-lint (make lint.fix corrige o que der) e squawk nas migrations
make test             # go test -race com cobertura: unit + componente (sobe o Postgres; exige o Docker; timeout de 3m)
make test.unit        # só os de unidade (-short), sem banco nem Docker (timeout de 30s)
make test.component   # só os de componente (-run Component; sobe o Postgres; timeout de 3m)
make sqlc             # regenera e confere (sqlc vet) o código das queries de cada serviço
make o11y.generate    # regenera os dashboards, as regras do Prometheus, as do ruler do Loki e as specs e regras de SLO (sloth generate) a partir do o11ygen (commitados; o quality_gate / format falha se estiverem desatualizados)
make iam.generate     # regenera o frontend/apps/web/shared/auth/roles.gen.ts (statement, ac e roles do Better Auth) a partir da matriz de Permissions em backend/services/iam/pkg/roles (commitado; o quality_gate / format falha se estiver desatualizado)
make o11y.lint        # dashboard-linter --strict nos dashboards; promtool check/test rules e pint nas regras; lokitool rules lint/check nas regras do Loki; sloth validate, promtool e pint nos SLOs (ferramentas instaladas pelo make setup)
make o11y.check-live  # confere o gerado contra o stack completo no ar (tilt up, não o minimal): métricas do catálogo, teto de séries, regras, dashboards e datasources
make proto.lint       # buf lint (STANDARD) em cada módulo buf (diretório com buf.yaml)
make proto.generate   # regenera o Go dos .proto (commitado; o quality_gate / format falha se estiver desatualizado)
make proto.breaking   # buf breaking (FILE) contra AGAINST (padrão: origin/main)
make mocks            # regenera internal/mocks/mocks.go a partir de cada contracts.go
make tidy             # go mod tidy
make vuln-check       # govulncheck (só local, fora do CI)
```

Os testes de componente (`TestComponent<Unit>_<Scenario>`, pulados com `-short`) rodam contra o Postgres de verdade, sem precisar do `<name>-db` nem do `<name>-migrate`: basta o container (`tilt up` ou `docker compose up -d postgres` em `backend/`). Cada teste ganha um database próprio, clonado de um template já migrado pelo [pgtestdb](https://github.com/peterldowns/pgtestdb), então eles rodam em paralelo e nunca tocam o database de dev. Eles conectam com um role sem superuser (o `pgtdbuser` do pgtestdb), porque superuser ignora row-level security mesmo com `FORCE`. O template é reaproveitado entre execuções e só é recriado quando as migrations mudam; o database de um teste que falha fica no servidor, com o nome no log do teste. A conexão de admin vem de `POSTGRES_ADMIN_URL` (padrão: o superuser do compose, `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`). Com o Postgres fora do ar, eles falham (não pulam) com uma mensagem dizendo o que rodar.

<!-- example:begin -->

Os testes de componente de mensageria usam o helper público `eventstest` (em `notes/pkg/events/eventstest`): um container Redpanda (broker e Schema Registry) por pacote de teste, via [testcontainers](https://golang.testcontainers.org/), com os schemas já registrados, e tópico e consumer group `test_<uuid>` próprios de cada teste. O `Publish` dele produz no envelope CloudEvents e no wire format Confluent reais, então um consumidor é testado sem o forwarder. Eles só exigem o Docker; o pull e o start do container cabem no timeout de 3 minutos.

<!-- example:end -->

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

No CI, o job `quality_gate / e2e` faz o mesmo caminho com os alvos da raiz, e você o reproduz localmente com `make stack.up && make test.e2e; make stack.down`. O `make stack.up` sobe o stack completo (`tilt up`, com observabilidade e telemetria de verdade; o `minimal` é só para desenvolvimento) em background e só retorna quando ficam prontos todos os recursos que iniciam sozinhos (os manuais, como os `test-*` do grupo `testing`, ficam de fora); o `make test.e2e` roda o `pnpm test:e2e` e depois o `make o11y.check-live`; o job sempre termina com `make stack.down`. Em falha, os logs do Tilt vão para o log do passo e para o artefato `tilt-logs`, e o relatório do Playwright vai para o artefato `playwright-report`.

O `make o11y.check-live` é um teste Go do `o11ygen` (`backend/tools/o11ygen/live_test.go`), ligado por `O11Y_LIVE_PROMETHEUS_URL` e pulado sem ela; o alvo do Make aponta para as portas do Tilt (`O11Y_LIVE_PROMETHEUS_URL`, `O11Y_LIVE_LOKI_URL` e `O11Y_LIVE_GRAFANA_URL`, com o usuário `admin` na URL do Grafana). Ele repete as consultas por até 4 minutos: um processo recém-iniciado leva até uns 3 minutos para chegar a uma recording rule (o primeiro export, a próxima avaliação e o `rule_query_offset`), e falha se faltar:

- uma métrica `PresenceAlways` do catálogo: as do stack sempre, e as de um serviço para cada processo que a emite (o `EmittedBy` da métrica, em `internal/catalog/metrics.go`);
- os logs do Faro no Loki de cada processo com `browser` (o evento de page load e as medidas de Web Vitals, em `internal/catalog/faro.go`);
- o `job:series:count` de cada processo abaixo do teto do serviço;
- um grupo gerado no `/api/v1/rules` do Prometheus ou do ruler do Loki, ou uma regra dele sem `health: ok`;
- um dashboard gerado (pelo UID) no `/api/search` do Grafana, ou um datasource provisionado sem health OK.

O que conferir vem do catálogo e dos arquivos gerados, então uma métrica, um dashboard ou um serviço novo entram na checagem sem mexer no teste. Uma métrica que vem do SDK de um processo (`OTLP`) precisa declarar o `EmittedBy`, e um tipo de artefato novo precisa dizer quem o carrega (`LoadedBy`): o `make test` falha sem isso.

Para depurar o e2e, o recurso `playwright` do Tilt abre o [UI Mode](https://playwright.dev/docs/test-ui-mode) do Playwright em <http://127.0.0.1:9330>, contra o mesmo stack.

**OpenAPI** (na raiz; o `make lint` já roda):

```sh
docker run --rm -v "$PWD:/work:ro" -w /work dshanley/vacuum:v0.33.0 lint --details openapi/*.yaml
```

O pre-commit (lefthook) roda `make lint` quando o commit tem `.go` e `pnpm lint` quando tem TS/JS. O commit-msg exige [Conventional Commits](https://www.conventionalcommits.org/).

O pre-push roda, na raiz, `make format.check` e depois `make lint`, os mesmos alvos dos jobs `format` e `lint` da CI. O `make format.check` recusa uma árvore suja (lista os arquivos; faça commit ou `git stash -u`), roda o `make format` e falha se ele mudar algum arquivo, imprimindo o diff e os arquivos novos: assim um código gerado desatualizado (sqlc, mocks, o11y, proto) ou um arquivo fora do formato não chega ao push. Ele precisa do setup da primeira vez (`pnpm install`, `pnpm -C frontend install` e `make -C backend setup`) e do Docker rodando. Para pular o hook num push, use `LEFTHOOK=0 git push` ou `git push --no-verify`.

### Explorando a API e depurando

- **Bruno**: abra `backend/config/bruno` no Bruno, ou rode uma coleção pela CLI a partir da pasta dela (`backend/config/bruno/collections/<Name>`): `npx @usebruno/cli run --env local`. As coleções autenticam com `{{process.env.IAM_TOKEN}}`, que o `iam:token` grava (ver "Token para o Bruno" em [Autenticação e autorização](#autenticação-e-autorização)).
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

1. Crie `backend/services/<name>/` com dois módulos Go: `go.mod` (`github.com/lcnascimento/boilerplate/backend/services/<name>`) e `pkg/go.mod` (`.../<name>/pkg`, com o `replace` para ele no primeiro). O domínio vive em `pkg/domain`, `pkg/models`, `internal/application`, `internal/repository` e `internal/transport`, e cada construtor é registrado no fx uma vez em `bundle/`. O `Makefile` do serviço define o `OTEL_SERVICE_NAME`, o binário (`build.api.frontend` / `run.api.frontend`) e o `FRONTEND_PORT`; o `Tiltfile` dele, os recursos do Tilt. O banco vem junto: `db.setup`/`db.migrate`/`db.rollback`/`db.reset` no `Makefile` (database e role com o nome do serviço), os recursos `<name>-db` e `<name>-migrate` no `Tiltfile`, as migrations em `internal/repository/migrations/` com o `cmd/migrate`, o `sqlc.yaml` com as `queries.sql` dos repositórios, e o `DATABASE_URL` no `.env.example`. A identidade também: `AUTH_JWKS_URL`, `AUTH_ISSUER` e `AUTH_AUDIENCE` no `.env.example`, o `actor.NewMiddleware` no subrouter das rotas de User (fora dele, só as de health), o Actor passado explicitamente do handler ao service, e as Permissions do serviço declaradas em `backend/services/iam/pkg/roles/matrix.go`, seguidas de `make iam.generate`. Os testes fabricam tokens e Actors com `actortest`, sem subir o web.
   <!-- example:begin -->
   O caminho mais curto é copiar `backend/services/notes/` e trocar `notes` pelo novo nome nesses lugares e nos imports.
   <!-- example:end -->
2. Escolha uma porta livre (ex.: `8082`) e use-a no `.env.example` do serviço (`FRONTEND_PORT`), no `FRONTEND_PORT ?=` do `Makefile` e na `readiness_probe` e no link do `Tiltfile` do serviço.
   <!-- example:begin -->
   A `8080` é do `notes`, e a `8091`, do health do forwarder dele.
   <!-- example:end -->
3. Registre os dois módulos no `backend/go.work`:

   ```text
   use (
   	./services/<name>
   	./services/<name>/pkg
   )
   ```

4. Inclua o serviço no `backend/Tiltfile`: `include('./services/<name>/Tiltfile')`.
5. Declare o serviço no gerador de observabilidade, em `backend/tools/o11ygen/services.go`: o nome e o `service.name` de cada processo, com `service.Daemon` e os componentes que ele roda (`service.API`, `service.Consumer`, `service.Worker`, `service.Browser`) ou `service.OneShot` para um processo que termina (como o `<name>-migrate`). Marque com `.WithDBPool()` o processo que abre um pool pgx, com `.WithActorAuth()` o processo cuja API autentica as requisições com o middleware `actor.NewMiddleware` e, num `consumer`, declare o consumer group com `.WithConsumerGroup("<group>")`: os alertas herdados dependem disso. Um `worker` é o forwarder do outbox e emite `<name>.outbox.oldest_pending.age`. Toda métrica que um artefato gerado consulta precisa estar no catálogo, em `backend/tools/o11ygen/internal/catalog/metrics.go`, com os labels permitidos; o `make test` falha com uma métrica ou um label fora dele. Depois rode `make o11y.generate tidy mocks sqlc lint test` em `backend/`, com o `postgres` no ar. Um teste com `fx.ValidateApp` em `bundle/` quebra se o grafo do fx estiver incompleto.
6. (Opcional) Crie a coleção Bruno em `backend/config/bruno/collections/<Name>` e liste-a em `collections:` no `workspace.yml`. Na raiz da coleção (`opencollection.yml`), declare `request.auth` com `type: bearer` e `token: "{{process.env.IAM_TOKEN}}"`, e use `auth: inherit` nos requests.

**OpenAPI**

7. Crie `openapi/<name>.yaml` (OpenAPI 3.1, autocontido) e monte-o no Scalar com uma linha em `openapi/docker-compose.yml`: `- ./<name>.yaml:/sources/<name>.yaml:ro`. Declare `components.securitySchemes.bearerAuth` (`type: http`, `scheme: bearer`, `bearerFormat: JWT`) com as claims do token, `security` global com `bearerAuth`, `security: []` em `/health` e `/ping`, e as respostas `401` e `403`.

**Frontend**

8. Crie `frontend/packages/api/src/<name>/index.ts`. O `exports` com curinga do pacote já publica o subpath `@boilerplate/api/<name>`, sem editar `package.json`. Ele traz os schemas Zod em `schemas/index.ts` (o entry isomórfico que client components importam), o cliente sobre `request()` de `@boilerplate/api/core`, que recebe um `getToken: GetToken` e passa o `token` de cada chamada ao `request()`, e um `parse<Name>Env` que valida `<NAME>_API_URL`. A action do app cria o cliente com um `getToken` que lê o `getRequestActor()` e passa o Actor ao `mintActorToken`, como o `requestActorToken` de `actions/notes.ts`.
   <!-- example:begin -->
   Siga `src/notes/`.
   <!-- example:end -->
9. Declare `<NAME>_API_URL`:
   - em `frontend/Tiltfile`, no `serve_env` do recurso `web`;
   - em `frontend/apps/web/.env.example`;
   - em `frontend/apps/web/instrumentation.ts`, chamando o `parse<Name>Env` dentro do `Promise.all`, para o servidor falhar na subida sem ela.

   O `turbo.json` já repassa `*_API_URL` para as tasks.

10. Telas: as rotas em `frontend/apps/web/app/(dashboard)/<name>/`, as funções `"use server"` em `frontend/apps/web/actions/<name>.ts`, as mensagens em `frontend/apps/web/shared/_messages/<name>.json` (pt-BR e en no mesmo arquivo) registradas em `shared/_messages/index.ts`, e o item do menu em `navRoutes` (`shared/navigation.ts`).
11. e2e: em `frontend/apps/web/e2e/stack.ts`, declare a `<NAME>_URL` (padrão, a porta do serviço) e acrescente o `/health` dela em `stackProbes`; se o e2e criar dados no serviço, acrescente em `e2e/fixtures.ts` uma limpeza em `cleanups` que apague pela API os registros com os marcadores. Acrescente o recurso do serviço às `resource_deps` do `playwright` (em `frontend/Tiltfile`) e do `test-frontend-e2e` (no `Tiltfile` da raiz). O CI não muda: o `make stack.up` sobe o stack do Tilt sem citar recurso nenhum.

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
   - O script exige a árvore git limpa e o pnpm/Node instalados, e não faz commit. Depois de trocar o nome, ele se apaga junto com o alvo `init`, o workflow `template` e esta seção do README, roda o `pnpm install` (na raiz e em `frontend/`) e o `pnpm -C frontend format` (o nome novo muda o comprimento de algumas linhas, e o prettier quebra ou junta imports de novo), então a árvore já sai formatada.
   - Se algo der errado, `git checkout . && git clean -fd` volta ao estado anterior.
   - Os `.env` não são versionados, então um `backend/.env` criado antes do `make init` (por um `make -C backend setup`, por exemplo) continua com o nome do template. Rode o `make init` antes do setup ou apague o `backend/.env` e rode `make -C backend setup` de novo.

3. Revise o diff e siga os próximos passos que o script imprime: o setup do ai-foundation e o commit.

## Removendo o exemplo

Para começar sem os serviços de exemplo `notes` e `audit`:

```sh
make clean-example
```

- É opcional e independente do `make init`: rode antes, depois ou nunca.
- Mantém o `iam` inteiro: login, cadastro, troca de Organization e gestão de Members continuam funcionando, e o `make iam.generate`, que o script roda, deixa no `roles.gen.ts` só as Permissions de membership.
- Apaga os arquivos do exemplo (`backend/services/notes/`, `backend/services/audit/`, os testes das Permissions de Note em `iam/pkg/roles`, a coleção Bruno `Notes`, `openapi/notes.yaml`, as telas e a action do `notes`, `frontend/packages/api/src/notes/`, as mensagens `notes` e os cenários e2e dele) e todo trecho entre cercas `example:begin` / `example:end` nos arquivos compartilhados, este README incluído (cercas em comentário HTML). Falha se sobrar alguma cerca.
- Exige a árvore git limpa e o pnpm/Node instalados, e não faz commit. Depois de rodar com sucesso, se apaga junto com o alvo `clean-example` e esta seção do README. Para desfazer antes do commit: `git checkout . && git clean -fd`.
- Tirar as cercas pode deixar código fora do formato do prettier (um import que passa a caber numa linha, um array vazio). Por isso o script termina com `pnpm install` (na raiz e em `frontend/`) e `pnpm -C frontend format`; falta só revisar o diff e commitar.

O estado sem os serviços de exemplo também é válido e fica verde: o `go.work` fica com `quality`, `tools`, `services/iam` e `services/iam/pkg` (`make lint` e `make test` passam, com o squawk nas migrations do `iam`), o `tilt up` sobe o Postgres, a observabilidade, a cadeia do `iam` e o `web`, o `openapi/` mantém a infra sem subir o Scalar, o app fica com `/` (a home depois do login), `/sign-in`, `/sign-up`, `/settings/members`, `/profile/preferences` e o `not-found`, o e2e roda os testes do shell, de login, cadastro, Organizations e Members, e o `@boilerplate/api` fica só com o `core` (`request`, `Result` e `APIError`). Um serviço novo entra como diz [Como adicionar um serviço](#como-adicionar-um-serviço), já com o `actor.NewMiddleware` para autenticar as rotas.
