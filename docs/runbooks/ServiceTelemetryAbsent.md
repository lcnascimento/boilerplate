# ServiceTelemetryAbsent

Severidade: `critical`. Herdado por todo processo de longa duração declarado no `o11ygen` (`service.Daemon`).

## Sintoma

Nenhuma série `target_info` com `job="<service.namespace>/<service.name>"` chegou ao Prometheus nos últimos 5 minutos. O label `service` do alerta diz qual processo sumiu.

## Impacto

O processo parou, travou antes de exportar, ou a telemetria dele não chega ao Prometheus. No primeiro caso, o que ele faz (servir a API, consumir um tópico, esvaziar o outbox) está parado. No segundo, os dashboards, os SLOs e os outros alertas do processo estão cegos.

## Diagnóstico

1. Veja se o processo está no ar: no Tilt (`tilt up`), o recurso do processo; fora dele, o orquestrador que o roda.
2. Leia os logs do processo no recurso do Tilt, ou no Loki com `{service_name="<service.name>"}` se ainda chegam.
3. Confira no Prometheus se alguma série do processo chega: `count by (__name__) ({job="<service.namespace>/<service.name>"})`. Se só o `target_info` sumiu, o problema é no SDK; se tudo sumiu, é no processo ou no caminho até o Prometheus.
4. Confira o caminho: o Alloy no ar (`http://localhost:12345`), o `OTEL_OTLP_ENDPOINT` do processo e se outros serviços também sumiram (um `ServiceTelemetryAbsent` em todos os processos aponta para o Alloy ou o Prometheus).

## Mitigação

- Processo caído: reinicie-o e corrija a causa do crash vista nos logs (config, banco, broker).
- Processo travado: reinicie-o e guarde um goroutine dump antes, se possível, para investigar.
- Alloy ou Prometheus fora: suba-os de novo; o alerta resolve quando o próximo export (a cada 60s) chega.
