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

### `permissions.deny` e o hook `guard-secrets`

O `.claude/settings.json` vale para toda sessão do Claude Code no checkout e em `.claude/worktrees/`. Ele tem duas camadas.

A primeira é o `permissions.deny`, que nega:

- `Read`/`Edit` em `.secrets`, `Read` em `.env*` (exceto `.env.example`) e `Read` em `~/.doppler/**`;
- `doppler`, `tofu`, `make iac.plan`, `make iac.apply` e `make iac.dev-token`, os comandos que usam credencial. O `make iac.check` não usa credencial e continua liberado;
- `git commit … --no-verify` e `LEFTHOOK=0`, que pulariam o gitleaks do `pre-commit`.

A ordem das regras importa: o `Read(!.env.example)` só recorta o `Read(.env*)` porque vem depois dele, no mesmo arquivo.

A segunda é o hook `PreToolUse` `.claude/hooks/guard-secrets.sh`, que roda antes de todo comando `Bash`. Ele lê o JSON do `PreToolUse` no stdin e bloqueia o comando (exit 2, com o motivo em stderr) quando o texto, em qualquer parte, inclusive dentro de `sh -c`, de `python -c` ou num caminho absoluto:

- despeja o ambiente: `env` ou `printenv` sem comando, `export`/`export -p`/`declare -x`/`set` sem argumentos, `ps` com o modificador `e`, `/proc/*/environ`;
- cita `.secrets` (exceto `.secrets.example`) ou `.doppler/`;
- cita `doppler` ou `tofu` como palavra.

`env FOO=bar cmd` continua permitido. O hook é fail-closed: uma entrada que ele não consegue interpretar também é bloqueada. A tabela de casos fica em `.claude/hooks/guard-secrets_test.sh` e roda no `make hooks.test`, que o `make test` chama.

### Sem sandbox e sem env scrub

- **Sandbox do Bash:** é a única barreira do Claude Code que não casa texto, mas é incompatível com o Docker, e o agente roda `tilt up`, `make test` e `make stack.up`. Ele também não cobre MCP nem hooks. Fica desligado.
- **`CLAUDE_CODE_SUBPROCESS_ENV_SCRUB`:** não há secret do projeto no ambiente do Claude Code para limpar. O `DOPPLER_TOKEN` só chega ao processo `doppler` do wrapper, e o `.envrc` não carrega o `.secrets`. A heurística do scrub ainda poderia tirar do ambiente o token do MCP do Grafana, um comportamento que a documentação não descreve. Fica fora.

### Risco aceito

O `deny` e o hook casam o texto do comando. Nas palavras da documentação do Claude Code, isso "isn't a security boundary": um agente determinado escreve o mesmo comando de outro jeito e alcança os valores de `dev` e o token pessoal do administrador.

Esse token fica em `~/.doppler/` depois do `doppler login`, na mesma máquina do agente, e lê `prd`. Um `doppler` rodado pela sessão o usaria sozinho, e contra isso só valem o `deny` e o hook. O que protege `prd` de fato não depende do texto do comando: nenhum token gravado em arquivo do repositório lê `prd`, a passphrase do state de `prd` fica fora do disco e o agente nunca roda `tofu`.

## Contrato da CI

## Contrato de `prd`

## Rotação e vazamento
