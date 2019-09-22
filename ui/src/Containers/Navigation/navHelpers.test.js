import { knownBackendFlags } from 'utils/featureFlags';

// system under test
import { filterLinksByFeatureFlag } from './navHelpers';

const mockFeatureFlags = [
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

describe('nav helpers', () => {
    describe('filterLinksByFeatureFlag', () => {
        it('should not filter a menu with no flags', () => {
            const menuWithoutFlags = [
                {
                    text: 'Dashboard',
                    to: '/main/dashboard'
                },
                {
                    text: 'Network',
                    to: '/main/network'
                },
                {
                    text: 'Violations',
                    to: '/main/violations'
                }
            ];

            const filtersLinks = filterLinksByFeatureFlag(mockFeatureFlags, menuWithoutFlags);

            expect(filtersLinks.length).toEqual(menuWithoutFlags.length);
        });

        it('should filter a menu with a flag that is NOT turned on', () => {
            const menuWithFlag = [
                {
                    text: 'Compliance',
                    to: '/main/compliance'
                },
                {
                    text: 'Config Management',
                    to: '/main/configmanagement',
                    featureFlag: knownBackendFlags.ROX_CONFIG_MGMT_UI
                },
                {
                    text: 'Risk',
                    to: '/main/risk'
                }
            ];

            const filtersLinks = filterLinksByFeatureFlag(mockFeatureFlags, menuWithFlag);

            expect(filtersLinks.length).toEqual(menuWithFlag.length - 1);
            expect(filtersLinks[1]).toBe(menuWithFlag[2]);
        });

        it('should pass through a menu with a flag that is turned on', () => {
            const menuWithFlag = [
                {
                    text: 'clusters',
                    to: '/main/clusters',
                    featureFlag: knownBackendFlags.ROX_SENSOR_AUTOUPGRADE
                },
                {
                    text: 'Config Management',
                    to: '/main/configmanagement'
                },
                {
                    text: 'Risk',
                    to: '/main/risk'
                }
            ];

            const filtersLinks = filterLinksByFeatureFlag(mockFeatureFlags, menuWithFlag);

            expect(filtersLinks.length).toEqual(menuWithFlag.length);
            expect(filtersLinks[0]).toBe(menuWithFlag[0]);
        });
    });
});
