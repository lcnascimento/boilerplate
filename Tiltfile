# -*- mode: Python -*-

if os.getenv('TILT_STACK', '') not in ['', 'minimal']:
    fail('TILT_STACK must be unset or "minimal", got "{}"'.format(os.getenv('TILT_STACK')))

local_resource(
    'root-deps',
    cmd='pnpm install',
    deps=['package.json', 'pnpm-lock.yaml', 'pnpm-workspace.yaml', 'lefthook.yml'],
    labels=['tooling'],
)

def test_resource(name, target, resource_deps=[]):
    local_resource(
        name,
        cmd='make -s {}'.format(target),
        resource_deps=resource_deps,
        auto_init=False,
        trigger_mode=TRIGGER_MODE_MANUAL,
        labels=['testing'],
    )

test_resource('test-backend-unit', 'test.backend.unit')
test_resource('test-backend-component', 'test.backend.component', ['postgres'])
test_resource('test-frontend-unit', 'test.frontend.unit', ['frontend-deps'])
test_resource('test-frontend-component', 'test.frontend.component', ['frontend-deps'])
test_resource('test-frontend-e2e', 'test.frontend.e2e', [
    'web',
    # example:begin
    'notes-api-frontend',
    # example:end
])
test_resource('test', 'test.unit test.component', ['postgres', 'frontend-deps'])

include('./backend/Tiltfile')
include('./openapi/Tiltfile')
include('./frontend/Tiltfile')
