import { isBackendFeatureFlagEnabled } from './featureFlags';

const backendFeatureFlags = [
    {
        name: 'Enable Config Mgmt UI',
        envVar: 'ROX_CONFIG_MGMT_UI',
        enabled: false
    },
    {
        name: 'Enable Sensor Autoupgrades',
        envVar: 'ROX_SENSOR_AUTOUPGRADE',
        enabled: true
    }
];

describe('featureFlags utils', () => {
    describe('isBackendFeatureFlagEnabled', () => {
        // Note: we have to do the beforeEach/afterEach, in order to test the env var in one test
        const OLD_ENV = process.env;

        beforeEach(() => {
            jest.resetModules(); // this is important - it clears the cache
            process.env = { ...OLD_ENV };
            delete process.env.NODE_ENV;
        });

        afterEach(() => {
            process.env = OLD_ENV;
        });

        it('should return default value if there is no matching flag in the list of known flags', () => {
            const flagToFind = 'ROX_BADGER_DB';
            const defaultVal = false;

            const isEnabled = isBackendFeatureFlagEnabled(
                backendFeatureFlags,
                flagToFind,
                defaultVal
            );

            expect(isEnabled).toEqual(defaultVal);
        });

        it('should throw, in the dev environment, if there is no matching flag in the list of known flags', () => {
            const flagToFind = 'ROX_BADGER_DB';
            const defaultVal = false;
            process.env.NODE_ENV = 'development';

            expect(() => {
                isBackendFeatureFlagEnabled(backendFeatureFlags, flagToFind, defaultVal);
            }).toThrow();
        });

        it('should return the current value of the given flag when matched and disabled', () => {
            const flagToFind = 'ROX_CONFIG_MGMT_UI'; // backendFeatureFlags[0].name
            const defaultVal = false;

            const isEnabled = isBackendFeatureFlagEnabled(
                backendFeatureFlags,
                flagToFind,
                defaultVal
            );

            expect(isEnabled).toEqual(backendFeatureFlags[0].enabled);
        });

        it('should return the current value of the given flag when matched and disabled', () => {
            const flagToFind = 'ROX_SENSOR_AUTOUPGRADE'; // backendFeatureFlags[1].name
            const defaultVal = false;

            const isEnabled = isBackendFeatureFlagEnabled(
                backendFeatureFlags,
                flagToFind,
                defaultVal
            );

            expect(isEnabled).toEqual(backendFeatureFlags[1].enabled);
        });
    });
});
