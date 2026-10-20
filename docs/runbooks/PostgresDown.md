# PostgresDown

Severidade: `critical`. Alerta de infraestrutura, grupo `postgres`.

## Sintoma

O `prometheus.exporter.postgres` do Alloy não consegue se conectar ao Postgres (`pg_up == 0`), ou o Prometheus não recebe `pg_up` há 5 minutos.

## Impacto

Com o Postgres fora, todo serviço com banco falha: a API devolve erro, os consumers mandam mensagens para retentativa e os forwarders param. Sem a métrica, os painéis e alertas de banco ficam cegos.

## Diagnóstico

1. Veja se o banco está no ar: `docker ps --filter name=postgres` e `docker exec postgres pg_isready -U postgres`; no Tilt, o recurso `postgres`.
2. Leia os logs do banco (`docker logs postgres`) atrás de falta de disco, crash ou erro de configuração.
3. Se o banco está no ar, o problema é na coleta: veja o Alloy (`http://localhost:12345`), o componente `prometheus.exporter.postgres.local_postgres` e os logs do Alloy.

## Mitigação

- Banco fora: suba o `postgres` de novo; os serviços reconectam pelo pool.
- Alloy sem o exporter: reinicie o Alloy.
