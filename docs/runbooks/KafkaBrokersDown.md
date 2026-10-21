# KafkaBrokersDown

Severidade: `critical`. Alerta de infraestrutura, grupo `kafka`.

## Sintoma

O `prometheus.exporter.kafka` do Alloy reporta menos de um broker (`kafka_brokers < 1`), a coleta dele falha (`up{job="integrations/kafka"} == 0`), ou o Prometheus não recebe `up` desse job há 5 minutos.

Com tópicos para descrever, o exporter estoura o timeout da coleta quando o broker some: `up` cai para 0 e `kafka_brokers` some, em vez de ir a 0. Por isso o alerta olha `up` também.

## Impacto

Sem broker, nenhum Integration Event é publicado nem consumido: os outboxes acumulam e os consumers param. Sem a métrica, o lag dos consumer groups e os alertas de DLQ também ficam cegos.

## Diagnóstico

1. Veja se o broker está no ar e saudável: `docker ps --filter name=kafka` e o healthcheck do container; no Tilt, o recurso `kafka`.
2. Leia os logs do broker (`docker logs kafka`) atrás de falta de disco ou erro de configuração.
3. Se o broker está no ar, o problema é na coleta: veja o Alloy (`http://localhost:12345`), o componente `prometheus.exporter.kafka.local_kafka` e os logs do Alloy. O exporter derruba a config do Alloy se o broker estava fora quando ele subiu.
4. Confira se a coleta do exporter funciona (`up{job="integrations/kafka"}` e `scrape_duration_seconds{job="integrations/kafka"}`: perto de 10s é o timeout).

## Mitigação

- Broker fora: suba o `kafka` de novo; os forwarders republicam o que acumulou nos outboxes.
- Alloy sem o exporter: reinicie o Alloy depois que o broker estiver saudável.
