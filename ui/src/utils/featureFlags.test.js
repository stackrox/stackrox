import { isBackendFeatureFlagEnabled } from './featureFlags';

const backendFeatureFlags = [
    {
        name: 'Enable Config Mgmt UI',
        envVar: 'ROX_CONFIG_MGMT_UI',
        enabled: false,
    },
    {
        name: 'Enable Sensor Autoupgrades',
        envVar: 'ROX_SENSOR_AUTOUPGRADE',
        enabled: true,
    },
];

describe('featureFlags utils', () => {
    describe('isBackendFeatureFlagEnabled', () => {
        // Note: we have to do the beforeEach/afterEach, in order to test the env var in one test
        const OLD_ENV = process.env;

        let spy; // for checking the console.warn call

        beforeEach(() => {
            jest.resetModules(); // this is important - it clears the cache
            process.env = { ...OLD_ENV };
            delete process.env.NODE_ENV;

            spy = jest.spyOn(console, 'warn').mockImplementation();
        });

        afterEach(() => {
            process.env = OLD_ENV;

            spy.mockRestore();
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

        it('should log warning, in the dev environment, if there is no matching flag in the list of known flags', () => {
            jest.spyOn(global.console, 'warn');

            const flagToFind = 'ROX_BADGER_DB';
            const defaultVal = false;
            process.env.NODE_ENV = 'development';

            // eslint-disable-next-line no-unused-vars
            const isEnabled = isBackendFeatureFlagEnabled(
                backendFeatureFlags,
                flagToFind,
                defaultVal
            );

            expect(spy).toHaveBeenCalledTimes(1);
            expect(spy).toHaveBeenLastCalledWith(
                `EnvVar ${flagToFind} not found in the backend list, possibly stale?`
            );
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

        it('should return the current value of the given flag when matched and enabled', () => {
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
