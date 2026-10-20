# OutboxAgeAbsent

Severidade: `critical`. Herdado por todo processo com o componente `worker` (o forwarder do outbox) no `o11ygen`.

## Sintoma

O processo não reporta o gauge `<serviço>_outbox_oldest_pending_age_seconds` (a idade da mensagem pendente mais antiga do outbox) há mais de 5 minutos.

## Impacto

Sem o forwarder, ninguém leva o outbox para o Kafka: os Integration Events do serviço ficam presos no banco e nenhum consumidor os recebe. Sem o gauge, o SLO de frescor do outbox também fica cego. A escrita no serviço continua funcionando, então o problema não aparece para quem usa a API.

## Diagnóstico

1. Veja se o processo está no ar (há um `ServiceTelemetryAbsent` junto?) e leia os logs dele no Loki: `{service_name="<service.name>"}`.
2. Se o processo está no ar mas não reporta o gauge, a consulta da idade está falhando: procure erros de banco nos logs e um `DBPoolSaturated` no mesmo processo.
3. Confira o tamanho do outbox direto no banco do serviço, para saber quanto acumulou.

## Mitigação

- Processo caído: reinicie o forwarder e corrija a causa do crash nos logs.
- Banco inacessível: resolva a conexão (veja `PostgresDown`); o forwarder retoma sozinho e esvazia o acúmulo em ordem.
- Depois de voltar, acompanhe a idade da pendente mais antiga descer até perto de zero.
