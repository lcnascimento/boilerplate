# CriticalLogRecorded

Severidade: `critical`. Herdado por todo processo de longa duração declarado no `o11ygen` (`service.Daemon`).

## Sintoma

O processo registrou ao menos um log com `severity_number >= 21` (níveis `Critical` e `Fatal` do `go-kit/o11y`) nos últimos 5 minutos. O alerta lê a métrica derivada `service:log_records_critical:rate1m`, que o ruler do Loki grava no Prometheus a cada minuto a partir dos logs do serviço. O label `service` diz qual processo, e o valor do alerta é o número aproximado de logs críticos na janela.

## Impacto

Um log crítico costuma ser um panic recuperado, um `Fatal` logo antes de o processo cair, ou um erro que o código classificou como bug (`errors.KindCritical`). Ele pode não queimar budget de SLO (um único request, uma mensagem que foi para a DLQ), por isso alerta na hora: o bug continua lá até alguém olhar.

## Diagnóstico

1. No dashboard `logs`, ou no Explore do Loki, filtre os logs críticos do processo: `{service_name="<service.name>"} | severity_number >= 21`.
2. Leia a mensagem e os atributos do log (o erro e, num panic, o stack trace) e abra o trace pelo `trace_id` para ver a requisição ou a mensagem que causou o erro.
3. Veja se o processo continua no ar: um `Fatal` derruba o processo, e então `ServiceTelemetryAbsent` também deve disparar.
4. Confira se o erro se repete (o valor do alerta, ou o volume por nível em `service:log_records:rate1m{service="<service.name>"}`) ou foi um caso isolado.

## Mitigação

- Bug no código: corrija, faça o deploy e, se uma mensagem ou requisição foi perdida, reprocesse-a (veja `DLQMessageReceived`).
- Dependência fora do ar tratada como erro crítico: restaure a dependência e reclassifique o erro no código, se ele for esperado.
- O alerta resolve sozinho 5 minutos depois do último log crítico; isso não significa que a causa foi tratada.
