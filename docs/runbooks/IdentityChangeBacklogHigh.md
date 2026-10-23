# IdentityChangeBacklogHigh

Severidade: `critical`. Herdado por todo processo declarado com `WithIdentityChanges()` no `o11ygen` (o `iam-worker`).

## Sintoma

O gauge `iam_identity_change_oldest_pending_age_seconds` (a idade da linha mais antiga de `identity_change` ainda não traduzida em Integration Event) passou de 5 minutos.

## Impacto

As mudanças de identidade (cadastro, troca de senha, exclusão de User, e as demais que os triggers do database `iam` gravam) continuam acontecendo, mas não viram Integration Events: o Audit Log para de receber as mudanças de identidade, e quem consome os tópicos `iam.*` fica sem elas. Nada se perde, porque as linhas ficam em `identity_change` até serem traduzidas.

## Diagnóstico

1. Veja se o `iam-worker` está no ar (há um `ServiceTelemetryAbsent` junto?) e leia os logs dele no Loki: `{service_name="iam-worker"}`.
2. Procure `UNTRANSLATABLE_CHANGE` nos logs: uma linha sem tradução segura o lote inteiro. Os atributos `change_id`, `table` e `operation` do erro dizem qual linha é e de qual trigger ela veio.
3. Procure erros de banco ou de outbox (`failed to write the Integration Event to the outbox`) nos logs e um `DBPoolSaturated` no mesmo processo.
4. Confira o acúmulo direto no database `iam`: `SELECT table_name, operation, count(*), min(occurred_at) FROM identity_change GROUP BY 1, 2;`.

## Mitigação

- Linha sem tradução: adicione o mapeamento da tabela e da operação em `backend/services/iam/internal/idomain/change.go` (no mesmo change do trigger que a grava) e faça o deploy do worker. Não apague a linha: ela é uma mudança de identidade que o Audit Log precisa receber.
- Linha sem tradução e deploy demorado: separe a linha para destravar as outras, sem perdê-la. No database `iam`, com o `change_id` do erro:

  ```sql
  CREATE TABLE IF NOT EXISTS identity_change_held (LIKE identity_change);
  WITH held AS (DELETE FROM identity_change WHERE id = <change_id> RETURNING *)
  INSERT INTO identity_change_held SELECT * FROM held;
  ```

  Depois do deploy com o mapeamento, devolva as linhas separadas e apague a tabela:

  ```sql
  INSERT INTO identity_change (transaction_id, occurred_at, table_name, operation, fields, actor_id, actor_organization_id, ip_hash)
  SELECT transaction_id, occurred_at, table_name, operation, fields, actor_id, actor_organization_id, ip_hash
  FROM identity_change_held ORDER BY id;
  DROP TABLE identity_change_held;
  ```

  O evento sai com o horário e o id de correlação originais, mas depois dos eventos que a linha segurava: quem consome o tópico vê esse fato fora de ordem.
- Processo caído ou banco inacessível: reinicie o worker e corrija a causa (veja `PostgresDown`); a tradução retoma sozinha e esvazia o acúmulo em ordem.
- Depois de voltar, acompanhe a idade da mudança pendente mais antiga descer até zero na seção "Mudanças de identidade" do dashboard `iam`.
