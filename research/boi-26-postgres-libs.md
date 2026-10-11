# Bibliotecas de acesso ao Postgres em Go

> Pesquisa do ticket [BOI-26](https://linear.app/lcnascimento-boilerplate/issue/BOI-26), filho do mapa [BOI-25](https://linear.app/lcnascimento-boilerplate/issue/BOI-25).
> Coletado em 2026-10-11. **Fatos e trade-offs, sem escolha**: a decisão é do ticket de grilling "Stack de persistência" (BOI-32).

## Pergunta

Quais são as opções maduras para acessar Postgres num serviço Go do boilerplate (pgx/v5 puro, pgx + sqlc, bun, sqlx, gorm) e como se comparam em instrumentação OpenTelemetry, type-safety/geração de código, pool + config por env + `uber/fx`, testabilidade, e o que um eventual módulo `go-kit/postgres` encapsularia?

## Resumo

- Todas as opções relevantes, exceto o driver próprio do bun (`pgdriver`), usam **pgx/v5** por baixo: direto (`pgxpool`) ou via adaptador `database/sql` (`pgx/v5/stdlib`). A diferença real está na camada acima do driver: SQL à mão, SQL gerado (sqlc), query builder/ORM (bun, gorm) ou extensão leve de `database/sql` (sqlx).
- Existem **dois caminhos de instrumentação OTel**: `otelpgx` (hook nativo `pgx.QueryTracer` no `pgxpool`) e `otelsql`-like (wrap do driver `database/sql`). bun e gorm têm plugins próprios que usam o segundo caminho por baixo para métricas de pool.
- Todos os instrumentadores pegam `otel.GetTracerProvider()`/`otel.GetMeterProvider()` globais por padrão, que é exatamente o que `go-kit/o11y.Start` configura. Encaixe com o `go-kit/o11y` é de configuração zero em todas as opções.
- O go-kit atual não tem nenhum módulo de banco nem dependência de `uber/fx`; seus módulos leem env em variáveis de pacote e expõem `New()`/`MustNew()`.

## Tabela comparativa

| | pgx/v5 puro (`pgxpool`) | pgx/v5 + sqlc | bun | sqlx | gorm |
|---|---|---|---|---|---|
| Versão atual | v5.11.0 (2026-09-07) | sqlc v1.31.1 (2026-04-22) + pgx v5.11.0 | v1.3.0 (2026-10-01) | v1.4.0 (2024-04-15) | v1.31.2 (2026-06-22), driver postgres v1.6.3 |
| Atividade do repo (último push) | 2026-10-11 | 2026-10-10 | 2026-10-03 | **2024-08-15** | 2026-09-14 |
| Licença | MIT | MIT | BSD-2 | MIT | MIT |
| Estilo | SQL à mão, API nativa pgx | SQL à mão em `.sql`, Go **gerado** | query builder + ORM leve (struct tags `bun:`) | `database/sql` + scan para struct / named params | ORM completo (associações, hooks, AutoMigrate) |
| Driver por baixo | pgx nativo | pgx nativo (`sql_package: pgx/v5`) ou `database/sql` | `pgdriver` próprio **ou** pgx via `stdlib` | qualquer `database/sql` (ex.: `pgx/v5/stdlib`) | pgx via `stdlib` |
| Type-safety | Parâmetros `any`; scan manual ou `pgx.CollectRows`/`RowToStructByName` (reflexão em runtime) | **Compile-time**: tipos e funções gerados do schema + queries; erro de SQL/coluna aparece no `sqlc generate` | Reflexão em runtime sobre structs; SQL montado por builder | Reflexão em runtime; SQL string | Reflexão em runtime; SQL gerado pelo ORM |
| Instrumentação OTel | `exaring/otelpgx` (spans + métricas de operação + `RecordStats` do pool) | idem (sqlc chama o `pgxpool`, então o tracer do pool cobre tudo) | `bun/extra/bunotel` (query hook; métricas de pool via `uptrace/opentelemetry-go-extra/otelsql`) | `XSAM/otelsql` (wrap do driver + `RegisterDBStatsMetrics`) | `gorm.io/plugin/opentelemetry` (spans; métricas de `sql.DBStats`) |
| Pool | `pgxpool` (nativo, `pool_max_conns` etc. na connstring) | `pgxpool` | `sql.DB` | `sql.DB` | `sql.DB` |
| Recursos Postgres (COPY, LISTEN/NOTIFY, batch) | Todos | Todos via pgx (`:copyfrom`, `:batchexec`) | Parcial (via driver) | Não pela API `database/sql` | Não pela API do ORM |
| Dependências extras de build | nenhuma | binário `sqlc` + `sqlc.yaml` + código gerado versionado | nenhuma | nenhuma | nenhuma |
| Migrations | fora do escopo da lib | sqlc **lê** arquivos de migration (golang-migrate, goose, tern, dbmate, atlas, sql-migrate) como schema | pacote `bun/migrate` próprio | fora do escopo | `AutoMigrate` (schema derivado das structs) |

Fontes da tabela: versões e datas do [Go module proxy](https://proxy.golang.org) (`/@latest` de cada módulo) e da API do GitHub (`repos/<owner>/<repo>`, campo `pushed_at`); demais linhas nas seções abaixo.

## 1. Instrumentação OpenTelemetry e encaixe com `go-kit/o11y`

### Como o `go-kit/o11y` configura OTel

- `o11y.Start(ctx)` inicializa log, trace, metric e profile, e chama `propagator.Setup()` (fonte: `go-kit/o11y/o11y.go`).
- Os providers são registrados **globalmente**: `otel.SetTracerProvider(tp)` em `o11y/internal/trace/trace.go:91`, `otel.SetMeterProvider(mp)` em `o11y/internal/metric/metric.go:73`, `otel.SetTextMapPropagator(...)` em `o11y/internal/propagator/propagator.go:27`.
- Toda configuração vem das variáveis `OTEL_*` padrão (`o11y/internal/config/config.go`, via `go-kit/env`).
- Os módulos existentes (`http`, `kafka`, `temporal`) obtêm tracer/meter com `otel.Tracer(pkg)` / `otel.Meter(pkg)` em variáveis de pacote (ex.: `kafka/telemetry.go:18`, `http/httpclient/telemetry.go:23-24`, `temporal/temporalclient/client.go:26`).
- No `notes`, `bundle.NewFrontendServer()` é avaliado **antes** de `o11y.MustStart` (`cmd/api/frontend/main.go` → `bundle.Run`), e `fx.New` executa os `Invoke` imediatamente ([fx `app.go:419`](https://pkg.go.dev/go.uber.org/fx#New): "New creates and initializes an App, immediately executing any functions registered via Invoke options"). Logo, construtores do fx rodam antes do `o11y.Start`. Isso funciona porque o provider global do OTel é um placeholder que **delega** ao provider real quando ele é registrado depois (`go.opentelemetry.io/otel@v1.45.0/internal/global/trace.go:36-38`: "All TracerProvider functionality is forwarded to a delegate once configured").

### `exaring/otelpgx` (pgx puro, pgx + sqlc)

- Uso: `cfg.ConnConfig.Tracer = otelpgx.NewTracer()` antes de `pgxpool.NewWithConfig`, e `otelpgx.RecordStats(pool)` para métricas do pool ([README v0.13.0](https://github.com/exaring/otelpgx/blob/v0.13.0/README.md)). O próprio README do pgx lista `otelpgx` como adaptador de tracer ([pgx README, "Adapters for 3rd Party Tracers"](https://github.com/jackc/pgx/blob/v5.11.0/README.md)).
- Defaults: `otel.GetTracerProvider()` e `otel.GetMeterProvider()` (`tracer.go:114-115`, `meter.go:48`), sobrescrevíveis com `WithTracerProvider`/`WithMeterProvider`.
- Cobre query, batch, copy, prepare, connect e acquire do pool (`TraceQueryStart` … `TraceAcquireEnd`, `tracer.go:282-700`).
- Spans seguem a semconv de DB: nome = operação de baixa cardinalidade (ex.: `SELECT`) por padrão; `WithFullSQLInSpanName`/`WithQuerySpanNamePrefix` mudam isso (`tracer.go:334-352`). Comentários iniciais (como o `-- name: X :one` do sqlc) são ignorados ao derivar o nome (`operation_name.go`, `skipLeadingComments`).
- Atributos: `db.query.text`, `db.operation.name`, detalhes de conexão (desligáveis: `WithDisableSQLStatementInAttributes`, `WithDisableConnectionDetailsInAttributes`); parâmetros só com `WithIncludeQueryParameters` (`options.go`).
- **Só cria span de query se já houver um span pai gravando** (`if !trace.SpanFromContext(ctx).IsRecording() { return ctx }`, `tracer.go:290`). As métricas são registradas mesmo assim.
- Métricas: `db.client.operation.duration` (histograma, semconv) e `db.client.operation.errors` (`tracer.go:171-191`); de pool via `RecordStats`: acquire count/duration, conexões idle/acquired/total/max, destroys por lifetime/idle etc. (`meter.go:81-93`).
- Maturidade: v0.x (v0.13.0, 2026-10-08), 228 estrelas, Apache-2.0.
- Para combinar com outro tracer (ex.: `tracelog` para logs de query), pgx tem `pgx/v5/multitracer` ("Package multitracer provides a Tracer that can combine several tracers into one").

### `XSAM/otelsql` (sqlx, ou qualquer `database/sql`)

- API: `otelsql.Open`, `OpenDB(connector)`, `Register`, `WrapDriver` e `RegisterDBStatsMetrics(db)` (`sql.go:39-131`, v0.44.0, 2026-09-06).
- Instrumenta no nível de `database/sql/driver`, então vê o que passa por `sql.DB` (sem visão de acquire do pool pgx, pois o pool é o do `database/sql`).

### `bunotel` (bun)

- `db.AddQueryHook(bunotel.NewQueryHook(bunotel.WithDBName("...")))` ([bun README](https://github.com/uptrace/bun/blob/v1.3.0/README.md)). Opções `WithTracerProvider`/`WithMeterProvider` (`extra/bunotel/option.go`); sem elas, usa globais.
- Métricas do pool via `otelsql.ReportDBStatsMetrics(db.DB, ...)` do pacote `github.com/uptrace/opentelemetry-go-extra/otelsql` (`extra/bunotel/otel.go:57-71`).

### `gorm.io/plugin/opentelemetry` (gorm)

- `tracing.NewPlugin(opts...)`; usa `otel.GetTracerProvider()` se nenhum provider for passado (`tracing/tracing.go:46-53`) e reporta `sql.DBStats` como métricas (`tracing.go:73`). Opções incluem `WithoutQueryVariables`, `WithoutMetrics` (`tracing/option.go`). v0.1.16, último release 2025-07-18.

## 2. Geração de código / type-safety (sqlc) vs SQL à mão

- **sqlc** gera Go a partir de schema + queries SQL. Para Go, `sql_package` aceita `pgx/v4`, `pgx/v5` ou `database/sql` (default) ([config reference](https://docs.sqlc.dev/en/latest/reference/config.html)).
- Opções relevantes: `emit_interface` (gera interface `Querier`), `emit_pointers_for_null_types` (ponteiros em vez de `sql.Null*`, disponível para pgx), `query_parameter_limit` (0 = sempre struct de parâmetros), `overrides` (`db_type`/`column` → `go_type`) (mesma fonte).
- O código gerado recebe uma interface `DBTX` satisfeita por `*pgx.Conn`, `*pgxpool.Pool` e `pgx.Tx`; `New(db DBTX)` e `WithTx(tx)` permitem usar as mesmas queries dentro de transação ([Using Go and pgx](https://docs.sqlc.dev/en/latest/guides/using-go-and-pgx.html)).
- sqlc lê diretórios de migration de atlas, dbmate, golang-migrate, goose, sql-migrate e tern como schema e **ignora as down migrations** ([Handling SQL migrations](https://docs.sqlc.dev/en/latest/howto/ddl.html)). Ou seja, a escolha da ferramenta de migration (outro ticket do mapa) é compatível com sqlc se for uma dessas.
- `sqlc vet` (regras de lint) e `sqlc verify` (checagem de mudanças de schema) existem como checagens de CI (config reference).
- **pgx puro sem sqlc**: helpers genéricos `pgx.CollectRows`, `CollectOneRow`, `CollectExactlyOneRow`, `RowToStructByName[T]`, `ForEachRow`, `BeginFunc` (`rows.go:427-689`, `tx.go:402`). Type-safety de colunas é verificada só em runtime (reflexão por nome de coluna).
- Ponto de contato com o domínio do `notes`: `domain.NoteID` embute `util.CanonicalID` (ID textual com prefixo `note`, baseado em UUID v7; `pkg/domain/value_objects.go`). Em qualquer opção, o repositório precisa traduzir entre `domain.Note` e a linha do banco; com sqlc isso é tradução entre struct gerada e domínio (ou `overrides` para tipos próprios), com ORMs é tradução entre struct com tags e domínio (o pacote `domain` não carrega tags de persistência hoje).

## 3. Pool, configuração por env e `uber/fx`

### Pool e configuração

- `pgxpool.ParseConfig(connString)` aceita URL ou `key=value` e parâmetros de pool na própria connstring: `pool_max_conns` (default `max(4, NumCPU)`), `pool_min_conns`, `pool_min_idle_conns`, `pool_max_conn_lifetime` (1h), `pool_max_conn_idle_time` (30min), `pool_health_check_period` (1min), `pool_max_conn_lifetime_jitter` (`pgxpool/pool.go:346-361`).
- `pgconn.ParseConfig` respeita as variáveis libpq `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`, `PGAPPNAME` etc. (`pgconn/config.go:268-282`). Ou seja, sem nenhum código de config, uma connstring vazia já lê o ambiente.
- `pgxpool.NewWithConfig` exige config criado por `ParseConfig` (panic caso contrário, `pool.go:221-226`). Conexões são criadas sob demanda; só há criação antecipada quando `MinConns`/`MinIdleConns` > 0 (`createIdleResources`, `pool.go:336`). `Pool.Ping(ctx)` e `Pool.Close()` existem para checagem e encerramento (`pool.go:470, 850`).
- Opções baseadas em `database/sql` (sqlx, gorm, bun) usam o pool do `sql.DB` (`SetMaxOpenConns`, `SetConnMaxLifetime` etc.), não o `pgxpool`.
- pgx recomenda a interface nativa quando "the application only targets PostgreSQL" e "no other libraries that require database/sql are in use"; a interface nativa é mais rápida e expõe `LISTEN/NOTIFY` e `COPY` ([pgx README, "Choosing Between the pgx and database/sql Interfaces"](https://github.com/jackc/pgx/blob/v5.11.0/README.md)).
- bun com pgx: `stdlib.OpenDB(*pgxConfig)` + `pgdialect.New()`; a doc do bun sugere `QueryExecModeSimpleProtocol` "because Bun does not benefit from using them [prepared statements]" ([bun PostgreSQL](https://bun.uptrace.dev/postgres/)).

### Convenção do go-kit para config

- Módulos leem env em variáveis de pacote com default: `httpserver` `PORT` (3000), `kafka` `KAFKA_BROKERS` (`localhost:9092`), `temporalclient` `TEMPORAL_HOST`/`TEMPORAL_NAMESPACE` (fontes: `http/httpserver/server.go:18`, `kafka/value_objects.go:8-9`, `temporal/temporalclient/client.go:37-38`).
- `go-kit/env` oferece `env.Get[T]` com `WithDefaultValue`/`WithEnum` e `env.Parse(struct)` (sobre `caarlos0/env/v10`, com `WithPrefix`) (`env/getter.go`, `env/parser.go`).

### Integração com `uber/fx`

- O go-kit **não** depende de `uber/fx` (nenhum arquivo do repo importa `go.uber.org/fx`); a fiação fx fica no `bundle/` de cada serviço (ex.: `bundle/infrastructure.go`, `bundle/runners.go` no `notes`, onde o servidor HTTP entra com `lifecycle.Append(fx.Hook{OnStart: server.Start, OnStop: server.Shutdown})`).
- Precedente no `remarks` (Mongo e Redis, não Postgres): `bundle/infrastructure.go` do `cdp` expõe `fx.Module("Mongo Client", fx.Provide(mongoClient))`, onde `mongoClient()` lê `MONGO_URI` via `go-kit/env` com default local, liga o monitor `otelmongo.NewMonitor()` e dá `panic` em erro; sem hook de `Disconnect` no fx.
- Para Postgres, o formato equivalente seria um `fx.Provide` que devolve `*pgxpool.Pool` (ou `*sql.DB`/`*bun.DB`/`*gorm.DB`) e um `fx.Lifecycle` com `OnStart: pool.Ping` / `OnStop: pool.Close`. Isso vale para todas as opções; só muda o tipo do handle.

## 4. Testabilidade (testes de componente contra Postgres real)

- Os testes da aplicação já usam mocks de `internal/application/contracts.go` (`NotesRepository`); o teste de componente do repositório é ortogonal à biblioteca: em todas as opções ele abre conexão real, aplica migrations e exercita `Create/Get/List/Delete`.
- Referência do mapa (`remarks`): testes `TestComponent*` nos repositórios do `cdp` (ex.: `internal/repository/profiles/repository_test.go`), conexão por `env.Get("MONGO_URI", env.WithDefaultValue(...))` apontando para a dependência local, e `t.Cleanup` com `client.Disconnect(context.WithoutCancel(t.Context()))`. O mesmo formato serve para Postgres com qualquer lib (trocar `MONGO_URI` por uma connstring/`PG*`).
- Diferenças entre opções que afetam testes:
  - **pgx + sqlc**: as queries são métodos gerados sobre `DBTX`; dá para rodar cada teste dentro de uma `pgx.Tx` com `Queries.WithTx(tx)` e `Rollback` no cleanup (isolamento por transação), além do isolamento por dados únicos.
  - **pgx puro**: mesma possibilidade via `pgx.Tx`, se o repositório aceitar uma interface comum a `*pgxpool.Pool` e `pgx.Tx`.
  - **sqlx/bun/gorm**: isolamento por transação via `*sql.Tx`/`bun.Tx`/`db.Begin()`; ORMs com `AutoMigrate` permitem criar schema sem migration, mas aí o teste não exercita as migrations reais.
- Alternativa a depender do Postgres do Tilt/compose: `testcontainers-go/modules/postgres` (v0.44.0) com `postgres.Run(ctx, img, ...)`, `WithInitScripts`, `ConnectionString(ctx, "sslmode=disable")` e `Snapshot()`/`Restore()` para estado limpo por teste (caveat: não usar o banco `postgres` como banco do container com snapshots) ([Testcontainers for Go, Postgres](https://golang.testcontainers.org/modules/postgres/)). Exige Docker no ambiente de teste; o mapa já assume `docker compose up -d <dep>` no CI.

## 5. O que um módulo `go-kit/postgres` encapsularia

### Formato dos módulos existentes do go-kit

- Um `go.mod` por módulo (`http`, `kafka`, `temporal`, `o11y`, `env`, …), com `replace` para irmãos e tags por módulo (ex.: `go-kit/http v1.4.7`, `go-kit/o11y v1.6.5` no `go.mod` do `notes`).
- Construtor `New(opts ...Option)` + `MustNew()` que lê env com defaults locais e devolve o tipo do SDK subjacente (`temporalclient.New` devolve `client.Client` do SDK Temporal) ou um wrapper fino (`kafka.NewProducer` devolve `*kafka.Producer` sobre `segmentio/kafka-go`).
- Telemetria em `telemetry.go`: `logger`, `tracer`, `meter` em variáveis de pacote e hooks `onStart`/`onStop`/`onPublishStart` (ex.: `kafka/telemetry.go`). Instrumentação de terceiros é plugada no construtor (`temporalclient` usa `go.temporal.io/sdk/contrib/opentelemetry` com o `meter` do pacote).
- Sem `uber/fx` e sem health check embutido.

### Escopo provável (fatos do que cada peça exigiria)

| Peça | Com pgx/pgx+sqlc | Com bun/sqlx/gorm |
|---|---|---|
| Config por env | `pgxpool.ParseConfig` já lê `PG*` e `pool_*`; o módulo poderia só ler uma var (ex.: `DATABASE_URL`) com default local, no estilo `TEMPORAL_HOST` | idem para a DSN; pool configurado via `sql.DB.Set*` |
| OTel | `cfg.ConnConfig.Tracer = otelpgx.NewTracer(...)` + `otelpgx.RecordStats(pool)` | `otelsql`/`bunotel`/plugin gorm |
| Handle devolvido | `*pgxpool.Pool` (satisfaz `DBTX` do sqlc) | `*sql.DB` / `*sqlx.DB` / `*bun.DB` / `*gorm.DB` |
| Health | `pool.Ping(ctx)` | `db.PingContext(ctx)` |
| Encerramento | `pool.Close()` | `db.Close()` |
| Dependências que o módulo puxaria | pgx/v5 + otelpgx | lib escolhida + seu plugin OTel |

- Com sqlc, o código gerado vive no serviço (gerado de suas queries); um módulo go-kit cobriria só a conexão, não as queries.
- Migrations (golang-migrate, goose, tern…) seriam uma peça separada, assunto de outro ticket do mapa; sqlc consegue ler o diretório de migrations de qualquer uma das ferramentas listadas na seção 2.

## Fontes

- Código local: `backend/services/notes` (este repo, `main` @ `e12b42f`); `~/github/lcnascimento/go-kit` (`o11y`, `env`, `http`, `kafka`, `temporal`); `~/github/lcnascimento/remarks/backend/services/cdp`.
- Código dos módulos baixado do Go proxy: `github.com/jackc/pgx/v5@v5.11.0`, `github.com/exaring/otelpgx@v0.13.0`, `github.com/XSAM/otelsql@v0.44.0`, `github.com/uptrace/bun@v1.3.0` (+ `extra/bunotel`), `gorm.io/driver/postgres@v1.6.3`, `gorm.io/plugin/opentelemetry@v0.1.16`, `github.com/jmoiron/sqlx@v1.4.0`, `go.opentelemetry.io/otel@v1.45.0`, `go.uber.org/fx@v1.24.0`.
- Versões: `https://proxy.golang.org/<module>/@latest`. Atividade/licença: `gh api repos/<owner>/<repo>`.
- Docs: [pgx README](https://github.com/jackc/pgx/blob/v5.11.0/README.md); [otelpgx README](https://github.com/exaring/otelpgx); [sqlc config](https://docs.sqlc.dev/en/latest/reference/config.html); [sqlc + pgx](https://docs.sqlc.dev/en/latest/guides/using-go-and-pgx.html); [sqlc + migrations](https://docs.sqlc.dev/en/latest/howto/ddl.html); [bun PostgreSQL](https://bun.uptrace.dev/postgres/); [bun README](https://github.com/uptrace/bun); [Testcontainers Postgres](https://golang.testcontainers.org/modules/postgres/); [fx.New](https://pkg.go.dev/go.uber.org/fx#New).
