# BOI-27 — Ferramentas de migrations em Go

> Pesquisa do mapa BOI-25 (Persistência em Postgres, migrations e e2e sobre o stack do Tilt).
> Só fatos e trade-offs; **a escolha fica com o ticket de grilling BOI-32 ("Stack de persistência")**.
> Data da coleta: 2026-10-11. Versões conferidas em `gh release list` de cada repositório.

## Candidatas

| Ferramenta | Versão mais recente | Licença | Bancos | Fonte |
|---|---|---|---|---|
| goose (`github.com/pressly/goose/v3`) | v3.28.0 (2026-09-02) | MIT | vários (Postgres, MySQL, SQLite, ClickHouse, …) | [repo](https://github.com/pressly/goose) |
| golang-migrate (`github.com/golang-migrate/migrate/v4`) | v4.20.1 (2026-09-09) | MIT | vários | [repo](https://github.com/golang-migrate/migrate) |
| Atlas (`ariga.io/atlas`) | v1.3.0 (2026-08-02) | open-core: Community Edition Apache-2.0; binário oficial sob Atlas MSA | vários | [community edition](https://atlasgo.io/community-edition), [features](https://atlasgo.io/features) |
| dbmate (`github.com/amacneil/dbmate/v2`) | v2.36.0 (2026-09-19) | MIT | Postgres, MySQL/MariaDB, SQLite, ClickHouse | [repo](https://github.com/amacneil/dbmate) |
| tern (`github.com/jackc/tern/v2`) | v2.4.3 (2026-08-23) | MIT | **só Postgres** (pgx) | [repo](https://github.com/jackc/tern) |

## Tabela comparativa

| Critério | goose | golang-migrate | Atlas | dbmate | tern |
|---|---|---|---|---|---|
| **Formato** | SQL versionado (um arquivo, `-- +goose Up` / `-- +goose Down`) **e** migrations em Go (`NewGoMigration` / `WithGoMigrations`) | SQL versionado, dois arquivos por versão (`N_name.up.sql` / `N_name.down.sql`); sem migrations em Go | Versionado (SQL gerado por `migrate diff` + `atlas.sum`) **ou** declarativo (schema desejado em HCL/SQL, Atlas calcula o diff). Declarativo (`schema plan`) não está na Community Edition | SQL versionado, um arquivo com `-- migrate:up` / `-- migrate:down`; gera `db/schema.sql` | SQL versionado (`001_x.sql`) dividido por `---- create above / drop below ----`; templates `text/template` + Sprig; "code packages" para funções/views |
| **Embutir no binário (`embed.FS`) / biblioteca** | Sim: `goose.NewProvider(dialect, *sql.DB, fs.FS, opts...)` aceita `embed.FS` | Sim: source driver `iofs` + `migrate.NewWithInstance`; driver `pgx/v5` disponível | Não como biblioteca pura: o Go SDK (`ariga.io/atlas/atlasexec`) **executa o binário `atlas`** (exige o CLI instalado); o Go SDK não faz parte da Community Edition | Sim: `dbmate.New(url)` + campo `db.FS` aceita `embed.FS`; usa `lib/pq` no driver Postgres | Sim: `migrate.NewMigrator(ctx, *pgx.Conn, versionTable)` + `LoadMigrations(fs.FS)`; exige `*pgx.Conn` (não `*sql.DB`) |
| **Down migrations** | Sim (seção Down; opcional) | Sim (arquivo `.down.sql`) | `migrate down` existe, mas está fora da Community Edition | Sim (`-- migrate:down`) | Sim (abaixo do separador; sem separador = irreversível) |
| **Locking contra execução concorrente** | **Desligado por padrão.** Opt-in: `WithSessionLocker(lock.NewPostgresSessionLocker())` (advisory lock) ou `WithLocker(lock.NewPostgresTableLocker())` (tabela `goose_lock`, usa o pool) | Advisory lock no Postgres (`pg_advisory_lock`) sempre | Advisory lock por padrão (`atlas_migrate_execute`, `--lock-timeout` 10s, `--skip-lock`) | Opt-in: `--migration-lock` (só Postgres, advisory lock) | Advisory lock sempre (`pg_advisory_lock`) |
| **Transação e falha parcial** | Cada migration SQL em transação por padrão; `-- +goose NO TRANSACTION` para `CREATE INDEX CONCURRENTLY` etc. Falha faz rollback, sem estado "sujo" | Grava `dirty=true` antes de rodar e `false` depois; falha deixa o banco **dirty** e exige `migrate force <v>` manual. Arquivo é enviado num único `Exec` (Postgres trata multi-statement como transação implícita) | `--tx-mode file` (padrão) / `all` / `none`; em `none` retoma do statement que falhou | Arquivo inteiro em transação ("succeeds or fails as a whole"); `transaction:false` por arquivo | Cada migration em transação; opt-out por migration |
| **Reexecução (idempotência)** | `up` sem pendências é no-op (tabela `goose_db_version`) | `up` sem pendências retorna `ErrNoChange` (o chamador trata como sucesso) | `migrate apply` sem pendências é no-op; `atlas.sum` detecta arquivo alterado | No-op; `--strict` falha se houver aplicação fora de ordem | No-op (tabela de versão) |
| **Fora de ordem** | `WithAllowOutofOrder` | Não | via `--exec-order` | Permitido por padrão; `--strict` proíbe | Não |
| **Lint / validação no CI** | `goose validate` (só sintaxe/anotações) | Nenhum | `migrate lint` com analisadores (destrutivo, incompatível, locks/rewrites) e `ariga/atlas-action` — **desde a v0.38 exige Atlas Pro** (`atlas login`); fora da Community Edition | Nenhum | Nenhum |
| **Convivência com sqlc** | Suportado; sqlc ignora a seção Down. Aviso: sqlc lê em ordem **lexicográfica**, então nomes precisam de zero à esquerda ou timestamp | Suportado; ignora `.down.sql`; mesmo aviso de ordem lexicográfica | Suportado | Suportado (ignora `migrate:down`); `schema.sql` também pode servir de schema | Suportado (ignora o que está abaixo do separador) |

Fontes da tabela: [goose README](https://github.com/pressly/goose), [goose Provider](https://pressly.github.io/goose/documentation/provider/), [goose CHANGELOG / pacote `lock`](https://github.com/pressly/goose/tree/main/lock), [golang-migrate README](https://github.com/golang-migrate/migrate), [golang-migrate postgres driver (`postgres.go`: `Lock`, `SetVersion`, `ErrDatabaseDirty`)](https://github.com/golang-migrate/migrate/tree/master/database/postgres), [Atlas getting started](https://atlasgo.io/getting-started), [Atlas apply](https://atlasgo.io/versioned/apply), [Atlas lint](https://atlasgo.io/versioned/lint), [Atlas Community Edition](https://atlasgo.io/community-edition), [atlas-go-sdk (arquivado → `ariga.io/atlas/atlasexec`)](https://github.com/ariga/atlas-go-sdk), [dbmate README](https://github.com/amacneil/dbmate) e [`pkg/driver/postgres`](https://github.com/amacneil/dbmate/tree/main/pkg/driver/postgres), [tern README](https://github.com/jackc/tern) e [`migrate/migrate.go`](https://github.com/jackc/tern/blob/master/migrate/migrate.go), [sqlc — Using migration tools](https://docs.sqlc.dev/en/latest/howto/ddl.html).

## Lint independente de ferramenta

Só o Atlas traz lint de mudanças destrutivas, e atrás do plano Pro. Alternativa agnóstica: **[squawk](https://github.com/sbdchd/squawk)** (Apache-2.0/MIT), linter de migrations Postgres sobre arquivos SQL puros (regras como `adding-not-nullable-field`, `require-concurrent-index-creation`, `ban-drop-column`), distribuído via npm, pip, Docker e binário, com GitHub Action. A documentação não menciona suporte às anotações de goose/dbmate/tern: as seções Down ficariam no mesmo arquivo e seriam lintadas também (no golang-migrate, que separa `.down.sql`, basta passar só os `.up.sql`). Este fato pesa no formato de arquivo se lint entrar no escopo.

## Formas de execução

| Forma | Como fica | Trade-offs |
|---|---|---|
| **No boot do serviço** | `fx.Invoke`/`OnStart` chama a biblioteca com `embed.FS` antes do HTTP subir | Um artefato só, sem passo extra no Tilt/CI. Com várias réplicas, exige lock (goose: opt-in). Migration lenta atrasa o readiness. Mistura permissão de DDL com o usuário da aplicação. Inviável com Atlas sem o binário na imagem. |
| **Comando separado no mesmo módulo** (`cmd/migrate`) | Outro `main` no serviço, mesmo `embed.FS` | Mantém migrations versionadas junto do serviço e reaproveita config/env. Vira um `local_resource` no Tilt com `resource_deps` do Postgres, e o `notes-api-frontend` depende dele. Mesmo binário no CI e no e2e. |
| **Só o CLI** (sem código Go) | `goose`/`migrate`/`dbmate`/`atlas`/`tern` instalado via `go install`/imagem Docker | Zero código, mas ferramenta extra no `make setup` e no CI. Exigência natural para o Atlas. |
| **Job dedicado** | Tilt `local_resource` (como o `notes-api-frontend` em `backend/services/notes/Tiltfile`) ou serviço one-shot no compose com `depends_on: condition: service_completed_successfully` | Separa o ciclo de vida. No compose, a imagem oficial do CLI (`migrate/migrate`, `ghcr.io/amacneil/dbmate`, `arigaio/atlas`) evita build próprio. |

Contexto do repo: o Tilt já sobe a infra por `docker_compose('./docker-compose.yml')` em `backend/Tiltfile` e o serviço `notes` como `local_resource` com probe em `/health`. O CI de backend (`.github/workflows/ci-backend.yml`) roda `make setup`, `make lint` e `make test` sem nenhum container hoje.

## O que um módulo `go-kit/migrations` agregaria

Formato atual do go-kit (`~/github/lcnascimento/go-kit`): módulos independentes no `go.work` (`env`, `errors`, `grpc`, `http`, `kafka`, `o11y`, `temporal`, `util`, `validator`), cada um com `go.mod` e tag própria (`http/v1.4.8`, `kafka/v1.2.4`, …). **Nenhum depende de `go.uber.org/fx` nem de banco**; a fiação fx fica no `bundle/` de cada serviço (ex.: `backend/services/notes/bundle/infrastructure.go` envolve `validator.New` em `fx.Module`). O `kafka` é o precedente mais próximo: um wrapper fino sobre `segmentio/kafka-go` com telemetria OTel e config por env (`go-kit/env`).

O que um `go-kit/migrations` poderia somar sobre a ferramenta:

- Config por env (`DATABASE_URL`, tabela, timeout) no padrão do `go-kit/env`.
- Defaults seguros já ligados: lock (no goose é opt-in), dialeto Postgres, `embed.FS` como entrada.
- Logs/traces OTel e erros do `go-kit/errors` por migration aplicada.
- Uma função `Up(ctx, db, fsys)` usável tanto no boot quanto em `cmd/migrate`.

Contrapontos:

- As bibliotecas candidatas (goose Provider, tern Migrator, golang-migrate) já têm uma API de 2–3 chamadas com `fs.FS`. O wrapper seria raso, com uma camada a mais entre o leitor e a ferramenta.
- O wrapper fixa a ferramenta e o driver (`*sql.DB` vs `*pgx.Conn`) e acopla a um eventual `go-kit/postgres` (pool). Isso pede uma decisão conjunta com a escolha de driver/pool (BOI-32), caso exista um módulo de conexão.
- Para Atlas, um módulo Go agrega pouco, já que a execução é via binário.
- Hoje só o `notes` consome. Um helper local no template (`bundle` ou `pkg/`) é a alternativa sem novo módulo versionado.

## Resumo dos trade-offs (sem escolha)

- **goose**: o mais flexível como biblioteca (SQL + Go, `embed.FS`, `*sql.DB`, sem estado dirty), mas o lock é **opt-in**.
- **golang-migrate**: o mais difundido, lock sempre ligado e arquivos up/down separados (bons para lint). Em contrapartida, tem estado **dirty** com `force` manual e não suporta migrations em Go.
- **Atlas**: o único com diff declarativo e lint de destrutivas, porém lint, `down`, declarativo e Go SDK ficam fora da Community Edition (Pro/MSA) e a execução depende do binário.
- **dbmate**: CLI simples, agnóstico de linguagem, gera `schema.sql`, com `wait` embutido. Lock é opt-in e o driver é `lib/pq`.
- **tern**: só Postgres, nativo pgx, lock sempre ligado e templates. A API exige `*pgx.Conn`.
- **sqlc** lê o schema de todos os cinco. Para goose e golang-migrate, os nomes precisam ordenar lexicograficamente.
- **Lint de destrutivas** sem custo: squawk, independente da ferramenta, mais simples com arquivos up separados.
