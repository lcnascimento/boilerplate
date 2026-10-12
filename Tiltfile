# -*- mode: Python -*-

# TILT_STACK picks the stack: unset for everything, `minimal` for the services,
# the web app and their deps only (no observability, Scalar nor web-e2e-ui).
# Each area's Tiltfile reads it and skips registering what the stack leaves out.
if os.getenv('TILT_STACK', '') not in ['', 'minimal']:
    fail('TILT_STACK must be unset or "minimal", got "{}"'.format(os.getenv('TILT_STACK')))

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
