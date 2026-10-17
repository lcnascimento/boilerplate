# Kafka local: broker, UI e criação de tópicos no compose/Tilt

Pesquisa do ticket [BOI-49](https://linear.app/lcnascimento-boilerplate/issue/BOI-49/kafka-local-broker-ui-e-criacao-de-topicos-no-composetilt) (mapa BOI-47). Data: 2026-10-17.

## Pergunta

Qual a melhor forma de rodar Kafka localmente no `backend/docker-compose.yml` + Tilt deste boilerplate?

- Broker: Apache Kafka oficial em KRaft (`apache/kafka`) vs. Redpanda: footprint, boot, healthcheck, compatibilidade com o cliente do Watermill.
- UI de inspeção: kafbat/kafka-ui, Redpanda Console, AKHQ.
- Criação de tópicos: auto-create vs. declarativa; como declarar `notes.note.v1` com N partições.
- Testes de integração: módulos Kafka/Redpanda do testcontainers-go.

## Resposta curta

| Decisão | Recomendação |
|---|---|
| Broker | `apache/kafka:4.3.1` (JVM, KRaft, nó único combinado), dois listeners: `INTERNAL://kafka:19092` (rede `boilerplate`) e `EXTERNAL://localhost:9092` (serviços Go rodando no host via Tilt) |
| UI | `redpandadata/console:v3.12.0` em `localhost:8085` (alternativa: `kafbat/kafka-ui:v1.5.0`) |
| Tópicos | `auto.create.topics.enable=false` no broker; cada serviço declara os próprios tópicos num alvo `make kafka.topics` idempotente (`kafka-topics.sh --create --if-not-exists`), chamado por um `local_resource` `notes-topics` no Tiltfile do serviço, no mesmo molde de `notes-db` |
| Partições de `notes.note.v1` | 3 localmente (exercita rebalance e ordenação por chave `NoteID`) |
| Testes de integração | módulo `redpanda` do testcontainers-go (rápido, de primeira classe); `kafka.Run` com `confluentinc/confluent-local:8.0.x` se precisar de paridade com Apache Kafka |
| `go-kit/kafka` `EnsureTopics` | não usar: `kafka/v1.2.4` não é importável (ver abaixo) e traz um terceiro cliente Kafka (`segmentio/kafka-go`) |

## Contexto do repositório

- `backend/docker-compose.yml` tem só o `postgres`, com `container_name`, healthcheck e a rede `boilerplate`; `backend/docker-compose.observability.yml` traz a stack de observabilidade.
- `backend/Tiltfile` sobe os dois composes com `docker_compose(..., wait=True)` (Tilt espera o healthcheck) e declara cada serviço com `dc_resource`.
- `backend/services/notes/Tiltfile` prepara o banco com `local_resource('notes-db', cmd='make db.setup -s')`, e o `Makefile` do serviço faz `docker exec -i postgres psql ...` de forma idempotente. A criação de tópicos segue esse padrão: quem é dono do recurso declara, o compose compartilhado só sobe a infraestrutura.
- Os binários Go rodam **no host** (`serve_cmd` do Tilt), então o broker precisa anunciar `localhost:9092` para eles e `kafka:19092` para containers na rede (UI, ferramentas).
- Portas já ocupadas: 3000, 3010, 3100, 3200, 4040, 4317, 4318, 5432, 8080 (notes API), 8090, 9090, 12345. Por isso a UI fica em 8085.

## Broker

### Fatos das fontes primárias

- **Watermill usa sarama.** `watermill-kafka/v3` v3.1.4 depende de `github.com/IBM/sarama` ([go.mod](https://github.com/ThreeDotsLabs/watermill-kafka/blob/master/go.mod)). `DefaultSaramaSyncPublisherConfig` e `DefaultSaramaSubscriberConfig` não fixam `config.Version` ([publisher.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/master/pkg/kafka/publisher.go), [subscriber.go](https://github.com/ThreeDotsLabs/watermill-kafka/blob/master/pkg/kafka/subscriber.go)), então vale o `DefaultVersion = V2_8_0_0` do sarama ([utils.go](https://github.com/IBM/sarama/blob/main/utils.go)). Isso fica acima do piso de protocolo do Kafka 4.x (KIP-896 removeu versões de API anteriores ao 2.1).
- **O próprio Watermill testa contra Redpanda**: o `docker-compose.yml` de desenvolvimento do `watermill-kafka` sobe 3 nós `redpandadata/redpanda` ([docker-compose.yml](https://github.com/ThreeDotsLabs/watermill-kafka/blob/master/docker-compose.yml)).
- **Imagem `apache/kafka`**: variáveis `KAFKA_*` viram propriedades (`.` → `_`, `_` → `__`, `-` → `___`). Sem nenhuma configuração do usuário, vale a configuração KRaft padrão de nó único; o exemplo oficial manda fixar `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1`, porque senão vale o padrão 3; `CLUSTER_ID` é opcional (a imagem traz um padrão) ([docker/examples/README.md](https://github.com/apache/kafka/blob/trunk/docker/examples/README.md)).
- **Imagem `apache/kafka-native`** (GraalVM): "well-suited for non-production development and testing scenarios", com boot mais rápido e menos memória ([Docker Hub](https://hub.docker.com/r/apache/kafka-native)).
- **Redpanda `--mode dev-container`** liga `developer_mode` e `overprovisioned`, desliga os autotuners e pula `fsync` ([rpk redpanda mode](https://docs.redpanda.com/current/reference/rpk/rpk-redpanda/rpk-redpanda-mode/)).
- **Licença**: Redpanda e Redpanda Console são BSL (uso livre exceto oferecer como serviço) ([licenses/README.md](https://github.com/redpanda-data/console/blob/master/licenses/README.md)); `kafbat/kafka-ui` é Apache-2.0.
- **arm64**: todas as imagens avaliadas publicam `linux/arm64` (Docker Hub, consultado em 2026-10-17): `apache/kafka:4.3.1`, `apache/kafka-native:4.3.1`, `redpandadata/redpanda:v26.2.5`, `redpandadata/console:v3.12.0`, `kafbat/kafka-ui:v1.5.0`, `tchiotludo/akhq:0.28.0`, `confluentinc/confluent-local:8.0.8`. A versão estável mais recente do Kafka é 4.3.1 (4.3.2 e 4.4.0 ainda estão em RC).

### Medições (MacBook Apple Silicon, Docker Desktop arm64, 10 CPUs, 8 GB para a VM)

Método: `docker compose up -d` seguido de um probe Go no host (franz-go) que repete, até conseguir, "criar `notes.note.v1` (3 partições) + produzir com chave + consumir" via `localhost`. Ou seja, mede "pronto para o app", não "porta aberta". Três execuções a frio (volume novo); memória medida 30 s depois, em repouso.

| Broker | Pronto para o app (3 execuções) | Memória em repouso | Imagem (arm64, comprimida) | Ferramentas na imagem |
|---|---|---|---|---|
| `apache/kafka:4.3.1` | 2.0 / 2.2 / 2.5 s | ~345 MiB | 239 MB | `kafka-topics.sh` e demais CLIs |
| `apache/kafka-native:4.3.1` | 0.6 / 0.8 / 0.8 s | ~234 MiB | 55 MB | nenhuma CLI (só `nc`) |
| `redpandadata/redpanda:v26.2.5` (`--smp=1 --memory=1G`) | 1.3 / 1.4 / 1.5 s | ~280 MiB | 116 MB | `rpk` |

Compatibilidade (programa Go no host, contra cada broker): `watermill-kafka` v3.1.4 (sarama 1.61.1) publica com chave e consome via consumer group, e `segmentio/kafka-go` v0.4.51 `CreateTopics` (o caminho do `EnsureTopics` do go-kit) funciona. **Os três brokers passam.**

Armadilhas encontradas ao medir:

1. **`apache/kafka-native` com volume quebra.** A imagem não tem `/var/lib/kafka`, o named volume é criado como root e o broker morre com `AccessDeniedException: /var/lib/kafka/data/bootstrap.checkpoint.tmp`. Com healthcheck `nc -z`, ele chegou a aparecer "healthy" em 0.7 s antes de morrer: a primeira medição enganou. Só funciona com `KAFKA_LOG_DIRS` em `/tmp` (sem persistência entre `down`/`up`).
2. **Ferramentas dentro do container precisam usar o listener INTERNAL.** `kafka-topics.sh --bootstrap-server localhost:9092` dentro do container recebe de volta o endereço anunciado do listener EXTERNAL. Com uma porta de host diferente da porta do container, ele fica inalcançável e o comando trava 60 s até o timeout. Use sempre `kafka:19092`.
3. **O healthcheck da imagem JVM custa uma JVM por execução** (~0.9 s por `kafka-broker-api-versions.sh`). Com `interval: 2s`, o broker ficou em ~89% de CPU em repouso; com `interval: 10s` + `start_interval: 1s` o custo some e o boot continua rápido (compose `up --wait` em 2.9 s).
4. **Redpanda em `dev-container` vem com `auto_create_topics_enabled=true`** (verificado com `rpk cluster config get`), e `rpk topic create` não é idempotente: sai com código 1 e `TOPIC_ALREADY_EXISTS` na segunda vez.

### Escolha: `apache/kafka` (JVM)

- **Paridade**: é o broker que roda em produção (Apache Kafka em qualquer provedor gerenciado). Diferenças de comportamento entre Redpanda e Kafka aparecem justamente em bordas como configuração de tópico, transações e coordenação de grupos, que é onde não queremos surpresa.
- **Custo**: a diferença medida é ~1 s de boot e ~65 MiB, irrelevante perto da stack de observabilidade que já sobe junto.
- **Ferramentas embutidas**: `kafka-topics.sh --create --if-not-exists` dá criação de tópico idempotente sem imagem extra, e `kafka-broker-api-versions.sh` dá um healthcheck que testa o protocolo, não só a porta.
- **Contra o `kafka-native`**: sem CLI (o healthcheck vira `nc`, que deu falso positivo), sem persistência por volume, e é explicitamente uma imagem de dev/teste. O ganho de 1 s não compensa.
- **Contra o Redpanda**: bom candidato (é o que o Watermill usa no CI), mas é BSL, não é o broker de produção e o auto-create vem ligado. Fica como escolha para testes de integração (abaixo).

## UI de inspeção

| UI | Imagem | Pronta em | Memória | Licença | Notas |
|---|---|---|---|---|---|
| Redpanda Console | `redpandadata/console:v3.12.0` | ~3 s | **~48 MiB** | BSL (recursos enterprise exigem licença e são ignorados sem ela) | Funciona com Apache Kafka só com `KAFKA_BROKERS` (testado: listou `notes.note.v1` com 3 partições) |
| kafbat/kafka-ui | `kafbat/kafka-ui:v1.5.0` | ~3 s | ~405 MiB | Apache-2.0 | Fork mantido do provectus/kafka-ui, cujo último push foi em 2024-07, ou seja, abandonado. Config via `KAFKA_CLUSTERS_0_*` |
| AKHQ | `tchiotludo/akhq:0.28.0` | não medido | n/d | Apache-2.0 | Imagem de 395 MB; não avaliado em execução |

Fontes: [configuração do Console](https://docs.redpanda.com/current/console/config/configure-console/) (variáveis de ambiente derivadas do YAML; `KAFKA_BROKERS`; porta 8080; recursos enterprise ignorados sem licença), [README do kafbat](https://github.com/kafbat/kafka-ui/blob/main/README.md).

**Escolha: Redpanda Console**, por ser 8x mais leve com o mesmo resultado prático (tópicos, mensagens com chave/headers, consumer groups e lag). Se a licença BSL incomodar quem herdar o boilerplate, `kafbat/kafka-ui` é a troca direta (só muda imagem e variáveis).

## Criação de tópicos

- **Auto-create desligado** (`KAFKA_AUTO_CREATE_TOPICS_ENABLE: "false"`; o padrão do Kafka é `true`, ver [`auto.create.topics.enable`](https://kafka.apache.org/documentation/#brokerconfigs_auto.create.topics.enable)). Com auto-create, um erro de digitação no nome do tópico cria silenciosamente um tópico de 1 partição (`num.partitions=1` na config padrão da imagem). Desligado, o publisher do Watermill falha em ~6 s com `Request was for a topic or partition that does not exist on this broker` (testado com `notes.typo.v1`).
- **Declarativo e idempotente, por serviço**, no mesmo molde de `notes-db`:

```make
# backend/services/notes/Makefile
KAFKA_TOPICS := docker exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:19092

.PHONY: kafka.topics
kafka.topics:
	@$(KAFKA_TOPICS) --create --if-not-exists --topic notes.note.v1 --partitions 3 --replication-factor 1
```

```python
# backend/services/notes/Tiltfile
local_resource(
    'notes-topics',
    labels=['notes'],
    cmd='make kafka.topics -s',
    deps=['Makefile'],
    resource_deps=['kafka'],
)
# notes-api-frontend (e o futuro worker de outbox) ganham 'notes-topics' em resource_deps.
```

Verificado: a primeira execução cria o tópico em ~1 s, a segunda sai com código 0 sem mudar nada, e o tópico sobrevive a `restart` do broker (volume `kafka_data`). Limite do `--if-not-exists`: ele não converge partições. Para aumentar de 3 para N é preciso `--alter --partitions N`, e diminuir não é possível no Kafka.

- **Por que não um init container no compose**: o compose compartilhado precisaria conhecer os tópicos de cada serviço. Hoje ele não conhece nem o banco `notes`, que é criado pelo `Makefile` do serviço.
- **Por que não `go-kit/kafka` `EnsureTopics` no boot do app**:
  - `github.com/lcnascimento/go-kit/kafka@v1.2.4` **não é importável**: o `go.mod` publicado exige `github.com/lcnascimento/go-kit/env v0.0.0-00010101000000-000000000000` com `replace => ../env`, e `replace` é ignorado em dependências. `go get` falha com `invalid version: unknown revision 000000000000` (reproduzido).
  - Usa `segmentio/kafka-go`, um terceiro cliente além do sarama (Watermill) e do franz-go (em avaliação).
  - Lê os brokers de estado global de pacote (`brokers[0]`).
  - Criar tópico no boot mistura provisionamento de infraestrutura com o app. Em produção, tópicos devem vir de IaC.
- **Nome com ponto**: `kafka-topics.sh` avisa que `.` e `_` colidem em nomes de métricas. A convenção `<serviço>.<agregado>.v<N>` só usa pontos; basta nunca misturar `_`.

## Testes de integração (testcontainers-go v0.44.0)

| Opção | Resultado (máquina aquecida) | Observações |
|---|---|---|
| `redpanda.Run(ctx, "redpandadata/redpanda:v26.2.5")` | pronto + round-trip em **1.5 s** (teste inteiro 10.8 s, com teardown) | Módulo de primeira classe: `KafkaSeedBroker`, `WithAutoCreateTopics`, Admin API ([docs](https://golang.testcontainers.org/modules/redpanda/)) |
| `kafka.Run(ctx, "confluentinc/confluent-local:8.0.8")` | pronto + round-trip em **2.9 s** (teste inteiro 13.1 s) | Único image suportado pelo módulo `kafka` ([docs](https://golang.testcontainers.org/modules/kafka/)); CP 8.0.x = Apache Kafka 4.0.x ([interoperabilidade](https://docs.confluent.io/platform/current/installation/versions-interoperability.html)) |
| `kafka.Run(ctx, "apache/kafka:4.3.1")` | **falha** (`container exited with code 127`) | O módulo injeta um script que faz `source /etc/confluent/docker/bash-config` ([kafka.go](https://github.com/testcontainers/testcontainers-go/blob/main/modules/kafka/kafka.go)); exigiria `GenericContainer` escrito à mão |

**Escolha: módulo `redpanda`**, um container por pacote (`TestMain`), criando os tópicos no setup do teste, já que o auto-create do módulo vem desligado por padrão. É o mesmo broker que o Watermill usa no próprio CI e é o mais rápido. Se algum teste depender de semântica específica do Apache Kafka, `confluent-local:8.0.x` via `kafka.Run` é a alternativa suportada (Kafka 4.0, uma minor atrás do compose).

## Snippet recomendado

Testado como está (rede renomeada só para não tocar a stack local em execução): `docker compose up --wait` em 2.9 s, Console listando o tópico, Watermill e kafka-go no host via `localhost:9092`.

```yaml
# backend/docker-compose.yml (novos serviços e volume)
services:
  kafka:
    container_name: kafka
    image: apache/kafka:4.3.1
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_LISTENERS: INTERNAL://:19092,EXTERNAL://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: INTERNAL://kafka:19092,EXTERNAL://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT,CONTROLLER:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: INTERNAL
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "false"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
    ports:
      - "9092:9092"
    volumes:
      - kafka_data:/var/lib/kafka/data
    healthcheck:
      test: ["CMD-SHELL", "/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server kafka:19092 > /dev/null 2>&1"]
      interval: 10s
      timeout: 10s
      retries: 30
      start_period: 60s
      start_interval: 1s
    networks:
      - boilerplate

  kafka-console:
    container_name: kafka-console
    image: redpandadata/console:v3.12.0
    environment:
      KAFKA_BROKERS: kafka:19092
    ports:
      - "8085:8080"
    depends_on:
      kafka:
        condition: service_healthy
    networks:
      - boilerplate

volumes:
  kafka_data:
```

```python
# backend/Tiltfile
dc_resource('kafka', labels=['Messaging'])
dc_resource(
    'kafka-console',
    labels=['Messaging'],
    resource_deps=['kafka'],
    links=[link('http://localhost:8085', 'UI')],
)
```

Os dois serviços ficam no `docker-compose.yml` base, não no de observabilidade: o app depende do broker também em `TILT_STACK=minimal`, e o Console custa ~48 MiB.

## Impacto em outros tickets

- **Cliente Kafka (pesquisa irmã)**: os três brokers funcionam com o sarama do `watermill-kafka` v3.1.4 no padrão dele (`V2_8_0_0`); nada no broker restringe a escolha entre sarama e franz-go.
- **`go-kit/kafka`**: a release `kafka/v1.2.4` está quebrada para consumidores (`replace` + pseudo-versão zero de `go-kit/env`). Qualquer ticket que pretenda usar `go-kit/kafka` precisa antes de uma release corrigida no go-kit.
- **Nomenclatura de tópicos**: `<serviço>.<agregado>.v<N>` só com pontos; nunca misturar com `_`.
- **Testes de integração do notes/audit**: Redpanda via testcontainers, com os tópicos criados no setup do teste.

## Reproduzir

Os scripts de medição (compose de bench, probe franz-go, programa de compatibilidade Watermill/kafka-go, testes testcontainers) ficaram fora do repositório. Para reproduzir o essencial: suba o snippet acima, rode `make kafka.topics` duas vezes e publique com Watermill em `localhost:9092`.
