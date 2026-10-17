# Capacidades do Watermill e do go-kit/kafka para publish, outbox e consumo

Pesquisa do ticket [BOI-48](https://linear.app/lcnascimento-boilerplate/issue/BOI-48/capacidades-do-watermill-e-do-go-kitkafka-para-publish-outbox-e), filho do mapa [BOI-47](https://linear.app/lcnascimento-boilerplate/issue/BOI-47/mensageria-integration-events-de-notes-e-servico-de-audit).
Data: 2026-10-17. Todas as afirmações vêm do código-fonte nas versões abaixo (lidas por inteiro nos trechos relevantes) ou da documentação oficial no próprio repositório do Watermill.

## Versões lidas

| Pacote | Versão | Cliente / dependência relevante |
|---|---|---|
| `github.com/ThreeDotsLabs/watermill` | v1.5.3 (2026-08-25) | Go 1.25, `cenkalti/backoff/v5` |
| `github.com/ThreeDotsLabs/watermill-kafka/v3` | v3.1.4 (2026-07-29) | **`IBM/sarama` v1.50.1**, `dnwe/otelsarama` (pseudo-versão de 2024-03) |
| `github.com/ThreeDotsLabs/watermill-sql/v4` | v4.1.5 (2026-05-14) | **`jackc/pgx/v5` v5.9.2** (adapter nativo) |
| `github.com/voi-oss/watermill-opentelemetry` | v0.1.3 (2022-09-01) | última release há 4 anos |
| `github.com/lcnascimento/go-kit/kafka` | `kafka/v1.2.4` (branch default lida) | **`segmentio/kafka-go` v0.4.51** |

Fontes: `go.mod` de cada repositório ([watermill-kafka](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/go.mod), [watermill-sql](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/go.mod), [go-kit/kafka](https://github.com/lcnascimento/go-kit/blob/main/kafka/go.mod)) e `gh release list`.

## 1. `watermill-kafka`

**Cliente: só sarama.** Não há backend franz-go nem segmentio no repositório oficial; `Marshaler`/`Unmarshaler` trabalham com `*sarama.ProducerMessage`/`*sarama.ConsumerMessage` ([marshaler.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/marshaler.go)).

**Metadata ↔ headers.** `DefaultMarshaler` grava `msg.UUID` no header reservado `_watermill_message_uuid` e copia cada entrada de `msg.Metadata` como header Kafka; o `Unmarshal` faz o inverso ([marshaler.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/marshaler.go)). O payload vai cru (`sarama.ByteEncoder(msg.Payload)`), sem envelope próprio do Watermill.

**Partition key.** `DefaultMarshaler` **não define key** (record com key nula). Para chave é preciso `kafka.NewWithPartitioningMarshaler(func(topic string, msg *message.Message) (string, error))`, que normalmente lê a chave de `msg.Metadata` ([marshaler.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/marshaler.go)). O particionador default do sarama é `NewHashPartitioner` ([sarama config.go](https://github.com/IBM/sarama/blob/v1.50.1/config.go#L571)), então mesma key → mesma partição. A doc oficial diz: "GuaranteedOrder: yes — require partition key usage" e "ExactlyOnceDelivery: no" ([docs/pubsubs/kafka.md](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/docs/content/pubsubs/kafka.md)).

**Publisher.** `sarama.SyncProducer`; `Publish` envia **uma mensagem por vez** e bloqueia até o ack; para no primeiro erro ([publisher.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/publisher.go)). `DefaultSaramaSyncPublisherConfig` só muda `Retry.Max = 10` e `Return.Successes = true`; o resto é default do sarama: `RequiredAcks = WaitForLocal` (acks=1), `Idempotent = false`, `Net.MaxOpenRequests = 5` ([sarama config.go](https://github.com/IBM/sarama/blob/v1.50.1/config.go#L554-L590)). Com retries e 5 requests em voo sem idempotência, **o sarama pode reordenar** dentro da partição; o próprio sarama avisa disso no comentário de `MaxOpenRequests` (config.go linha 46). Para ordem por `NoteID` é preciso sobrescrever: `Idempotent = true`, `RequiredAcks = WaitForAll`, `Net.MaxOpenRequests = 1`, `Version >= V0_11_0_0` (exigências validadas em [config.go#L802-L813](https://github.com/IBM/sarama/blob/v1.50.1/config.go#L802-L813)).

**Subscriber / consumer group.** `SubscriberConfig{Brokers, ConsumerGroup, NackResendSleep (100ms), ReconnectRetrySleep (1s), InitializeTopicDetails, Tracer, OverwriteSaramaConfig}` ([subscriber.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/subscriber.go)). Default do sarama para grupo novo: `Consumer.Offsets.Initial = OffsetNewest` e `AutoCommit.Enable = true` ([config.go#L588-L590](https://github.com/IBM/sarama/blob/v1.50.1/config.go#L588-L590)): **um grupo novo do `audit` perderia o histórico já no tópico** a menos que se configure `OffsetOldest`.

**Semântica de ack/nack** ([subscriber.go `processMessage`](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/subscriber.go)): cada partição é processada **sequencialmente**; a próxima mensagem só é entregue após o Ack. `Ack` → `sess.MarkMessage` (commit do offset, explícito se auto-commit desligado). `Nack` → dorme `NackResendSleep` e **reentrega a mesma mensagem indefinidamente**, bloqueando a partição (preserva ordem, gera head-of-line blocking). Erro de `Unmarshal` derruba o `ConsumeClaim` (não há DLQ automático nesse ponto). O contexto entregue ao handler carrega partição, offset, timestamp e key (`MessagePartitionFromCtx`, `MessageKeyFromCtx`… em [context.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/context.go)).

**Tópicos.** `SubscribeInitialize` cria o tópico via `ClusterAdmin` com `InitializeTopicDetails` e espera as partições ficarem prontas ([subscriber.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/v3.1.4/pkg/kafka/subscriber.go)).

## 2. `watermill-sql` com pgx: publish transacional

**pgx nativo.** O pacote tem `BeginnerFromPgx(conn)` (aceita qualquer coisa com `BeginTx/Exec/Query` do pgx, ou seja, `*pgxpool.Pool`) e `TxFromPgx(pgx.Tx)` ([adapters_pgx.go](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/pkg/sql/adapters_pgx.go)). Não precisa de `database/sql` nem de `stdlib`.

**Publicar dentro da transação.** A doc: "If you need to publish messages within a database transaction, you have to pass a `*sql.Tx` in the `NewPublisher` constructor. You have to create one publisher for each transaction." ([docs/pubsubs/sql.md](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/docs/content/pubsubs/sql.md)). Com pgx: `sql.NewPublisher(sql.TxFromPgx(tx), sql.PublisherConfig{SchemaAdapter: sql.DefaultPostgreSQLSchema{}}, logger)` — construção barata, só guarda o executor. `Publish` faz **um único `INSERT` multi-linha** usando `messages[0].Context()` como ctx ([publisher.go](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/pkg/sql/publisher.go)). `AutoInitializeSchema` é proibido com transação em andamento (o `CREATE TABLE` poderia commitar implicitamente) — o schema tem que vir de migration (goose, no nosso caso).

Encaixe com o código atual: o repositório de `notes` já usa sqlc com `Queries.WithTx(pgx.Tx)` (`internal/repository/notes/internal/db/db.go`), então o adapter de outbox recebe o mesmo `pgx.Tx`.

**Schema da tabela** (`DefaultPostgreSQLSchema`, [schema_adapter_postgresql.go](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/pkg/sql/schema_adapter_postgresql.go)) — uma tabela por tópico, `"watermill_<topic>"`:

```sql
CREATE TABLE IF NOT EXISTS "watermill_<topic>" (
  "offset" BIGSERIAL,
  "uuid" VARCHAR(36) NOT NULL,
  "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "payload" JSON DEFAULT NULL,          -- configurável (BYTEA) via GeneratePayloadType
  "metadata" JSON DEFAULT NULL,
  "transaction_id" xid8 NOT NULL,       -- pg_current_xact_id() no INSERT
  PRIMARY KEY ("transaction_id", "offset")
);
-- offsets por consumer group (offsets_adapter_postgresql.go):
CREATE TABLE IF NOT EXISTS "watermill_offsets_<topic>" (
  consumer_group VARCHAR(255) NOT NULL,
  offset_acked BIGINT,
  last_processed_transaction_id xid8 NOT NULL,
  PRIMARY KEY(consumer_group)
);
```

`xid8`/`pg_current_xact_id()` exigem Postgres 13+; o compose usa `postgres:18-alpine`. Nomes de tópico aceitam `[A-Za-z0-9-$:._]` ([topic.go](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/pkg/sql/topic.go)). **O schema default nunca apaga linhas**: retenção/limpeza do outbox fica por nossa conta. Existe também `PostgreSQLQueueSchema` (sem `xid8`, com coluna `acked`, sem consumer groups), recomendada pela doc só se houver transações muito longas.

**Ordenação e garantias do lado SQL.** O `SELECT` lê em ordem `(transaction_id, offset)` e só enxerga transações já visíveis (`transaction_id < pg_snapshot_xmin(pg_current_snapshot())`), técnica explicitada na doc ("Using last processed transaction ID in PostgreSQL to ensure no messages are lost") — resolve o problema clássico de `SERIAL` atribuído antes do commit. Consequência documentada: uma transação longa atrasa a entrega de tudo que veio depois. A doc marca o SQL Pub/Sub como `ExactlyOnceDelivery: yes` e `GuaranteedOrder: yes`.

Como o subscriber roda ([subscriber.go](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/pkg/sql/subscriber.go)): uma goroutine por `Subscribe` faz polling (`PollInterval` 1s ou `BackoffManager`); cada rodada abre uma transação, trava a linha do consumer group com `SELECT … FOR UPDATE` ([offsets_adapter_postgresql.go](https://github.com/ThreeDotsLabs/watermill-sql/blob/v4.1.5/pkg/sql/offsets_adapter_postgresql.go)), lê até `SubscribeBatchSize` (default 100) linhas, entrega **uma por vez** esperando Ack, e grava o offset do último acked **no fim do lote**, na mesma transação. Implicações:

- Várias réplicas do forwarder são seguras: as outras ficam bloqueadas no `FOR UPDATE` (ativo/passivo implícito, não paralelismo).
- O "exactly once" vale para efeitos **dentro do mesmo Postgres**. Quando o efeito é publicar no Kafka, um crash no meio do lote reenvia tudo que já foi publicado naquele lote: **entrega at-least-once no Kafka**. O próprio comentário do schema: "Higher value, increases a chance of message re-delivery in case of crash … 1 is the safest value".
- `Nack` reenvia a mesma linha após `ResendInterval` (1s) e bloqueia as seguintes: ordem total preservada, head-of-line blocking.
- `AckDeadline` (30s default) cancela o ctx da mensagem; sem deadline o `xmin` pode travar a leitura.

## 3. Forwarder (outbox → Kafka)

`components/forwarder` no módulo core ([forwarder.go](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/components/forwarder/forwarder.go), [envelope.go](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/components/forwarder/envelope.go), [publisher.go](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/components/forwarder/publisher.go), doc [advanced/forwarder.md](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/docs/content/advanced/forwarder.md)):

- **Lado da escrita**: `forwarder.NewPublisher(sqlPublisher, forwarder.PublisherConfig{ForwarderTopic})` embrulha cada mensagem num envelope JSON `{destination_topic, uuid, payload, metadata}` e publica tudo num **único tópico intermediário** (default `forwarder_topic` → tabela `watermill_forwarder_topic`). Ou seja, uma tabela de outbox para todos os tópicos de destino; ordem total entre aggregates.
- **Lado do envio**: `forwarder.NewForwarder(sqlSubscriber, kafkaPublisher, logger, Config)` registra um handler num `message.Router` (próprio ou passado em `Config.Router`) que desembrulha e chama `kafkaPublisher.Publish(destTopic, msg)`. Roda com `Run(ctx)` — é uma goroutine/router, pode viver no mesmo processo do `notes` (um `fx.Lifecycle` hook) ou num binário separado. `Config.Middlewares` permite plugar Retry/Recoverer/tracing; `AckWhenCannotUnwrap` decide se envelope corrompido é descartado ou bloqueia.
- **Preserva UUID e Metadata** do original: o UUID vira o header `_watermill_message_uuid` no Kafka e a metadata vira headers. A partition key deve viajar em Metadata e ser lida por `NewWithPartitioningMarshaler` no publisher Kafka.
- **Garantia ponta a ponta: at-least-once, ordenado.** Publish no Kafka acontece antes do ack/offset SQL; falha no publish → Nack → reenvio da mesma linha (bloqueia as seguintes). Ordem do outbox = ordem de commit; com um forwarder ativo e publisher síncrono idempotente, a ordem por key se mantém no Kafka.
- **Atenção ao envelope**: `Payload []byte` é serializado em JSON como base64 pelo `DefaultMarshaler` do forwarder; o JSON do payload fica opaco na coluna `payload` do outbox (afeta inspeção/debug via SQL, não o consumidor).

## 4. Router, middlewares e ack/nack

`message.Router`: handler sem erro → `Ack`; handler com erro → `Nack` ([router.go](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/message/router.go)). O efeito do Nack depende do subscriber (Kafka e SQL: reentrega imediata da mesma mensagem, bloqueando a partição/tabela).

Middlewares relevantes em [message/router/middleware](https://github.com/ThreeDotsLabs/watermill/tree/v1.5.3/message/router/middleware):

| Middleware | O que faz | Observação para nós |
|---|---|---|
| `Retry` | backoff exponencial (`MaxRetries`, `InitialInterval`, `MaxInterval`, `Multiplier`, `MaxElapsedTime`), `ShouldRetry`, `OnRetriesExhausted`, `ResetContextOnRetry` | retry in-process antes do Nack |
| `PoisonQueue` / `PoisonQueueWithFilter` | após erro, publica a mensagem num tópico de poison com metadata `reason_poisoned`, `topic_poisoned`, `handler_poisoned`, `subscriber_poisoned` e dá Ack | é o "DLQ"; quebra ordem por design (a mensagem sai da fila) |
| `Recoverer` | converte panic em erro com stacktrace | sempre |
| `Deduplicator` | descarta mensagens cujo hash (Adler32/SHA256 do payload) já foi visto; repositório default é **um map em memória com janela de 1 minuto**, "does not support distributed operations" | não serve para idempotência real; usar constraint única no Postgres do `audit` |
| `Timeout`, `Throttle`, `CircuitBreaker`, `DelayOnError`, `IgnoreErrors`, `InstantAck`, `CorrelationID` | utilitários | `InstantAck` quebra at-least-once; evitar |

Composição sugerida (inferência nossa, a partir da semântica acima): `Recoverer` por fora, `PoisonQueue` envolvendo `Retry`, handler no centro — o Poison só recebe o erro depois que o Retry esgota.

## 5. OpenTelemetry

- **Watermill core não tem OTel.** Existem pacotes da comunidade listados em [docs/awesome.md](https://github.com/ThreeDotsLabs/watermill/blob/v1.5.3/docs/content/docs/awesome.md): `voi-oss/watermill-opentelemetry` (última release 2022), `dentech-floss/watermill-opentelemetry-go-extra`, `nkonev/watermill-opentelemetry`.
- `voi-oss/watermill-opentelemetry`: `NewPublisherDecorator` (span `PRODUCER`) e middleware `Trace()` (span `CONSUMER`) baseados apenas em `msg.Context()`. **Não injeta nem extrai contexto em `Metadata`** (não há `Inject`/`Extract` em [publishers.go](https://github.com/voi-oss/watermill-opentelemetry/blob/master/pkg/opentelemetry/publishers.go)/[subscribers.go](https://github.com/voi-oss/watermill-opentelemetry/blob/master/pkg/opentelemetry/subscribers.go)); o `-go-extra` existe justamente para adicionar essa propagação.
- `watermill-kafka` com `Tracer: kafka.NewOTELSaramaTracer()` usa `dnwe/otelsarama`: o producer **extrai** o contexto pai dos headers da própria mensagem, cria o span e reinjeta ([otelsarama producer.go#L281-L302](https://github.com/dnwe/otelsarama/blob/main/producer.go#L281-L302)). O consumer cria span por mensagem, mas **o Watermill não coloca esse span em `msg.Context()`** (o ctx da mensagem vem do subscriber, ver `processMessage`). O repositório `dnwe/otelsarama` não recebe commits desde 2024-05.
- **Ponto decisivo para o outbox**: o contexto de `msg.Context()` **não sobrevive** à ida para o Postgres; o subscriber SQL cria um ctx novo. A única coisa que atravessa é `Metadata` (coluna `metadata JSON`), que o forwarder copia e o `DefaultMarshaler` do Kafka transforma em headers. Portanto o trace só fecha ponta a ponta se o `traceparent` for injetado em `Metadata` **no momento do publish transacional**.
- Como `message.Metadata` é `map[string]string`, `propagation.MapCarrier(msg.Metadata)` já é um `TextMapCarrier`: injetar/extrair é uma linha com `otel.GetTextMapPropagator()`, o mesmo propagador global configurado pelo `go-kit/o11y`. O `carrier.go` do go-kit/kafka faz exatamente isso, só que sobre headers do segmentio.
- O go-kit/kafka, no consumo, cria o span com **link** (não parent) para o contexto extraído ([telemetry.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/telemetry.go)), padrão adequado a mensageria assíncrona; vale copiar a decisão.

## 6. `lcnascimento/go-kit/kafka`

Leitura integral de [producer.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/producer.go), [consumer.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/consumer.go), [admin.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/admin.go), [carrier.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/carrier.go), [telemetry.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/telemetry.go), [value_objects.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/value_objects.go), [errors.go](https://github.com/lcnascimento/go-kit/blob/main/kafka/errors.go) e `example/`.

- **Cliente**: `segmentio/kafka-go` (não sarama).
- **Modelo**: interface `Event{GetTopic(); GetKey() []byte; GetType(); GetVersion()}`; o producer serializa **o evento inteiro com `json.Marshal`** como value. Headers: só os do propagador OTel. Não há UUID de mensagem, nem metadata arbitrária, nem hook para envelope.
- **Producer**: `kafka.Writer` com `Balancer: &kafka.LeastBytes{}`, `RequiredAcks: RequireOne`, `BatchTimeout 5ms`. **`LeastBytes` ignora a key**: "routes messages to the partition that has received the least amount of data" ([kafka-go balancer.go](https://github.com/segmentio/kafka-go/blob/v0.4.51/balancer.go#L70-L76)); só `Hash`/`ReferenceHash` usam a key. Logo, hoje **`GetKey()` não garante partição por `NoteID`** — incompatível com a decisão de ordem por aggregate. Não há opção para trocar o balancer nem ativar idempotência.
- **Consumer**: `Subscriber[T Event]` genérico por tipo, `FetchMessage` + `CommitMessages` após o callback (at-least-once), `StartOffset: FirstOffset` (bom para grupo novo), `RunInBatch`. Mas: erro no callback **encerra `Run`** (sem retry, sem nack, sem DLQ); erro de `json.Unmarshal` também encerra. Como `T` é um tipo concreto, consumo genérico ("qualquer Integration Event") exige um `T` que seja o próprio envelope.
- **Config por env no `init` do pacote**: `KAFKA_BROKERS` e `groupID = OTEL_SERVICE_NAME` (variáveis de pacote em `value_objects.go`). Não dá para injetar brokers/grupo via fx/config nem ter dois grupos no mesmo processo.
- **Admin**: `EnsureTopics(ctx, TopicConfig{Name, NumPartitions, ReplicationFactor}...)`, idempotente (ignora `TopicAlreadyExists`). Útil e independente do resto.
- **Telemetry**: span `PublishMessages` no producer, span `Consume <type>` com link no consumer, logs via `go-kit/o11y/log`.

**Complementa ou concorre?** Concorre nos papéis de publish e consume e não ajuda no outbox. Daria para escrever um `message.Publisher`/`Subscriber` do Watermill sobre ele? Não sem reescrevê-lo: a API pública expõe só `Publish(ctx, ...Event)` com JSON do evento inteiro e `Run(ctx, cb)` sem ack/nack; o `kafka.Writer`/`Reader` é privado; o balancer ignora a key. Um adapter seria, na prática, um novo pub/sub sobre `segmentio/kafka-go` (marshaler, ack/nack, reconexão) — trabalho que o `watermill-kafka` já entrega e testa com a suíte oficial de pub/sub do Watermill. Não vale.

## Recomendação por papel

| Papel | Recomendação |
|---|---|
| **Publish (dentro da transação)** | Adapter da porta `application.EventPublisher` em `notes/internal`, recebendo o `pgx.Tx` do caso de uso: `forwarder.NewPublisher(sql.NewPublisher(sql.TxFromPgx(tx), sql.PublisherConfig{SchemaAdapter: sql.DefaultPostgreSQLSchema{}}, logger), forwarder.PublisherConfig{ForwarderTopic: …})`. Na `Metadata`: partition key (`NoteID`) e `traceparent`/`baggage` injetados com `propagation.MapCarrier(msg.Metadata)`. UUID da mensagem = ID do evento (v7). Tabelas `watermill_<forwarder topic>` e `watermill_offsets_<forwarder topic>` via migration goose; `AutoInitializeSchema` desligado. |
| **Forward (outbox → Kafka)** | `forwarder.NewForwarder` com `sql.NewSubscriber(sql.BeginnerFromPgx(pool), {ConsumerGroup, SchemaAdapter: DefaultPostgreSQLSchema{SubscribeBatchSize: pequeno}, OffsetsAdapter: DefaultPostgreSQLOffsetsAdapter{}})` → `watermill-kafka` `NewPublisher` com `NewWithPartitioningMarshaler` (key da metadata) e sarama `Idempotent=true`, `RequiredAcks=WaitForAll`, `Net.MaxOpenRequests=1`. Middlewares `Recoverer` + `Retry` (sem PoisonQueue: preferir bloquear a perder ordem). Rodar dentro do processo `notes` via `fx.Lifecycle` (réplicas extras ficam em espera no `FOR UPDATE`). Job de limpeza do outbox é nosso. |
| **Consume (`audit`)** | `watermill-kafka` `NewSubscriber` com `ConsumerGroup` explícito e `Consumer.Offsets.Initial = sarama.OffsetOldest`; `message.Router` com `Recoverer`, `PoisonQueue` (tópico de DLQ) envolvendo `Retry`. Idempotência por **unique constraint no ID do evento** (UUID) no Postgres do `audit` (`INSERT … ON CONFLICT DO NOTHING`), não pelo `Deduplicator`. Handler genérico: lê headers/metadata + payload sem interpretar o tipo. |
| **Tracing** | Sem dependência nova: ~20 linhas próprias (decorator de publisher que injeta `otel.GetTextMapPropagator()` em `propagation.MapCarrier(msg.Metadata)` a partir do ctx do caso de uso; middleware de router que extrai da metadata e abre span `CONSUMER` com **link**, como o go-kit/kafka já faz). Aplicar o mesmo middleware no forwarder para ter o span do hop. `OTELSaramaTracer` opcional (traz `dnwe/otelsarama`, parado desde 2024). Não usar `voi-oss/watermill-opentelemetry` sozinho: não propaga. |
| **go-kit/kafka** | Não usar para publish/consume. `EnsureTopics` pode servir para criar `notes.note.v1` no bootstrap local, mas também lê `KAFKA_BROKERS` do env global e puxa `segmentio/kafka-go`; alternativa equivalente é `InitializeTopicDetails`/`SubscribeInitialize` do watermill-kafka ou `sarama.ClusterAdmin`. Vale abrir issue no go-kit sobre o `LeastBytes` (key ignorada) independentemente desta decisão. |

## Pontos que afetam os tickets em aberto

- **Outbox (BOI-50)**: o forwarder usa **uma tabela única** para todos os destinos, com ordem total por commit e um consumidor ativo por vez. Entrega no Kafka é **at-least-once** (duplicata possível no crash, maior com lote grande). O schema default não apaga linhas: decidir retenção. Transações longas em `notes` atrasam todos os eventos.
- **Envelope (BOI-51)**: o forwarder já tem um envelope interno (`destination_topic/uuid/payload/metadata`, payload em base64) que **não chega ao Kafka**; no Kafka chegam value = payload cru e headers = metadata + `_watermill_message_uuid`. Logo o envelope do Integration Event é decisão nossa: campos em headers (Watermill metadata) vs. JSON no value. Para o `audit` genérico, headers (tipo, versão, ID, occurred_at) + payload opaco encaixam naturalmente no `DefaultMarshaler`. O `traceparent` também viaja em metadata/headers.
- **Semântica do consumidor (BOI-52)**: Nack no watermill-kafka reentrega a mesma mensagem para sempre, bloqueando a partição; erro de unmarshal derruba o claim. Sem `PoisonQueue` um evento venenoso trava o `audit`. Grupo novo começa em `OffsetNewest` por default. `Deduplicator` é só em memória: idempotência tem que ser no banco.
