# SLOBurnRate

Severidade: `critical` (`sloth_severity="page"`) ou `warning` (`sloth_severity="ticket"`). Gerado pelo Sloth para todo SLO que um processo herda dos componentes declarados no `o11ygen` (`api-availability`, `api-latency`, `consumer-success`, `consumer-latency`, `outbox-freshness`) ou declara com `WithSLO(...)`.

## Sintoma

O processo do label `service` está consumindo o error budget de 30 dias do SLO do label `sloth_slo` mais rápido do que o objetivo permite, em duas janelas ao mesmo tempo (multi-window, multi-burn-rate):

| Severidade | Janelas | Burn rate | Budget gasto se continuar |
| --- | --- | --- | --- |
| `critical` (page) | 5m e 1h, ou 30m e 6h | 14,4x ou 6x | 2% em 1h, ou 5% em 6h |
| `warning` (ticket) | 2h e 1d, ou 6h e 3d | 3x ou 1x | 10% em 1d, ou 10% em 3d |

Um burn rate de 1x gasta o budget inteiro exatamente nos 30 dias. A janela curta faz o alerta parar logo que o problema acaba; a longa evita disparar por um pico isolado.

## Impacto

Depende do SLO:

- `api-availability`: requisições HTTP respondendo 5xx (health e readiness não contam). Usuários veem erro.
- `api-latency`: requisições HTTP levando mais de 250ms. Usuários veem lentidão.
- `consumer-success`: tentativas de processamento de mensagens falhando. As mensagens são retentadas e, se continuarem falhando, vão para a DLQ.
- `consumer-latency`: tentativas de processamento levando mais de 1s. O lag do consumer group cresce.
- `outbox-freshness`: a mensagem pendente mais antiga do outbox passa de 30s. Os Integration Events atrasam para quem os consome.

Com o budget esgotado, o SLO de 30 dias está violado.

## Diagnóstico

1. Abra o dashboard do serviço (`dashboard_url`) e veja, no topo, o SLI atual, o budget restante e o burn rate do SLO do alerta, e desde quando ele piorou.
2. Confira as janelas no Explore: `slo:sli_error:ratio_rate5m{sloth_slo="<slo>", sloth_service="<serviço>"}` (troque `5m` por `30m`, `1h`, `2h`, `6h`, `1d`, `3d`). `slo:period_error_budget_remaining:ratio` mostra o budget restante em 30 dias.
3. Pela seção do componente no dashboard, encontre o que piorou:
   - `api-*`: as rotas com mais 5xx ou maior p99 (`http_route`), e os traces lentos ou com erro dessas rotas no Tempo.
   - `consumer-*`: o `error_type` das falhas, os tópicos afetados e os envios à DLQ.
   - `outbox-freshness`: se o forwarder está no ar, os envios por segundo e a duração do envio ao Kafka.
4. Veja os logs de erro do processo no Loki (`{service_name="<processo>"}` com `detected_level="error"`) e procure um deploy, uma migração ou uma dependência (Postgres, Kafka, outro serviço) que mudou no mesmo horário.

## Mitigação

- Deploy recente: reverta para a versão anterior.
- Dependência fora do ar ou lenta: siga o runbook do alerta dela (`PostgresDown`, `KafkaBrokersDown`, `DBPoolSaturated`, `ConsumerLagHigh`).
- Pico de tráfego legítimo: escale o processo ou aumente o pool de conexões.
- Mensagens envenenadas (`consumer-success`): corrija o handler; as que já foram para a DLQ precisam ser reprocessadas à mão.
