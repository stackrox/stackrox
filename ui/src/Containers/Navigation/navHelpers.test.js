// system under test
import { filterLinksByFeatureFlag } from './navHelpers';

const mockFeatureFlags = [
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
                    text: 'Config Management',
                    to: '/main/configmanagement'
                },
                {
                    text: 'Violations',
                    to: '/main/violations'
                }
            ];

            const filtersLinks = filterLinksByFeatureFlag(mockFeatureFlags, menuWithoutFlags);

            expect(filtersLinks.length).toEqual(menuWithoutFlags.length);
        });
    });
});
