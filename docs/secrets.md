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

## Barreiras do agente

## Contrato da CI

## Contrato de `prd`

## Rotação e vazamento

Não há rotação periódica: um secret ou um token só é rotacionado quando um evento pede. Quem rotaciona é sempre o administrador humano; o agente que perceber um dos eventos abaixo para e avisa.

### Rotacionar um service token

Recrie o token pelo endereço dele, commite o state e entregue o token novo à pessoa:

```sh
make iac.apply WS=dev REPLACE='doppler_service_token.dev["<pessoa>"]'
make iac.dev-token NAME=<pessoa>
```

O token antigo deixa de valer no apply.

### Saída de um holder

Tirar a pessoa de `dev_token_holders` (em `iac/environments/terraform.tfvars`) num PR e rodar `make iac.apply WS=dev` depois do merge destrói o token dela: a saída é a revogação. Commite o state em seguida.

### Passphrase de state vazada

Os states antigos continuam no histórico do git, cifrados com a passphrase vazada, e guardam os service tokens. Trocar a passphrase não basta: os tokens que aquele state guardou também precisam ser trocados.

1. Gere uma passphrase nova e guarde-a no gerenciador de senhas.
2. Abra um PR de rekey na raiz do workspace (`iac/global/` ou `iac/environments/`):
   - o `key_provider "pbkdf2"` atual mantém o nome, que é o que o state registra, passa a ler `var.previous_state_passphrase` e vira o `fallback` do bloco `state`;
   - um `key_provider "pbkdf2"` novo, com outro nome, lê `var.state_passphrase` e passa a ser o método do `state` e do `plan`;
   - a raiz declara `variable "previous_state_passphrase"` (`sensitive = true`).
3. Depois do merge, rode `make iac.apply` em cada workspace da raiz. Com a variável declarada, o alvo pede também a passphrase anterior. Num workspace cuja passphrase não vazou, digite a mesma nas duas perguntas: ele só passa para o `key_provider` novo. Confira no `git status` que cada state mudou e commite os states.
4. Abra um segundo PR que tira o `key_provider` antigo, o `fallback` e a variável `previous_state_passphrase`.
5. Recrie com `REPLACE` todo token que o state da passphrase vazada guardou (no workspace `dev`, um por holder) e entregue os tokens novos.

### Token pessoal do administrador vazado

O token do `doppler login` lê todos os ambientes, `prd` incluído.

1. Revogue o token no dashboard do Doppler.
2. Leia o activity log do workplace desde a criação do token, para saber o que foi lido ou mudado.
3. Rotacione todos os secrets de `prd` e de `ci` na origem (no provedor de cada credencial) e grave os valores novos pelo dashboard.
4. Recrie todos os service tokens com `REPLACE` e entregue os novos.
5. Rode `doppler login` de novo.

### Segredo commitado

Um segredo que chegou a um commit está queimado, mesmo que o commit nunca tenha saído da máquina.

- Fora da `main` (branch local ou PR aberto): rotacione o segredo e reescreva o branch para tirá-lo.
- Na `main`: rotacione o segredo e não reescreva o histórico; a `main` é imutável. O humano registra o fingerprint do achado do gitleaks no `.gitleaksignore`, com a data da rotação, para que o scan do histórico continue verde. O agente nunca edita o `.gitleaksignore`.

### Segredo que passou pelo agente

Um segredo real que entrou no contexto do agente (numa leitura, numa saída de comando ou num prompt) está vazado, mesmo sem commit: o contexto pode ter ido para logs, transcripts ou para o provedor do modelo. O agente para e avisa o humano, que rotaciona o segredo. O agente nunca rotaciona por conta própria.

## A estrutura em OpenTofu

A estrutura do Doppler fica em `iac/`, em duas raízes. O roteiro de uso está na seção `## Doppler` do `README.md`.

- `global` declara o projeto e os ambientes. O projeto é criado à mão (`doppler projects create`) e adotado por blocos `import`, junto com os ambientes que o Doppler cria com ele; o `stg` vira `ci` por update in-place. Cada ambiente leva `personal_configs = false`.
- `environments` é um único código para os workspaces `dev`, `ci` e `prd`, com o ambiente vindo de `terraform.workspace`. O que é de um ambiente só usa `count`/`for_each` condicionado ao workspace, como os service tokens de `dev`.
- O código nunca declara `doppler_secret` nem lê o data source `doppler_secrets`: os valores entrariam no state e no diff do plano. Os valores entram só pelo dashboard.
- O state é `local`, cifrado (`pbkdf2` + `aes_gcm`, `enforced = true`) e versionado. Cada raiz e cada workspace tem a própria passphrase, guardada no gerenciador de senhas e fora do Doppler: um acesso ao Doppler não abre o state, e o apply não depende do Doppler para abrir o state.
- O apply é sempre local, feito pelo administrador na `main`. A CI só roda o `make iac.check`, que não precisa de credencial.

## Plano gratuito e plano Team

O template implementa só o plano gratuito do Doppler:

- um único projeto com `dev`, `ci` e `prd`;
- service tokens read-only por config, um por pessoa em `dev`;
- o apply autenticado pelo token pessoal do administrador, que lê todos os ambientes.

O plano Team fica só descrito, sem código condicional no template. Com ele, o apply deixaria de usar o token pessoal: cada workspace teria a sua service account, com um papel limitado ao ambiente dela, e a CI poderia autenticar por OIDC, sem nenhum token guardado. Adotar o Team é uma mudança no `iac/` feita quando o projeto precisar.

## Por que não há configs por pessoa

Os ambientes levam `personal_configs = false`, e o template não usa branch configs por pessoa nem por sessão de agente:

- uma config por pessoa criaria valores fora do Terraform e fora da tabela de chaves, que deixaria de ser a única fonte do inventário;
- a config `dev` só guarda valores de dev, iguais para todos, e cada pessoa lê com o próprio token read-only, revogável sozinho;
- um valor que muda de pessoa para pessoa é config local e fica no `.env`, não no Doppler.
