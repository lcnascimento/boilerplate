# -*- mode: Python -*-

# Root tooling: keeps root devDependencies and git hooks (lefthook) installed.
# It also gives Tilt one resource, so the Tiltfile loads before any area exists.
local_resource(
    'root-deps',
    cmd='pnpm install',
    deps=['package.json', 'pnpm-lock.yaml', 'pnpm-workspace.yaml', 'lefthook.yml'],
    labels=['tooling'],
)

# Each area adds its own include below when it is created:
#   include('./backend/Tiltfile')
#   include('./openapi/Tiltfile')
#   include('./frontend/Tiltfile')
include('./backend/Tiltfile')
include('./openapi/Tiltfile')
include('./frontend/Tiltfile')

# spike/boi-37: with E2E_STACK set, only what the e2e suite needs (no
# observability, no Scalar). web-e2e exists only under `tilt ci` (T1).
if os.getenv('E2E_STACK'):
    enabled = ['frontend-deps', 'web', 'notes-api-frontend']
    if config.tilt_subcommand == 'ci':
        enabled.append('web-e2e')
    config.set_enabled_resources(enabled)
