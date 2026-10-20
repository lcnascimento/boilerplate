# PostgresConnectionsNearMax

Severidade: `warning`. Alerta de infraestrutura, grupo `postgres`.

## Sintoma

As conexões abertas no Postgres (`sum(pg_stat_activity_count)`) passam de 80% do `max_connections` há 5 minutos.

## Impacto

Perto do teto, o Postgres recusa conexões novas com `too many clients already`. Um pool que tenta crescer, ou um processo que acabou de subir, falha ao conectar.

## Diagnóstico

1. Veja quem abre as conexões: no dashboard `postgres`, seção Conexões, o painel por `application_name` e estado, ou `SELECT application_name, usename, datname, state, count(*) FROM pg_stat_activity GROUP BY 1, 2, 3, 4 ORDER BY 5 DESC;`. O `application_name` segue `<service.name>` de cada processo.
2. Muitas conexões `idle in transaction`: um processo esquece transações abertas. Procure-o pelo `application_name`.
3. Some o `pool_max_conns` de todos os pools e réplicas: se passa do `max_connections`, o teto será atingido em carga.
4. Procure conexões de fora dos serviços (testes, ferramentas) presas no banco.

## Mitigação

- Conexões presas: encerre-as com `SELECT pg_terminate_backend(<pid>);` e corrija o processo que as deixa abertas.
- Pools grandes demais: reduza o `pool_max_conns` dos serviços, ou aumente o `max_connections` do Postgres se a máquina comporta.
