# AuthenticationFailuresHigh

Severidade: `warning`. Herdado por todo processo declarado com `WithActorAuth()` no `o11ygen`.

## Sintoma

Mais de 10% das requisições do processo respondem 401 há 5 minutos. Num processo Go, a taxa vem de `http_server_request_total`, fora `/ping` e `/health`; no `web` (Node), vem dos spans de servidor em `traces_spanmetrics_calls_total`.

## Impacto

Usuários com sessão válida podem estar sendo recusados: as Server Actions do `web` falham e as telas mostram erro. Se a causa é alguém testando senhas, o impacto é o risco de uma conta comprometida.

## Diagnóstico

1. Abra a linha "Autenticação e autorização" do dashboard do serviço e veja o motivo que domina.
2. Processo Go (`iam_actor_authentication_failures_total` por `reason`):
   - `bad_signature` ou `unknown_kid`: o JWKS do `web` mudou (rotação de chave) ou o processo não alcança `AUTH_JWKS_URL`. Procure o log `failed to refresh the JWKS` do processo.
   - `bad_issuer` ou `bad_audience`: `AUTH_ISSUER`/`AUTH_AUDIENCE` do processo divergem do `BETTER_AUTH_URL` e da audience do `web`.
   - `expired`: relógio do host adiantado ou atrasado em relação ao `web`.
   - `missing_header` ou `malformed`: um cliente chama a API sem token, ou com um token que não é do `web`.
3. `web` (`iam_login_failures_total` por `reason`):
   - `invalid_credentials` concentrado num intervalo curto: tentativa de adivinhar senhas. Veja nos logs do `web` de onde vêm as requisições.
   - `rate_limited`: o limitador do Better Auth já está segurando a tentativa.
4. Nos traces do Tempo, filtre por `service.name` e `http.status_code = 401` para ver as rotas afetadas.

## Mitigação

- JWKS fora ou trocado: restabeleça o `web` e o acesso a `/api/auth/jwks`; o processo volta a aceitar tokens no próximo refresh, sem reinício.
- Configuração divergente: corrija `AUTH_ISSUER`, `AUTH_AUDIENCE` ou `AUTH_JWKS_URL` e reinicie o processo.
- Relógio: sincronize o NTP do host.
- Ataque de senha: bloqueie a origem na borda e confira se o rate limit do Better Auth está ativo.
