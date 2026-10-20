# SeriesBudgetExceeded

Severidade: `warning`. Herdado por todo processo de longa duração declarado no `o11ygen` (`service.Daemon`).

## Sintoma

A recording rule `job:series:count` (`count by (job) ({__name__=~".+"})`) do processo passou do teto de séries há 15 minutos. O teto padrão é 2000 séries por `job`; um serviço pode declarar outro em `SeriesBudget` (`backend/tools/o11ygen/services.go`). O label `service` do alerta diz qual processo passou do teto.

## Impacto

Ainda nenhum: a política é detectar, não remover. Mas uma série nova por valor de atributo (ID de entidade, URL crua, mensagem de erro) cresce sem limite, pesa na memória e no disco do Prometheus e deixa as queries dos dashboards lentas. Sem correção, o Prometheus acaba sem memória e todos os serviços ficam sem métricas.

## Diagnóstico

1. Veja quais métricas do processo têm mais séries: `topk(10, count by (__name__) ({job="<service.namespace>/<service.name>"}))`.
2. Para a métrica do topo, veja qual label multiplica as séries: `count(count by (<label>) (<métrica>{job="<service.namespace>/<service.name>"}))` para cada label dela. O label com centenas de valores é o vazamento.
3. Ache no código do serviço onde esse atributo é gravado (os hooks `on<Event>` em `telemetry.go`) e qual valor sem limite entra nele.
4. Se o crescimento é legítimo (mais rotas, mais partições, mais consumer groups), o teto é que ficou pequeno.

## Mitigação

- Vazamento: troque o valor sem limite por um com poucos valores possíveis (o template da rota em vez da URL, o tipo do erro em vez da mensagem) ou tire o atributo da métrica e leve o detalhe para o span ou o log.
- Crescimento legítimo: declare um `SeriesBudget` maior no serviço em `backend/tools/o11ygen/services.go` e rode `make -C backend o11y.generate`.
- As séries antigas somem do `job:series:count` assim que param de receber amostras (cerca de 5 minutos); o alerta resolve depois disso.
