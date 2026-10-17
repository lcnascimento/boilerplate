# Schema Registry local: imagem, compose/Tilt e integração com Redpanda Console e testcontainers

Pesquisa do ticket [BOI-54](https://linear.app/lcnascimento-boilerplate/issue/BOI-54/schema-registry-local-imagem-composetilt-e-integracao-com-redpanda) (mapa BOI-47). Data: 2026-10-17.

## Pergunta

Como subir um Confluent Schema Registry no ambiente local, ao lado do broker `apache/kafka:4.3.1` (KRaft), e integrá-lo ao resto do tooling?

1. Imagem e versão do `confluentinc/cp-schema-registry` compatíveis com o broker, a configuração mínima no compose e o recurso no Tilt.
2. Configuração do Redpanda Console para decodificar Protobuf no wire format Confluent com `RecordNameStrategy`.
3. `make schemas.register` idempotente: registrar os `.proto` (com o import de `google/protobuf/timestamp.proto`) e fixar `FULL_TRANSITIVE` por subject.
4. Testes: o registry embutido no módulo Redpanda do testcontainers-go é compatível o bastante com a API Confluent?
5. Lib Go para serializar no wire format Confluent no adapter do repositório, e como o subscriber de `notes/pkg/events` faz parse do framing sem consultar o registry.

Contexto já decidido (não reaberto aqui): broker `apache/kafka:4.3.1` com listeners `kafka:19092`/`localhost:9092` e Redpanda Console em `:8085` ([Kafka local](https://linear.app/lcnascimento-boilerplate/issue/BOI-49/kafka-local-broker-ui-e-criacao-de-topicos-no-composetilt)); CloudEvents binary mode, Protobuf no wire format Confluent, `RecordNameStrategy`, `FULL_TRANSITIVE`, `.proto` em `notes/pkg/events/proto/boilerplate/notes/v1/`, registro explícito sem auto-register ([Envelope e contrato dos Integration Events](https://linear.app/lcnascimento-boilerplate/issue/BOI-51/envelope-e-contrato-dos-integration-events)).

## Resposta curta

| Pergunta | Recomendação |
|---|---|
| Imagem | `confluentinc/cp-schema-registry:8.3.2`. CP 8.3.x = Apache Kafka 4.3.x, a mesma linha do broker. Três variáveis bastam; healthcheck via `bash` + `/dev/tcp` (a imagem não tem `curl`) |
| Tilt | `dc_resource('schema-registry', labels=['Messaging'], resource_deps=['kafka'])`; `kafka-console` passa a depender dele |
| Redpanda Console | Só `SCHEMAREGISTRY_ENABLED=true` e `SCHEMAREGISTRY_URLS=http://schema-registry:8081`. Nada de `serde.protobuf` e nada sobre subject strategy: o Console resolve pelo schema ID do framing |
| `make schemas.register` | Para cada subject: `PUT /config/{subject}` com `FULL_TRANSITIVE` e, em seguida, `POST /subjects/{subject}/versions?normalize=true`. Os dois são idempotentes (verificado). Os imports `google/protobuf/*` não levam `references`. Implementar como um registrador em Go (franz-go `pkg/sr`) que lê os `.proto` embutidos em `notes/pkg/events`, para os testes reaproveitarem; `curl` + `jq` é a alternativa sem Go |
| Testes | Sim. O registry do Redpanda (`redpanda.Run` + `SchemaRegistryAddress`) passou em tudo que usamos: Protobuf com import de Timestamp sem references, `FULL_TRANSITIVE` por subject, `normalize`, lookup, rejeição 409 de mudança incompatível e idempotência |
| Lib Go | `github.com/twmb/franz-go/pkg/sr` (BSD-3, módulo sem nenhuma dependência, Go puro). O producer resolve os IDs no boot com `LookupSchema` e enquadra com `sr.ConfluentHeader`/`sr.Serde`. O consumer usa `ConfluentHeader.DecodeID` + `DecodeIndex` e despacha pelo `ce_type`, sem tocar no registry. Os bytes saem idênticos aos do serializer oficial da Confluent (verificado) |
| Layout dos `.proto` | **Uma mensagem por arquivo** (`note_created.proto`, `note_deleted.proto`). Assim, cada subject `RecordNameStrategy` corresponde a um arquivo, o message index é sempre `[0]` (1 byte) e o comportamento do serializer Go da Confluent (que usa a primeira mensagem do arquivo como nome) não importa |

## Contexto do repositório

- `backend/docker-compose.yml` em `main` tem só o `postgres`. O snippet de `kafka` e `kafka-console` está em [`docs/research/local-kafka.md`](https://github.com/lcnascimento/boilerplate/blob/research/local-kafka/docs/research/local-kafka.md), na branch de pesquisa, e ainda não foi implementado.
- `backend/Tiltfile` sobe os composes com `docker_compose(..., wait=True)` e declara cada container com `dc_resource`. O `Tiltfile` de `notes` prepara recursos do serviço com `local_resource` + `make` (`notes-db`, `notes-migrate`). `notes-topics` (BOI-49) e o `notes-schemas` proposto aqui seguem o mesmo molde.
- `notes/pkg` é um módulo próprio (`backend/services/notes/pkg/go.mod`), hoje com `domain`, `fixture` e `models`. `events` entra como subpacote.
- Em `main` nada fixa `CGO_ENABLED` e ainda não há Dockerfile. Os testes rodam com `-race`.
- Porta 8081: nenhum serviço do repositório a usa (`git grep 8081` vazio). Durante a medição, uma instância avulsa de `notes_api_frontend` com `PORT=8081` ocupava a porta nesta máquina, e por isso o bench usou `18081`. O snippet recomendado mantém 8081, que é a porta padrão do Schema Registry e a que as ferramentas assumem.

## 1. Imagem, compose e Tilt

### Fatos das fontes primárias

- **Versão**: a tabela de interoperabilidade da Confluent mapeia CP 8.3.x para Apache Kafka 4.3.x, 8.2.x para 4.2.x, 8.1.x para 4.1.x e 8.0.x para 4.0.x. Ela diz ainda: "Schema Registry that is included in supported versions of Confluent Platform is compatible with any Kafka broker that is included in supported Confluent Platform versions" ([versions-interoperability](https://docs.confluent.io/platform/current/installation/versions-interoperability.html)). A tag mais recente no Docker Hub é `8.3.2` (2026-09-16), com `linux/arm64`, `amd64` e `s390x` (~556 MiB comprimida em arm64; 1.48 GB descomprimida).
- **Configuração obrigatória**: `SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS`, `SCHEMA_REGISTRY_HOST_NAME` e `SCHEMA_REGISTRY_LISTENERS`. Propriedades viram variáveis com prefixo `SCHEMA_REGISTRY_`, em maiúsculas, com `.` → `_`, `_` → `__` e `-` → `___` ([Docker config reference](https://docs.confluent.io/platform/current/installation/docker/config-reference.html)).
- **Licença**: o Schema Registry é Confluent Community License (uso livre, exceto oferecê-lo como SaaS concorrente). Os módulos `client-*` são Apache-2.0 ([LICENSE](https://github.com/confluentinc/schema-registry/blob/master/LICENSE)).

### Medições (MacBook Apple Silicon, Docker Desktop arm64)

- **Auto-create desligado não atrapalha**: o SR cria o tópico `_schemas` via AdminClient. Com um único broker, ele avisa `Creating the schema topic _schemas using a replication factor of 1, which is less than the desired one of 3` e segue em frente (`kafkastore.topic.replication.factor` padrão = 3). O tópico nasce compactado, com RF 1.
- **Boot**: o SR fica pronto ~2.3 s após o start do container (`Server started` no log). `docker compose up -d --wait` da stack kafka + schema-registry + console, a frio (volume novo), levou 6.4 / 5.4 / 5.4 s.
- **Memória em repouso**: ~268 MiB no SR (JVM), ~346 MiB no `kafka` e ~50 MiB no Console.
- **A imagem não tem `curl` nem `wget`** (RHEL 9 UBI minimal com `bash` e `python3`). Um healthcheck com `curl` deixa o container `unhealthy` para sempre: a primeira tentativa travou o `up --wait` por ~6 min. O healthcheck que funciona faz um `GET /subjects` por `bash` + `/dev/tcp`, sem processo extra pesado.
- **Persistência**: subjects, versões e o config por subject sobrevivem a `restart` do `kafka` e do SR, porque ficam em `_schemas`, no volume `kafka_data`. `down -v` apaga tudo e os IDs recomeçam em 1, o que é mais um motivo para o producer resolver os IDs no boot em vez de fixá-los.

### Snippet recomendado

Testado (com a porta de host trocada para 18081 no bench, veja acima):

```yaml
# backend/docker-compose.yml (novo serviço; kafka e kafka-console vêm de BOI-49)
  schema-registry:
    container_name: schema-registry
    image: confluentinc/cp-schema-registry:8.3.2
    environment:
      SCHEMA_REGISTRY_HOST_NAME: schema-registry
      SCHEMA_REGISTRY_LISTENERS: http://0.0.0.0:8081
      SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS: PLAINTEXT://kafka:19092
    ports:
      - "8081:8081"
    depends_on:
      kafka:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "bash", "-c", "exec 3<>/dev/tcp/localhost/8081 && printf 'GET /subjects HTTP/1.0\\r\\n\\r\\n' >&3 && head -1 <&3 | grep -q ' 200 '"]
      interval: 10s
      timeout: 5s
      retries: 30
      start_period: 60s
      start_interval: 1s
    networks:
      - boilerplate

  kafka-console:
    # ... como em BOI-49, mais:
    environment:
      KAFKA_BROKERS: kafka:19092
      SCHEMAREGISTRY_ENABLED: "true"
      SCHEMAREGISTRY_URLS: http://schema-registry:8081
    depends_on:
      schema-registry:
        condition: service_healthy
```

```python
# backend/Tiltfile
dc_resource(
    'schema-registry',
    labels=['Messaging'],
    resource_deps=['kafka'],
    links=[link('http://localhost:8081/subjects', 'API')],
)
dc_resource(
    'kafka-console',
    labels=['Messaging'],
    resource_deps=['kafka', 'schema-registry'],
    links=[link('http://localhost:8085', 'UI')],
)
```

Fica no `docker-compose.yml` base, ao lado do broker: o producer resolve os schema IDs no boot, então o app depende do registry também em `TILT_STACK=minimal`.

## 2. Redpanda Console

- **Fonte**: no Console v3, `schemaRegistry` é um bloco de topo (`schemaRegistry.enabled`, `schemaRegistry.urls`), e cada chave YAML vira variável de ambiente em maiúsculas, com `_` por nível de indentação ([configure-console](https://docs.redpanda.com/current/console/config/configure-console/)). Para Protobuf: "Redpanda Console detects and uses the Protobuf schemas from the Schema Registry automatically" e "When using Schema Registry for Protobuf, you must not configure `serde.protobuf`" ([deserialization](https://docs.redpanda.com/current/console/config/deserialization/)).
- **Subject strategy não entra**: quem lê não precisa saber o subject. O Console pega o schema ID do framing, busca o schema por ID e usa os message indexes para escolher a mensagem. `RecordNameStrategy` só importa para quem registra e para quem faz lookup.
- **Verificado**: com `SCHEMAREGISTRY_ENABLED=true` e `SCHEMAREGISTRY_URLS=http://schema-registry:8081`, a API `ConsoleService/ListMessages` devolveu os dois registros produzidos em `notes.note.v1` com `encoding: PAYLOAD_ENCODING_PROTOBUF`, `schemaId: 1`/`2` e payload normalizado em JSON, com o `google.protobuf.Timestamp` já formatado (`{"id":"0199f0a0-…","title":"hello","body":"world","createdAt":"2026-10-17T06:00:00Z"}`). Os headers `ce_*` também aparecem. O consumo do Console ficou em ~50 MiB.

## 3. `make schemas.register`

### API REST (Confluent)

| Passo | Chamada | Idempotência |
|---|---|---|
| Fixar compatibilidade | `PUT /config/{subject}` com `{"compatibility":"FULL_TRANSITIVE"}` | Sim. Funciona antes de o subject ter qualquer versão, e por isso vem primeiro: a primeira versão já nasce sob `FULL_TRANSITIVE` |
| Registrar | `POST /subjects/{subject}/versions?normalize=true` com `{"schemaType":"PROTOBUF","schema":"<texto do .proto>"}` | Sim. Reenviar o mesmo schema devolve o mesmo `id` e não cria versão nova (verificado: `versions` continua `[1]`) |

- **`google/protobuf/timestamp.proto` não precisa de `references`.** O registry conhece os well-known types, e a doc da Confluent diz: "Do not upload Google/Protobuf dependencies into Schema Registry" ([serdes-protobuf](https://docs.confluent.io/platform/current/schema-registry/fundamentals/serdes-develop/serdes-protobuf.html)). Verificado: o registro sem `references` funcionou, e o texto armazenado normaliza o campo para `.google.protobuf.Timestamp created_at = 4`. `references` só entram quando um `.proto` nosso importar outro `.proto` nosso, e aí o importado precisa ser registrado antes, sob o próprio subject.
- **`normalize=true` em todo registro e lookup.** "the client can ask Schema Registry to normalize the schema during registration or lookup … a query parameter of `normalize=true` to the REST APIs" ([serdes-develop](https://docs.confluent.io/platform/current/schema-registry/fundamentals/serdes-develop/index.html)). Sem isso, diferenças de formatação (comentários, ordem de imports) viram versões novas.
- **`FULL_TRANSITIVE` funciona como esperado**: um campo novo (`string author = 5`) passa em `POST /compatibility/subjects/{s}/versions`, e trocar `string title` por `int64 title` é rejeitado com `FIELD_SCALAR_KIND_CHANGED` nas duas direções. O registro devolve `409` (`error_code: 40901`). O `make` falha alto, que é o que queremos.
- **Uma mensagem por arquivo.** Registrar um arquivo com `NoteCreated` e `NoteDeleted` sob os dois subjects gera **o mesmo schema ID** para ambos (verificado: `id: 3` nos dois), e só o message index distingue as mensagens. Além disso, a checagem de compatibilidade de cada subject passa a cobrir o arquivo inteiro: uma mudança em `NoteDeleted` seria checada contra o histórico de `NoteCreated`. Com um arquivo por mensagem, subject ↔ arquivo ↔ ID ficam 1:1, e o message index é sempre `[0]`. O lint `STANDARD` do buf passa com esse layout (verificado com buf 1.74.0).
- **buf não registra no Confluent SR**: `buf push` publica no BSR. O papel do buf aqui continua sendo lint, geração e `buf breaking`. O registro é REST.

### Implementação recomendada: registrador em Go no `notes/pkg/events`

Esboço (não implementado; as chamadas usadas foram testadas contra os dois registries):

- `notes/pkg/events` embute os `.proto` (`//go:embed proto`) e expõe a lista `subject → texto`. Com uma mensagem por arquivo, o subject sai do descriptor gerado (`fd.Messages().Get(0).FullName()`) e o texto sai de `fd.Path()` no FS embutido, sem mapa escrito à mão.
- Uma função `RegisterSchemas(ctx, *sr.Client)` faz `SetCompatibility(FULL_TRANSITIVE)` e depois `CreateSchema` com `sr.WithParams(ctx, sr.Normalize)`, para cada subject.
- `make schemas.register` chama um `cmd/schemas` mínimo do serviço `notes` que roda essa função. Os testes de integração, do `notes` e do `audit`, chamam a mesma função contra o registry do testcontainers.
- **Por que não `curl` + `jq`**: funciona (foi assim que a API foi verificada, e o `jq --rawfile` resolve o escape do `.proto` em JSON), mas duplica a lista de subjects em shell, exige `jq` no host e não serve aos testes. Uma única fonte (os `.proto` embutidos) atende ao `make`, aos testes e ao lookup do producer no boot.

```make
# backend/services/notes/Makefile
SCHEMA_REGISTRY_URL ?= http://localhost:8081

.PHONY: schemas.register
schemas.register:
	@go run ./cmd/schemas
```

```python
# backend/services/notes/Tiltfile
local_resource(
    'notes-schemas',
    labels=['notes'],
    cmd='make schemas.register -s',
    deps=['pkg/events/proto/', 'cmd/schemas/'],
    resource_deps=['schema-registry'],
)
# notes-api-frontend (e o forwarder do outbox) ganham 'notes-schemas' em resource_deps.
```

Equivalente em shell, se não quisermos o `cmd` (testado):

```sh
curl -fsS -X PUT -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  --data '{"compatibility":"FULL_TRANSITIVE"}' "$SR/config/$SUBJECT"
jq -n --rawfile schema "$FILE" '{schemaType: "PROTOBUF", schema: $schema}' \
  | curl -fsS -X POST -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
      --data @- "$SR/subjects/$SUBJECT/versions?normalize=true"
```

## 4. Registry do Redpanda no testcontainers-go

Teste com `redpanda.Run(ctx, "redpandadata/redpanda:v26.2.5")` do testcontainers-go v0.44.0 e `SchemaRegistryAddress(ctx)` ([módulo redpanda](https://golang.testcontainers.org/modules/redpanda/)), com o mesmo registrador franz-go usado contra o Confluent. **Passou em tudo** (container pronto em ~1 s; teste inteiro em 10.9 s, com teardown):

| Verificação | Resultado |
|---|---|
| `GET /schemas/types` | `[AVRO PROTOBUF JSON]` |
| Protobuf com `import "google/protobuf/timestamp.proto"` sem `references` | aceito |
| `PUT /config/{subject}` `FULL_TRANSITIVE` antes da primeira versão | aceito; `GET /config/{subject}` devolve `FULL_TRANSITIVE` |
| Registro repetido com `?normalize=true` | mesmo ID e `versions=[1]` (idempotente) |
| Campo novo / troca de tipo | compatível / incompatível. O registro incompatível falha com a mesma mensagem da Confluent (`FIELD_SCALAR_KIND_CHANGED`, `{compatibility: 'FULL_TRANSITIVE'}`) |
| `LookupSchema` com `normalize` | encontra o ID |
| Texto armazenado | normalizado igual ao da Confluent (`.google.protobuf.Timestamp`) |

- **Fonte**: a doc do Redpanda lista os níveis `BACKWARD`…`FULL_TRANSITIVE`, `NONE`, e suporte a references (`name`, `subject`, `version`). As limitações documentadas são em JSON Schema (sem `$ref` externo) e em Avro (`format=resolved` → 501), nenhuma em Protobuf ([schema-reg-api](https://docs.redpanda.com/current/manage/schema-reg/schema-reg-api/)).
- **Limite**: os IDs são atribuídos de forma independente em cada registry, e os testes não podem assumir valores. Isso já é exigência do desenho, já que o producer resolve os IDs no boot. Não testei contexts, modes nem data contracts/rules da Confluent, que não usamos.
- **Recomendação**: um container `redpanda` por pacote de teste (`TestMain`) faz papel de broker e registry ao mesmo tempo. O setup chama `RegisterSchemas` e cria os tópicos. Não há necessidade de subir `cp-schema-registry` no teste.

## 5. Serialização no producer e parse no consumer

### Formato (fonte: [serdes-develop, Wire format](https://docs.confluent.io/platform/current/schema-registry/fundamentals/serdes-develop/index.html))

`0x00` (magic) | schema ID (uint32 big-endian) | message indexes | payload Protobuf. Os message indexes são um array de varints zigzag prefixado pelo tamanho, e o caso mais comum é otimizado: "Since most of the time the actual message type is the first message type (the array [0]) … this special case is optimized to simply 0". Com uma mensagem por arquivo, o header tem sempre 6 bytes: `00 <id:4> 00`.

### Opções avaliadas

| Opção | Go puro | Deps | RecordNameStrategy sem auto-register | Veredito |
|---|---|---|---|---|
| `github.com/twmb/franz-go/pkg/sr` v1.8.0 | sim | **zero** (`go.mod` sem `require`) | IDs vêm de `LookupSchema(subject, texto)`; o subject é passado explicitamente | **recomendado** |
| `confluent-kafka-go/v2/schemaregistry/serde/protobuf` v2.16.0 | **não** | 177 pacotes não-stdlib (Azure, AWS, GCP KMS, Vault, Tink, CEL, libs Avro) contra 18 do franz-go | só funciona derivando o schema do descriptor compilado e com `NormalizeSchemas=true` | descartado |
| Framing escrito à mão | sim | zero | igual ao franz-go | viável (~25 linhas), mas o franz-go já tem isso testado e sem custo de dependência |

Detalhes do `confluent-kafka-go` (verificados no código-fonte e em execução):

- **Exige cgo/librdkafka.** O próprio pacote `schemaregistry` (`config.go`) e o `serde` importam `github.com/confluentinc/confluent-kafka-go/v2/kafka`, que é o binding cgo do librdkafka. Com `CGO_ENABLED=0`, o build quebra (`undefined: TopicPartition`, `undefined: ConfigMap`). Usar só o registry já traz o librdkafka estático para dentro do binário e amarra builds e imagens a cgo.
- **`RecordNameStrategy` + `UseLatestVersion` não funciona.** Com `UseLatestVersion`, o serializer não deriva o schema, a estratégia recebe um `SchemaInfo` vazio, devolve subject `""` e a chamada termina em `404` (reproduzido). Sem `UseLatestVersion`, ele gera o texto a partir do descriptor compilado e faz lookup: sem `NormalizeSchemas=true` deu `40403 Schema not found` contra o `.proto` registrado, e com `NormalizeSchemas=true` funcionou.
- **O `GetRecordName` do Protobuf devolve `messages[0]` do arquivo** ([protobuf.go](https://github.com/confluentinc/confluent-kafka-go/blob/master/schemaregistry/serde/protobuf/protobuf.go)), não a mensagem serializada. Com duas mensagens no mesmo arquivo, a segunda iria para o subject da primeira. É mais um motivo para uma mensagem por arquivo.

### Producer (adapter `internal/repository/notes`)

- **IDs resolvidos no boot, nunca dentro da transação.** O adapter monta o payload dentro da tx do outbox, e uma chamada HTTP ao registry ali acoplaria a escrita no Postgres à disponibilidade do registry. No boot (fx `OnStart`), um `LookupSchema(sr.WithParams(ctx, sr.Normalize), subject, sr.Schema{Schema: <.proto embutido>, Type: sr.TypeProtobuf})` por subject devolve o ID. Se o schema compilado no binário não estiver registrado, o boot falha com `40403`, o que garante na prática o "sem auto-register" e pega um `make schemas.register` esquecido. Medido: 17–52 ms para os dois lookups.
- **Enquadramento sem rede**: `sr.Serde.Register(id, &eventsv1.NoteCreated{}, sr.EncodeFn(...), sr.Index(0))` mapeia o tipo Go para ID e index, e `serde.Encode(msg)` produz os bytes. Também dá para usar direto `(&sr.ConfluentHeader{}).AppendEncode(nil, id, []int{0})` + `proto.Marshal`.
- **Equivalência verificada**: os bytes produzidos pelo franz-go para `NoteCreated` foram **idênticos** (`bytes.Equal`) aos do serializer oficial da Confluent (`RecordNameStrategyType`, `NormalizeSchemas`), e o deserializer da Confluent leu os bytes do franz-go sem erro.
- O valor enquadrado vai como `message.Payload` do Watermill, passa pelo outbox (watermill-sql) e pelo forwarder sem alteração, e os headers `ce_*` vão como metadata. O framing é opaco para o Watermill.

### Consumer (`notes/pkg/events`)

Sem nenhuma chamada ao registry (verificado no round-trip, contra Kafka e contra o Redpanda):

```go
var h sr.ConfluentHeader
id, rest, err := h.DecodeID(value)     // magic byte + schema ID; ErrBadHeader se não for wire format
idx, rest, err := h.DecodeIndex(rest, 1) // [0]; maxLength 1, porque é uma mensagem por arquivo
// despacha por ce_type (header CloudEvents), não pelo ID, que muda entre ambientes:
switch ceType {
case "boilerplate.notes.v1.NoteCreated":
    m := new(eventsv1.NoteCreated); err = proto.Unmarshal(rest, m)
...
}
```

- **Despacho por `ce_type`, não por schema ID**: o ID varia entre ambientes e a cada `down -v`. O `ce_type` é o nome completo da mensagem, igual ao subject. O ID e o index servem para validar o framing e podem ir no `Event` para diagnóstico.
- **Por que não `sr.Serde.Decode` no consumer**: o `Serde` do franz-go indexa os decoders pelo schema ID, ou seja, exigiria conhecer os IDs do ambiente (lookup no registry). As funções do `ConfluentHeader` não têm esse requisito.
- **Evolução**: com `FULL_TRANSITIVE` e só mudanças aditivas, o tipo compilado do consumer lê qualquer versão do mesmo subject. Campos novos viram unknown fields no `proto.Unmarshal`. Um `ce_type` desconhecido não é unmarshalado e vai só para `OnAny`.

## Impacto em outros tickets

- **[Semântica do consumidor de audit: idempotência, retry e poison messages](https://linear.app/lcnascimento-boilerplate/issue/BOI-52/semantica-do-consumidor-de-audit-idempotencia-retry-e-poison-messages)**: framing inválido (`ErrBadHeader`: magic byte diferente de 0, menos de 5 bytes, index malformado) e `proto.Unmarshal` com erro são **erros permanentes**. Devem ir direto para a poison queue, sem retry, porque repetir nunca vai funcionar. Já um registry fora do ar não afeta o consumer, que não fala com ele.
- **`Event.Payload` do `OnAny`** (decisão de [Envelope e contrato dos Integration Events](https://linear.app/lcnascimento-boilerplate/issue/BOI-51/envelope-e-contrato-dos-integration-events)): recomendo que seja o valor Kafka **cru, com o header de 6 bytes**. É ele que permite ao `audit` decodificar depois, buscando o schema pelo ID no registry, inclusive para tipos que o binário não conhece. Expor também `SchemaID` no `Event` ajuda.
- **Layout dos `.proto`**: uma mensagem por arquivo em `notes/pkg/events/proto/boilerplate/notes/v1/` (`note_created.proto`, `note_deleted.proto`). Isso detalha a decisão de BOI-51 sem mudá-la.
- **Ordem no Tilt**: `kafka` → `schema-registry` → `notes-schemas` (e `notes-topics`) → `notes-api-frontend`/forwarder. O producer depende do registry no boot, então o SR precisa estar no compose base.
- **Cliente Kafka**: nada aqui restringe sarama (Watermill) vs. franz-go `kgo`. O `pkg/sr` é um módulo separado, sem dependência do `kgo`.

## Reproduzir

Os artefatos de medição (compose do bench, `.proto` de teste, programa e2e com franz-go e confluent-kafka-go, teste testcontainers) ficaram fora do repositório. Para reproduzir o essencial: suba o snippet acima, registre os dois `.proto` duas vezes com as chamadas da seção 3 (mesmo ID e `versions=[1]`), produza um `NoteCreated` enquadrado com `sr.ConfluentHeader` e abra `notes.note.v1` no Console em `localhost:8085`.
