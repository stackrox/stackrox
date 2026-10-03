const { describe, it } = require('node:test');
const assert = require('node:assert/strict');

const { getPublicEnv } = require('./getPublicEnv');

describe('getPublicEnv', () => {
    it('copies feature flags and orchestrator flavor, and drops secrets', () => {
        assert.deepEqual(
            getPublicEnv({
                ROX_SCANNER_V4: true,
                ROX_OTHER_FLAG: false,
                ORCHESTRATOR_FLAVOR: 'k8s',
                ROX_AUTH_TOKEN: 'secret-token',
                OPENSHIFT_CONSOLE_USERNAME: 'kubeadmin',
                OPENSHIFT_CONSOLE_PASSWORD: 'secret-password',
            }),
            {
                ROX_SCANNER_V4: true,
                ROX_OTHER_FLAG: false,
                ORCHESTRATOR_FLAVOR: 'k8s',
            }
        );
    });

    it('returns an empty object when env is empty', () => {
        assert.deepEqual(getPublicEnv({}), {});
    });
});
