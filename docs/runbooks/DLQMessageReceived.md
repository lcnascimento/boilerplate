# DLQMessageReceived

Severidade: `critical`. Herdado por todo processo com o componente `consumer`, para as DLQs do consumer group declarado em `WithConsumerGroup(...)` no `o11ygen`.

## Sintoma

O high-water mark de um tópico `<tópico>.<consumer group>.dlq` subiu nos últimos 5 minutos: o consumer mandou ao menos uma mensagem à DLQ. O label `topic` do alerta diz qual DLQ.

## Impacto

A mensagem falhou de forma permanente (erro não retentável, ou retentativas esgotadas) e saiu do fluxo. O efeito dela no serviço consumidor não aconteceu e não vai acontecer sozinho: um Integration Event foi perdido até alguém reprocessá-lo.

## Diagnóstico

1. No dashboard `kafka`, veja na seção DLQ o painel de envios à DLQ por causa: o `error_type` diz por que a mensagem foi rejeitada.
2. No Loki, procure o aviso de envio à DLQ do processo: `{service_name="<service.name>"} |= "message sent to the DLQ"`. O log traz o `message.id` e o `error.type`, e o `trace_id` leva ao trace no Tempo.
3. Leia a mensagem na DLQ pelo Kafka Console (`http://localhost:8085`, perfil `console`): payload, headers e o motivo da falha.
4. Decida se o problema é a mensagem (payload inválido, versão de schema inesperada) ou o consumer (bug, dependência fora do ar tratada como erro permanente).

## Mitigação

- Bug no consumer: corrija, faça o deploy e reprocesse a mensagem, republicando-a no tópico original a partir da DLQ.
- Mensagem inválida: corrija na origem (o serviço que publica) e decida, com o dono do dado, se a mensagem deve ser reprocessada ou descartada.
- O alerta resolve sozinho 5 minutos depois do último envio à DLQ; isso não significa que a mensagem foi tratada.
