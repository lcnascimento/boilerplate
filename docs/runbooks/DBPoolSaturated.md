# DBPoolSaturated

Severidade: `warning`. Herdado por todo processo declarado com `WithDBPool()` no `o11ygen`.

## Sintoma

Mais de 90% das conexões do pool pgx do processo (`pgxpool_acquired_connections / pgxpool_max_connections`) estão em uso há 5 minutos.

## Impacto

Com o pool cheio, cada query espera uma conexão livre antes de rodar. A latência das requisições e do processamento de mensagens sobe, e chamadas com timeout curto começam a falhar.

## Diagnóstico

1. Veja a evolução do uso: `pgxpool_acquired_connections{job="<service.namespace>/<service.name>"}` e `pgxpool_max_connections` no Explore.
2. Procure queries lentas ou presas no Postgres: `SELECT pid, application_name, state, now() - query_start AS duracao, query FROM pg_stat_activity WHERE application_name LIKE '<service.name>%' ORDER BY duracao DESC;`.
3. Procure locks: `SELECT * FROM pg_locks WHERE NOT granted;`.
4. Compare com o volume de requisições ou mensagens do processo: um pico de tráfego satura o pool mesmo com queries rápidas.
5. Nos traces do Tempo, filtre por `service.name` e procure spans de banco longos.

## Mitigação

- Query lenta: corrija a query ou o índice; para destravar agora, cancele a sessão com `SELECT pg_cancel_backend(<pid>);`.
- Lock: encontre e encerre a transação que segura o lock.
- Tráfego legítimo: aumente o `pool_max_conns` na URL do banco do serviço, conferindo antes se o `max_connections` do Postgres comporta (veja `PostgresConnectionsNearMax`).
