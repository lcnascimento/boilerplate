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
