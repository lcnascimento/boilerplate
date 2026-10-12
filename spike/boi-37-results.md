# BOI-37: boot do stack do Tilt no GitHub Actions

Spike descartável (branch `spike/boi-37-tilt-ci-boot`, nunca mergeado). Workflow `.github/workflows/spike-e2e.yml` com três jobs em paralelo, mesma máquina (`ubuntu-latest`), mesmo commit:

- **baseline**: os passos do `ci-e2e` atual (`next build` + `next start` na 3001, binário do `notes` na 8081, via `webServer` do Playwright).
- **T1**: e2e como `local_resource` `web-e2e` (`resource_deps=['web', 'notes-api-frontend']`), definido só quando `config.tilt_subcommand == 'ci'`; o job roda `tilt ci`.
- **T2**: `tilt up --stream &` + `tilt wait --for=condition=Ready uiresource/web uiresource/notes-api-frontend` + `playwright test`, com um passo de aquecimento (`curl` em cada rota) antes do e2e.

Restrição de recursos: com `E2E_STACK` definido, o `Tiltfile` raiz chama `config.set_enabled_resources(['frontend-deps', 'web', 'notes-api-frontend'] (+ 'web-e2e' no T1))`. O Playwright, com `E2E_STACK=tilt`, aponta para 3000/8080 e não sobe `webServer`. Os `.env` do backend vêm do `.env.example` (OTel ligado, `OTEL_LOGS_EXPORTER=otlp`, sem Alloy).

## Runs

| Run | Cache `next dev` | baseline (job / e2e) | T1 (job / `tilt ci`) | T2 (job / boot / aquecimento / e2e) |
|---|---|---|---|---|
| [38250454891 #1](https://github.com/lcnascimento/boilerplate/actions/runs/38250454891/attempts/1) | frio | 86s / 27s | 87s / 30s (boot 9,6s + e2e 19,6s) | 123s / 12,0s / 7s / 11,9s |
| [38250454891 #2](https://github.com/lcnascimento/boilerplate/actions/runs/38250454891/attempts/2) | quente | 79s / 22s | 97s / 30s (boot 8,2s + e2e 21,2s) | 85s / 7,5s / 3s / 7,9s |
| [38251080297](https://github.com/lcnascimento/boilerplate/actions/runs/38251080297) | quente | 82s / 21s | 94s / 31s (boot 9,2s + e2e 21,0s) | 181s* / 10,3s / ~3s / 9,3s |

\* inclui um `sleep 90` deliberado (checagem do OTel); sem ele, ~91s.

Os 9 testes passaram em todas as runs, sem retry nem flaky.

## Achados

1. **O boot do stack é barato: 7,5–12s** do comando ao `web` + `notes` prontos (Tiltfile carrega em ~0,1–2,4s, `make build.api.frontend` com cache do Go, `pnpm install` no-op, `next dev` "Ready in ~350ms"). Instalar o Tilt pelo script oficial leva 1–2s (instala a última versão, hoje v0.37.8; não fixada).
2. **Tempo total do job empata com o baseline.** Baseline 79–86s; T1 87–97s; T2 85–123s. Os ~55–60s de setup (pnpm, node, Go, `playwright install --with-deps chromium` de 20–36s) são idênticos nos três e dominam o job; a diferença vem do e2e em si. Os ~1m33s citados para o `ci-e2e` eram tempo de relógio do run (com fila); o job medido hoje é ~80s.
3. **`set_enabled_resources` impede o compose de subir**: nenhum container nem imagem do projeto no runner (só as imagens pré-instaladas do GitHub). Observabilidade e Scalar ficam fora sem precisar de `tilt ci <nomes>`, e a lista vive dentro do Tiltfile.
4. **`next dev` compila sob demanda, mas o custo é pequeno neste app.** Primeira requisição por rota, cache frio: `/` 4,3s, `/notes` 1,2s, `/notes/<id>` 0,8s, 404 0,6s; com cache quente: `/` 0,8–1,0s, demais 0,26–0,55s. No T1 (sem aquecimento) o Next logou `Compiling /notes` em 784ms. Nenhum timeout com os padrões do Playwright.
5. **O cache do `next dev` só funciona no T2.** `.next/dev/cache` salvo pelo T2: ~110–117 MiB, restaurado nas runs seguintes (e2e 11,9s → 7,9–9,3s; aquecimento 7s → 3s). No **T1 o cache salvo tem ~815 B em todas as runs**: o `tilt ci` encerra o `next dev` ao sair antes de o Turbopack persistir o cache, então o T1 nunca aproveita cache (e2e fica em ~20s).
6. **O `notes` sobe e funciona sem o Alloy**: `/health` ok, e2e verde, e 90s depois do e2e o serviço segue saudável, sem erro nem atraso no stdout/stderr. Efeito colateral: com `OTEL_LOGS_EXPORTER=otlp` os logs da aplicação vão para o exportador e somem; no CI o `tilt logs notes-api-frontend` só mostra o build, o fx e o probe. Para depurar uma falha no CI, os logs do serviço não existem a menos que o exportador mude (ex.: `OTEL_SDK_DISABLED=true` ou um exportador de console).
7. **T1 dá menos superfície no workflow** (um passo, `tilt ci` sai com o resultado do e2e e mata tudo); **T2 dá mais controle** (passos separados para boot, aquecimento, e2e e logs; cache do `next dev` funcional; `tilt down` explícito).
