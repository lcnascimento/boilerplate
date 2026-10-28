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
