# ConsumerLagHigh

Severidade: `warning`. Herdado por todo processo com o componente `consumer`, para o consumer group declarado em `WithConsumerGroup(...)` no `o11ygen`.

## Sintoma

A soma das mensagens não commitadas do consumer group (`kafka_consumergroup_uncommitted_offsets`) passa de 100 há 5 minutos.

## Impacto

Os Integration Events chegam ao serviço consumidor com atraso. O que depende deles (projeções, auditoria, efeitos colaterais) fica defasado em relação ao serviço que publica. Nada se perde enquanto a retenção do tópico não vencer.

## Diagnóstico

1. Abra o dashboard do serviço (`dashboard_url`) e veja a seção `consumer`: o lag do group por tópico, as mensagens/s e a duração do processamento. O lag por partição e a produção no tópico ficam no dashboard `kafka`.
2. Lag crescendo e consumo em zero: o consumer está parado. Veja se o processo está no ar (há um `ServiceTelemetryAbsent` junto?) e os logs dele no Loki.
3. Consumo estável mas abaixo da produção: o processamento ficou lento. Veja a duração em `messaging_process_duration_seconds` e os traces do consumer no Tempo (banco lento, `DBPoolSaturated`, dependência externa).
4. Lag só numa partição: uma mensagem pode estar em retentativa. Procure retentativas e erros nos logs do processo.

## Mitigação

- Consumer parado: reinicie o processo e corrija a causa nos logs.
- Processamento lento: corrija o gargalo (query, dependência). Para drenar o acúmulo, suba mais réplicas do consumer, até o número de partições do tópico.
- Pico de produção pontual: acompanhe; o alerta resolve sozinho quando o lag volta para baixo de 100.
