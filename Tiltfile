# -*- mode: Python -*-

if os.getenv('TILT_STACK', '') not in ['', 'minimal']:
    fail('TILT_STACK must be unset or "minimal", got "{}"'.format(os.getenv('TILT_STACK')))

local_resource(
    'root-deps',
    cmd='pnpm install',
    deps=['package.json', 'pnpm-lock.yaml', 'pnpm-workspace.yaml', 'lefthook.yml'],
    labels=['tooling'],
)

include('./backend/Tiltfile')
include('./openapi/Tiltfile')
include('./frontend/Tiltfile')

if os.getenv('TILT_STACK', '') != 'minimal':
    local_resource(
        'grafana-token',
        cmd='make -s grafana.token',
        resource_deps=['grafana'],
        labels=['Monitoring'],
    )
