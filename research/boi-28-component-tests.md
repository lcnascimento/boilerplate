# BOI-28: Estratégias de teste de componente com Postgres

> Ticket de research do mapa [BOI-25](https://linear.app/lcnascimento-boilerplate/issue/BOI-25). Aqui só há fatos e trade-offs; a escolha fica com o ticket de grilling "Testes de componente do repositório" (BOI-35).

## Pergunta

Como testar um repositório Go contra um Postgres real, isolado dos dados de dev, rápido e paralelo (`t.Parallel()`), local e no CI?

## Resumo

- O padrão do `remarks` é **dependência compartilhada + isolamento lógico por teste**: `newRepository(t)` pula em `-short`, lê a URL de uma env com default na porta que o Tilt/compose publica, falha com "run `tilt up`" se não conectar, cria um *tenant* `test_<uuid>` e o apaga em `t.Cleanup`. No CI, `docker compose up -d mongo dragonfly`, um loop de espera e `make test`. O `remarks` **não tem nenhum teste de componente com Postgres**. O Postgres do compose dele serve ao Temporal.
- O isolamento por tenant funciona no `remarks` porque **toda query é escopada pelo tenant** (no Mongo, um banco por tenant). O repositório `notes` do boilerplate **não tem tenant**: `List(ctx)` devolve todas as notas. Dois dos testes atuais (`TestStartsEmpty`, `TestListReturnsNewestFirst`) afirmam o conteúdo exato da lista e quebram num banco compartilhado com ids únicos.
- Para Postgres, a unidade de isolamento que preserva `List` exato e permite `t.Parallel()` é **um banco (ou schema) por teste**. Medido localmente (Postgres 18.4): `CREATE DATABASE … TEMPLATE` leva ~14 ms de mediana; schema + tabela leva ~2 ms.
- Onde o Postgres vem (Tilt/compose, testcontainers-go, service container do Actions) é uma escolha **independente** da unidade de isolamento, com uma exceção: o `Snapshot/Restore` do testcontainers isola por reset serial, sem paralelismo.
- Hoje o `make test` do boilerplate não precisa de Docker nem do Tilt e não separa unit de component. O `ci-backend` não sobe nenhuma dependência e é chamado pelo `ci-template` também no caminho com zero serviços.

## 1. O padrão de referência: `remarks`

Fontes: `~/github/lcnascimento/remarks`, lidas no estado local em 2026-10-11.

### Helper `newRepository`

`backend/services/cdp/internal/repository/{alias,dedup,identity,profiles,schemas,warehouse}/repository_test.go`. Exemplo em `profiles/repository_test.go:720-757`:

```go
if testing.Short() { t.Skip("component test requires MongoDB") }
uri := env.Get("MONGO_URI", env.WithDefaultValue("mongodb://mongo:mongo@localhost:27017"))
// connect + Ping com timeout de 2s; falha: t.Fatalf("MongoDB unavailable at %s (run `tilt up`): %v", ...)
tenant := orgdomain.BrandSlug(fmt.Sprintf("test_profiles_%s", uuid.NewString()))
t.Cleanup(func() {
    ctx := context.WithoutCancel(t.Context())
    _ = client.Database(tenant.String()).Drop(ctx)
    _ = client.Disconnect(ctx)
})
repo := profiles.New(client, opts...)
repo.EnsureIndexes(t.Context(), tenant) // "migração" idempotente, por tenant
```

- Os testes se chamam `TestComponent<Repo>_<Cenário>`, são funções soltas (sem `testify/suite`) e cada um chama `t.Parallel()` (`profiles/repository_test.go:26-35`).
- **Mongo**: o tenant vira um banco inteiro (`client.Database(tenant)`), descartado com `Drop`. Na prática é "banco por teste".
- **Redis/Dragonfly** (`alias`, `dedup`, `identity`): o tenant é prefixo de chave (`{tenant}:cdp:alias:*`), e a limpeza faz `KEYS` + `DEL` (`alias/repository_test.go:257-278`). São dados com ids únicos num store compartilhado.
- A conexão vem de `env.Get` do go-kit, com default para a porta do compose/Tilt (`MONGO_URI`, `REDIS_URL`, `BIGQUERY_HTTP_ENDPOINT`).
- O "schema" é aplicado pelo próprio teste (`EnsureIndexes`) logo depois de criar o tenant. Não existe passo de migration separado.

### Makefiles

`backend/services/cdp/Makefile`:

```make
test: test.pkg test.unit test.component
	@go test -timeout 30s -cover ... $(go list ./... | grep -v /cmd | grep -v /mocks | grep -v /repository/warehouse)
test.unit: test.pkg
	@OTEL_SERVICE_NAME=cdp.test.unit go test -timeout 30s -short -cover ...
test.component:
	@OTEL_SERVICE_NAME=cdp.test.component go test -timeout 30s -run Component -cover ... (sem /repository/warehouse)
```

- `test` depende de `test.unit` e `test.component` e **depois roda tudo de novo** sem `-short`. Os testes de componente rodam duas vezes por `make test`.
- A exclusão do `warehouse` (emulador BigQuery instável) está documentada no Makefile e no CI, como manda a convenção da casa (`engineering:golang` → `references/testing.md`, "Every layer").
- `backend/Makefile` só itera `services/*` e delega `test`, `test.pkg`, `test.unit` e `test.component` a cada serviço.

### CI

`.github/workflows/ci-backend.yml` (somente `pull_request` em `backend/**`):

1. `docker compose -f backend/docker-compose.yml up -d mongo dragonfly`
2. Loops "Wait for MongoDB" / "Wait for Dragonfly" (30 tentativas × 2 s com `mongosh`/`redis-cli` via `docker compose exec`).
3. `make lint`, depois `make test`.

O mesmo `backend/docker-compose.yml` serve o Tilt local e o CI. No CI ele sobe só os serviços nomeados. O compose do remarks declara `postgresql` (`postgres:18`, usuário e senha `remarks`, porta 5432, volume nomeado), usado pelo Temporal.

### Convenção da casa (skill `engineering:golang`)

`references/testing.md`: o nível `component` cobre `repository` e `bridge` e roda com `go test -run Component` contra "the real thing". O teste se chama `TestComponent<Unit>_<Scenario>` e pula em `testing.Short()`. Ainda segundo a skill: "Each component test isolates its data (a random tenant or database name such as `test_schemas_<uuid>`) and removes it in `t.Cleanup` with `context.WithoutCancel(t.Context())`". A skill também manda usar `testify/suite` quando há setup compartilhado e rodar a suíte com `t.Parallel()`.

## 2. Estado atual do boilerplate

- `backend/Makefile`, alvo `test`: `go test -timeout 30s -race -cover -coverprofile=coverage.out $pkgs` em cada módulo do `go.work`. Não há `test.unit` nem `test.component` e não há `-short`.
- `backend/services/notes/Makefile` só tem `build.api.frontend` e `run.api.frontend`; os alvos de teste moram no `backend/Makefile`.
- `backend/services/notes/internal/repository/notes/repository_test.go` é uma `testify/suite` com `t.Parallel()` só em `TestRepository`. `SetupTest` cria um `notes.New()` novo (mapa em memória) por método. Os métodos são: `TestStartsEmpty`, `TestCreateThenGet`, `TestGetUnknownIDReturnsNotFound`, `TestStoredNoteIsNotAliasedByCaller`, `TestListReturnsNewestFirst`, `TestListBreaksTimestampTiesByID`, `TestDeleteRemovesNote`, `TestDeleteUnknownIDReturnsNotFound` e `TestConcurrentAccessIsSafe`.
- A interface do repositório (`repository.go`) é `Create`, `Get(id)`, `List(ctx)` (sem filtro) e `Delete(id)`. **Não há tenant nem nenhuma outra chave de partição.**
- `backend/docker-compose.yml` tem só observabilidade (alloy, prometheus, loki, tempo, pyroscope, grafana) e ainda não tem Postgres. `backend/Tiltfile` faz `docker_compose('./docker-compose.yml')` e inclui `services/notes/Tiltfile` dentro de `example:begin/end`.
- `.github/workflows/ci-backend.yml`: `checkout` → `Bootstrap` (input opcional) → `setup-go` → `make setup` → `make lint` → `make test`. Não sobe nenhuma dependência. Também é `workflow_call`.
- `.github/workflows/ci-template.yml` chama o `ci-backend` duas vezes: `make init` (com o `notes`) e `make clean-example && make init` (zero serviços; o `go.work` fica só com `use ./quality`).

### Restrição do `testify/suite`

O README do testify v1.12.1 (versão no `go.mod` do notes) e `suite/doc.go:8` dizem: "The suite package does not support parallel tests. See [#934](https://github.com/stretchr/testify/issues/934)". Com a suíte atual, os métodos rodam **em série dentro do pacote**. O `t.Parallel()` em `TestRepository` só paraleliza a suíte com outros `Test*` do pacote. Para ter `t.Parallel()` por cenário, como no `remarks`, os testes precisam ser funções `TestComponent…` soltas ou subtestes com `t.Run` + `t.Parallel()`.

## 3. De onde vem o Postgres

| | A. Postgres do Tilt/compose (padrão `remarks`) | B. testcontainers-go | C. Service container do GitHub Actions |
| --- | --- | --- | --- |
| Local | Exige `tilt up` (ou `docker compose up -d postgres`) antes do `make test.component`. Sem ele, `t.Fatalf("… run tilt up")` | Exige só um daemon Docker; o teste sobe o próprio container | Não se aplica; vale só no CI e precisa de outra fonte local (A ou B) |
| CI | `docker compose -f backend/docker-compose.yml up -d postgres` + espera (healthcheck ou loop `pg_isready`) | Nada no workflow; o `ubuntu-latest` já tem Docker | Bloco `services: postgres:` no job, com `--health-cmd pg_isready`. O runner espera o healthcheck antes dos passos |
| Mesmo caminho local/CI | Sim, o mesmo compose | Sim, o mesmo código Go | Não; o CI diverge do local |
| Dependência Go nova | Nenhuma (driver + `env`) | `github.com/testcontainers/testcontainers-go/modules/postgres` (+ Ryuk) | Nenhuma |
| Isolamento dos dados de dev | Mesmo servidor do dev; o isolamento vem da unidade de isolamento (seção 4) | Servidor descartável, isolado por construção | Servidor descartável no CI |
| Ciclo de vida | Persistente; sobrevive entre execuções (sobras de testes que falharam ficam lá) | Por processo de teste; o Ryuk remove containers órfãos | Por job |
| Caminho "zero serviços" do `ci-template` | O `up -d postgres` precisa existir no compose mesmo sem serviço, ou ser condicional | Nada sobe se não houver teste | O container sobe mesmo sem serviço (custo do pull) |

Fatos de suporte:

- **testcontainers-go, módulo postgres** ([docs](https://golang.testcontainers.org/modules/postgres/)): `postgres.Run(ctx, img, opts...)` com `WithDatabase/WithUsername/WithPassword`, `WithInitScripts` e `BasicWaitStrategies()` (a doc avisa que sem a wait strategy "the tests will be flaky" em macOS/Windows).
- **Snapshot/Restore** ([docs](https://golang.testcontainers.org/modules/postgres/); [fonte `postgres.go`](https://github.com/testcontainers/testcontainers-go/blob/main/modules/postgres/postgres.go)): `Snapshot` faz `CREATE DATABASE "migrated_template" WITH TEMPLATE "<db>"` + `ALTER DATABASE … is_template = TRUE`, e `Restore` derruba e recria o banco principal a partir dele. É **um único banco reiniciado entre testes**, ou seja, serial. A doc proíbe usar o banco `postgres` como banco do container quando se usa snapshot.
- **Ryuk** ([docs](https://golang.testcontainers.org/features/garbage_collector/)): "removes containers/networks/volumes created by Testcontainers for Go". Pode ser desligado com `TESTCONTAINERS_RYUK_DISABLED=true`, com aviso de "unexpected behavior".
- **Granularidade no B**: `go test` roda cada pacote num binário separado, com até `-p` = GOMAXPROCS binários em paralelo (`go help build`). Um container por pacote (via `TestMain` ou `sync.Once`) significa um container por pacote de repositório. Hoje o notes tem um pacote de repositório.
- **Service container** ([GitHub Docs](https://docs.github.com/en/actions/tutorials/use-containerized-services/create-postgresql-service-containers)): exige runner Linux. Com o job rodando direto no runner, publica-se `5432:5432` e o acesso é por `localhost`. O exemplo usa `--health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5`.

## 4. Unidade de isolamento

| Unidade | Como | `List` exato? | `t.Parallel()` | Custo medido por teste | Observações |
| --- | --- | --- | --- | --- | --- |
| **Banco por teste via TEMPLATE** | Migra um banco template uma vez; cada teste faz `CREATE DATABASE test_<uuid> TEMPLATE tmpl` e `DROP` no `t.Cleanup` | Sim | Sim | `CREATE … TEMPLATE` mediana 14,1 ms (máx. 32,6); `DROP` mediana 6,1 ms (um pico de 607 ms em 20) | Nenhuma outra sessão pode estar conectada ao template durante a cópia. `CREATE DATABASE` não roda em transação |
| **`pgtestdb`** (biblioteca da mesma técnica) | `pgtestdb.New(t, conf, migrator) *sql.DB` | Sim | Sim | README: "on the order of 10s of milliseconds"; template ~500 ms uma vez | Adapters prontos para goose, golang-migrate, atlas, dbmate, sql-migrate, pgmigrate, bun e tern |
| **Schema por teste** | `CREATE SCHEMA test_<uuid>`, migrations aplicadas nele, `search_path` na conexão, `DROP SCHEMA … CASCADE` | Sim | Sim | schema + tabela + índice: mediana 2,2 ms; `DROP SCHEMA CASCADE`: 1,6 ms | As migrations precisam ser neutras quanto a schema (sem `public.` fixo). O pool precisa fixar `search_path` por conexão. A tabela de versão do migrator fica por schema |
| **Transação com rollback** | Cada teste roda numa tx aberta e faz rollback no fim (manual ou `go-txdb`) | Sim, dentro da tx | Sim, uma tx por teste | `BEGIN; INSERT; ROLLBACK`: 0,4 ms | O repositório precisa aceitar a tx injetada (interface comum a `pgxpool.Pool` e `pgx.Tx`). Não testa commit, conflito entre conexões nem o caminho real do pool. `go-txdb` só fala `database/sql`, sem pgx nativo, e não suporta transações aninhadas/savepoints |
| **Ids únicos no banco compartilhado** (padrão Redis do `remarks`) | Insere com ids novos e apaga por id no `t.Cleanup` | **Não**: `List` vê dados de dev e de outros testes | Sim | `INSERT` + `DELETE`: 0,8 ms | Exige reescrever os asserts de `List` como "contém, na ordem relativa" ou adicionar uma chave de partição que o domínio do notes não tem |

Fontes das linhas:

- Postgres, [Template Databases](https://www.postgresql.org/docs/current/manage-ag-templatedbs.html): "one can copy any database in a cluster by specifying its name as the template"; "no other sessions can be connected to the source database while it is being copied. `CREATE DATABASE` will fail if any other connection exists when it starts". Com `datistemplate` ligado, qualquer usuário com `CREATEDB` pode clonar.
- Postgres, [CREATE DATABASE](https://www.postgresql.org/docs/current/sql-createdatabase.html): `STRATEGY WAL_LOG` (o default) "is the most efficient strategy in cases where the template database is small". `FILE_COPY` "forces the system to perform a checkpoint both before and after". "`CREATE DATABASE` cannot be executed inside a transaction block."
- [pgtestdb](https://github.com/peterldowns/pgtestdb): identifica o template pelo `Hash()` do migrator e o reaproveita entre execuções. Usa locks Go + advisory locks do Postgres na criação do template. Se o teste passa, o banco é removido no `t.Cleanup`; se falha, o banco fica e a connection string vai para o log. Recomenda servidor só de teste, `fsync=off` e tmpfs ("Do Not connect pgtestdb to the same server that contains your production data"). `pgtestdb.Custom()` devolve a config para quem usa `pgxpool`.
- [go-txdb](https://github.com/DATA-DOG/go-txdb): "When the connection is opened, it starts a transaction and all operations performed on this sql.DB will be within that transaction". O rollback acontece no `Close`.

### Medição local (como reproduzir)

Feita em 2026-10-11 com `postgres:18-alpine` (18.4), Docker 29.8.2 em macOS (10 CPUs). Os tempos vêm do `\timing` do `psql`, do lado do servidor, com n=20 por operação e uma tabela `notes` com índice no template. O cold start foi medido do `docker run` (imagem já em cache) até a primeira query por TCP: **0,75 s** nas três execuções. Os números não valem para o runner do GitHub (outra CPU, disco e pull da imagem); servem de ordem de grandeza. `max_connections` padrão = 100: com `-parallel` = GOMAXPROCS e um pool por teste, o teto de conexões pode ser atingido se cada teste abrir um pool grande.

## 5. Como as migrations chegam ao banco em cada estratégia

| Estratégia | Quando as migrations rodam |
| --- | --- |
| A + banco/schema por teste | O helper do teste aplica as migrations no template (uma vez, protegido por lock) ou em cada schema. Não depende de o serviço ter migrado o banco de dev |
| A + ids únicos / rollback | O banco compartilhado precisa estar migrado antes. Ou o serviço/Tilt migrou o banco de dev (o teste fica acoplado ao estado do dev), ou o helper roda "migrate up", que é idempotente, e serializa por advisory lock |
| B (testcontainers) | O helper roda as migrations no container novo (via `WithInitScripts` com os `.sql`, ou chamando o migrator em Go) e, se quiser, faz `Snapshot` ou usa o banco como template |
| C (service container) | Igual ao B, mas o container vem do workflow. O helper do teste ou um passo do workflow aplica as migrations |

Locks dos migrators, relevantes se vários binários de teste (`-p`) migram o mesmo banco em paralelo:

- golang-migrate (driver postgres) usa `SELECT pg_advisory_lock($1)` ([fonte](https://github.com/golang-migrate/migrate/blob/master/database/postgres/postgres.go), linhas 241/257).
- goose: o `Provider` aceita `WithSessionLocker`, que no Postgres usa advisory lock, e lê migrations de `fs.FS`/`embed.FS` ([post do goose](https://pressly.github.io/goose/blog/2023/goose-provider/)).
- pgtestdb não depende do lock do migrator: ele próprio trava a criação do template.

A escolha do migrator e de onde as migrations moram é de outro ticket do mapa. Qualquer helper de teste vai chamar a mesma API que o serviço usa para migrar.

## 6. Custo e requisitos para `make test` e `ci-backend`

| | Hoje | A (Tilt/compose) | B (testcontainers) | C (service container) |
| --- | --- | --- | --- | --- |
| `make test` local precisa de | Go | Go + `tilt up` (ou compose) rodando, ou `-short`/`test.unit` para pular | Go + Docker daemon | (como A ou B) |
| Overhead fixo por execução | 0 | 0 (o servidor já está de pé) + migração do template (uma vez por hash, com pgtestdb) | ~0,75 s de cold start por container (local, imagem em cache) + pull da imagem na primeira vez + migrations | pull + healthcheck do runner (intervalo de 10 s no exemplo oficial) |
| Overhead por teste | 0 | conforme a unidade (seção 4): ~0,4 ms a ~20 ms | idem | idem |
| Mudança no `ci-backend` | nenhuma | passo `docker compose up -d postgres` + espera (como o remarks), ou `--wait` com healthcheck no compose | nenhuma | bloco `services:` no job |
| Efeito no caminho zero-serviços do `ci-template` | n/a | precisa decidir se o Postgres continua no compose sem serviços | nenhum | o container sobe sem necessidade |
| Separação `test.unit` / `test.component` | inexistente | necessária para quem roda sem Tilt (`-short`) | opcional (Docker costuma estar presente) | necessária localmente |

O `-timeout 30s` atual do `make test` vale por binário de pacote. Em B, cold start + migrations entram nesse orçamento.

## 7. Trade-offs em uma linha cada (sem escolha)

- **A + banco por template**: é o mais próximo do `remarks` (mesmo compose, mesma env com default, mesmo `t.Skip` em `-short`) e preserva os asserts de `List` e o paralelismo. Em troca, depende do Tilt rodando localmente e deixa bancos `test_*` no servidor de dev quando um teste falha.
- **B (testcontainers)**: o `make test` fica autocontido com Docker e o CI não muda. Em troca, entra uma dependência e o Ryuk, o cold start vai para cada pacote e o caminho fica diferente do usado pelo e2e/Tilt.
- **C (service container)**: só existe no CI e precisa de outra fonte local (A ou B), então local e CI divergem. O mapa exige "o mesmo caminho" local e CI explicitamente para o e2e, não para os testes de componente.
- **Schema por teste** é a unidade mais barata depois do rollback, mas exige migrations sem schema fixo e `search_path` por conexão.
- **Rollback** é o mais rápido, mas obriga o repositório a aceitar uma tx injetada e não exercita o pool real nem o commit.
- **Ids únicos no banco compartilhado** são os mais baratos de montar, mas quebram os testes de `List` do notes, que não tem tenant.
