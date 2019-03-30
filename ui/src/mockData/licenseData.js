import { LICENSE_STATUS } from 'reducers/license';

export const licenseType = 'Managed Service Provider';

export const licenses = [
    {
        license: {
            supportContact: {
                phone: '1 (650) 489-6769',
                email: 'support@stackrox.com',
                url: '',
                name: ''
            },
            metadata: {
                id: '1234-5678-9123-4567',
                issueDate: '2019-01-01T12:00:00Z',
                licensedForId: '48ejd4jk9d3m',
                licensedForName: 'Alan Roy'
            },
            restrictions: {
                notValidBefore: '2018-12-31T12:00:00Z',
                notValidAfter: '2019-03-28T12:00:00Z',
                allowOffline: true,
                maxNodes: '500',
                buildFlavors: '',
                deploymentEnvironments: ''
            }
        },
        status: LICENSE_STATUS.VALID,
        statusReason: '',
        active: true
    }
];
