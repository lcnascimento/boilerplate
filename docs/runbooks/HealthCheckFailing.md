# HealthCheckFailing

Severidade: `critical`. Herdado por todo processo Go de longa duração declarado no `o11ygen` (`service.Daemon`) cujo `/health` checa ao menos uma dependência.

## Sintoma

Todo `GET /health` do processo respondeu 5xx nos últimos 2 minutos: `http_server_request_total{http_route="/health"}` só cresceu com `http_response_status_code` 5xx, sem nenhum 2xx. O label `service` do alerta diz qual processo.

O sinal vem de quem faz o probe do `/health` (o `readiness_probe` do Tilt a cada 5s; o kubelet fora dele). Um 503 isolado na subida, seguido de 200, não dispara. Processo que não responde nem exporta é o `ServiceTelemetryAbsent`.

## Impacto

O processo está no ar, mas uma dependência que ele checa no `/health` está fora; a descrição do alerta lista as dele. O Postgres para quem declara `WithDBPool`, o consumer e o Kafka para quem roda `service.Consumer`, o forwarder e o Kafka para quem roda `service.Worker`. O orquestrador o tira do balanceamento: a API para de receber tráfego, o consumer não processa e o forwarder não esvazia o outbox.

## Diagnóstico

1. Leia o motivo nos logs do processo: cada `/health` que falha registra um erro com a causa (`a readiness check failed`, `database did not answer the health ping`, `failed to connect to Kafka`...). No Tilt, o recurso do processo; no Loki, `{service_name="<service.name>"} | detected_level="error"`.
2. Veja se o alerta da dependência também está disparando: `PostgresDown`, `KafkaBrokersDown`. Vários processos com `HealthCheckFailing` ao mesmo tempo apontam para uma dependência comum.
3. Chame o endpoint direto: `curl -i http://localhost:<porta>/health`, com a porta do `readiness_probe` do processo no Tiltfile do serviço.
4. Se as dependências estão no ar, veja se o forwarder ou o consumer do processo parou (os logs `forwarder is not running` ou `events subscriber stopped while the process was running`).

## Mitigação

- Dependência fora: siga o runbook do alerta dela; o processo volta a responder 200 sozinho quando ela volta, e o alerta resolve em até 2 minutos.
- Forwarder ou consumer parado com as dependências no ar: reinicie o processo e investigue o erro nos logs.
