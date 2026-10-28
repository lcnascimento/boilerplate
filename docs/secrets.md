# Secrets

Como o projeto separa config de secret, onde cada secret mora e o que impede um agente de ver ou vazar um deles. Todo o código do projeto é escrito por agentes, então o desenho parte do agente como principal ponto de exposição.

## Modelo de ameaça

Os riscos, por prioridade:

- **(a)** Um segredo entra no contexto do agente: o agente lê um arquivo, despeja o ambiente ou roda uma CLI com credencial, e o valor aparece na saída que ele vê.
- **(b)** Um segredo vaza para código, commit, PR ou issue, por engano do agente ou de uma pessoa.
- **(d)** Um valor de dev chega a produção.

O risco **(c)**, prompt injection que pede ao agente para ler ou publicar um segredo, não tem barreira própria: ele fica coberto pelas barreiras de (a). Um segredo que nunca entra no contexto do agente não pode ser exfiltrado por uma instrução injetada.

## Config e secret

- **Config** é tudo o que pode ter um valor versionado, inclusive credenciais de dev como `postgres:postgres` e os valores com o prefixo `dev-only-`. Fica nos `.env.example`, de onde o `make env` cria os `.env`.
- **Secret** é o que não pode ter um valor de dev versionado. Mora só no Doppler, num único projeto com os ambientes `dev`, `ci` e `prd`, e chega a cada processo pelo ambiente. Nunca é gravado em arquivo.

A classificação é por par (variável, ambiente): a mesma variável pode ser config em dev, com um valor `dev-only-…` versionado, e secret em `prd`. A chave no Doppler é o nome da variável que o processo lê.

## Chaves do Doppler por ambiente

Esta tabela é a única fonte do inventário de secrets. Toda chave nova no Doppler entra aqui no mesmo PR.

| Chave | `dev` | `ci` | `prd` |
| --- | --- | --- | --- |
| `POSTHOG_PROJECT_KEY` | sim, vazia por padrão | | sim |
| `BETTER_AUTH_SECRET` | | | sim |
| `AUDIT_IP_HASH_KEY` | | | sim |
| `NOTES_DATABASE_URL` | | | sim |
| `AUDIT_DATABASE_URL` | | | sim |
| `IAM_DATABASE_URL` | | | sim |

A config `ci` nasce vazia. Em `dev`, `BETTER_AUTH_SECRET`, `AUDIT_IP_HASH_KEY` e as URLs de banco são config, com valor versionado nos `.env.example` (os segredos com o prefixo `dev-only-`, recusado em produção pelo parser de ambiente). Uma chave nunca está ao mesmo tempo num `.env.example` ou `serve_env` e na config `dev`; o `make env.check` falha quando uma chave de `.env.example` (fora o da raiz) casa com `*SECRET*`, `*TOKEN*`, `*_KEY` ou `*PASSWORD*` sem valor `dev-only-…`.

## Scan de segredos

O [gitleaks](https://github.com/gitleaks/gitleaks), na versão fixada em `GITLEAKS_VERSION` no `backend/Makefile` e instalado pelo `make setup`, procura segredos em três camadas:

| Camada | Quando | O que varre |
| --- | --- | --- |
| `pre-commit` do lefthook | a cada commit, em paralelo com os lints | só o que está no stage (`gitleaks git --pre-commit --staged`) |
| job `secrets` do `quality_gate` | em todo PR e na `main` | o histórico inteiro (`fetch-depth: 0`) |
| `make secrets.check` | quando você quiser, antes de abrir PR | o histórico inteiro do checkout local |

O hook local pega o segredo antes que ele vire commit; a CI pega o que escapou dele, inclusive o commit de quem não instalou os hooks ou os pulou. A push protection do GitHub, recomendada no README, é uma quarta camada opcional, que barra o próprio push.

As regras ficam no `.gitleaks.toml` da raiz: as regras padrão do gitleaks (`useDefault = true`) mais a regra `doppler-token`, que pega os tokens de serviço (`dp.st`), de CLI (`dp.ct`), de service account (`dp.sa`), de SCIM (`dp.scim`) e de auditoria (`dp.audit`) do Doppler. As regras padrão só conhecem o token pessoal (`dp.pt`) e pegam os outros, quando pegam, pela heurística genérica de chave e entropia.

O `.gitleaksignore` guarda as exceções, uma por linha, só como fingerprint (`<commit>:<arquivo>:<regra>:<linha>`, impresso pelo gitleaks em cada achado). Não há allowlist por caminho nem por regex: cada exceção aponta para um achado só e fica auditável no `git blame`. Só um humano acrescenta uma linha, e só por dois motivos:

- **falso positivo**: o valor não é segredo;
- **segredo já rotacionado que está na `main`**: a `main` não é reescrita, então o valor antigo fica no histórico. A linha leva um comentário (`#`) com a data da rotação.

Quando o hook barra um commit, ninguém o contorna com `--no-verify`, `LEFTHOOK=0` ou um comentário `gitleaks:allow`. Tire o valor do stage. Se ele pode ser real, trate-o como vazado e rotacione-o (ver Rotação e vazamento). Se é falso positivo, registre o fingerprint no `.gitleaksignore`. As regras do agente estão na seção `## Secrets` do `CLAUDE.md`.

## Barreiras do agente

## Contrato da CI

## Contrato de `prd`

## Rotação e vazamento
