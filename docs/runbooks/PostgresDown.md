# PostgresDown

Severidade: `critical`. Alerta de infraestrutura, grupo `postgres`.

## Sintoma

O `prometheus.exporter.postgres` do Alloy não consegue se conectar ao Postgres (`pg_up == 0`), a coleta dele falha (`up{job="integrations/postgres"} == 0`), ou o Prometheus não recebe `up` desse job há 5 minutos.

## Impacto

Com o Postgres fora, todo serviço com banco falha: a API devolve erro, os consumers mandam mensagens para retentativa e os forwarders param. Sem a métrica, os painéis e alertas de banco ficam cegos.

## Diagnóstico

1. Veja se o banco está no ar: `docker compose ps postgres` e `docker compose exec postgres pg_isready -U postgres` em `backend/`; no Tilt, o recurso `postgres`.
2. Leia os logs do banco (`docker compose logs postgres` em `backend/`) atrás de falta de disco, crash ou erro de configuração.
3. Se o banco está no ar, o problema é na coleta: veja o Alloy (`http://localhost:12345`), o componente `prometheus.exporter.postgres.local_postgres` e os logs do Alloy.

## Mitigação

- Banco fora: suba o `postgres` de novo; os serviços reconectam pelo pool.
- Alloy sem o exporter: reinicie o Alloy.
