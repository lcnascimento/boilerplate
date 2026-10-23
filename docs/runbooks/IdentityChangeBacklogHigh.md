# IdentityChangeBacklogHigh

Severidade: `critical`. Herdado por todo processo declarado com `WithIdentityChanges()` no `o11ygen` (o `iam-worker`).

## Sintoma

O gauge `iam_identity_change_oldest_pending_age_seconds` (a idade da linha mais antiga de `identity_change` ainda não traduzida em Integration Event) passou de 5 minutos.

## Impacto

As mudanças de identidade (cadastro, troca de senha, exclusão de User, e as demais que os triggers do database `iam` gravam) continuam acontecendo, mas não viram Integration Events: o Audit Log para de receber as mudanças de identidade, e quem consome os tópicos `iam.*` fica sem elas. Nada se perde, porque as linhas ficam em `identity_change` até serem traduzidas.

## Diagnóstico

1. Veja se o `iam-worker` está no ar (há um `ServiceTelemetryAbsent` junto?) e leia os logs dele no Loki: `{service_name="iam-worker"}`.
2. Procure `UNTRANSLATABLE_CHANGE` nos logs: uma linha sem tradução segura o lote inteiro. Os atributos `change_id`, `table` e `operation` do erro dizem qual linha é e de qual trigger ela veio.
3. Procure erros de banco ou de outbox (`OUTBOX_EMIT`) nos logs e um `DBPoolSaturated` no mesmo processo.
4. Confira o acúmulo direto no database `iam`: `SELECT table_name, operation, count(*), min(occurred_at) FROM identity_change GROUP BY 1, 2;`.

## Mitigação

- Linha sem tradução: adicione o mapeamento da tabela e da operação em `backend/services/iam/internal/idomain/change.go` (no mesmo change do trigger que a grava) e faça o deploy do worker. Não apague a linha: ela é uma mudança de identidade que o Audit Log precisa receber.
- Processo caído ou banco inacessível: reinicie o worker e corrija a causa (veja `PostgresDown`); a tradução retoma sozinha e esvazia o acúmulo em ordem.
- Depois de voltar, acompanhe a idade da mudança pendente mais antiga descer até zero na seção "Mudanças de identidade" do dashboard `iam`.
