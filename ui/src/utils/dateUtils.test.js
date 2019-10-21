/* eslint-disable no-use-before-define */
// system under test (SUT)
import { getLatestDatedItemByKey } from './dateUtils';

describe('dateUtils', () => {
    describe('getLatestDatedItemByKey', () => {
        it('should return null when not passed a field key', () => {
            const deployAlerts = getDatedList();

            const latestItem = getLatestDatedItemByKey(null, deployAlerts);

            expect(latestItem).toEqual(null);
        });

        it('should return null when passed an empty list', () => {
            const deployAlerts = [];

            const latestItem = getLatestDatedItemByKey('time', deployAlerts);

            expect(latestItem).toEqual(null);
        });

        it('should return item with the most recent date in the specified field', () => {
            const deployAlerts = getDatedList();

            const latestItem = getLatestDatedItemByKey('time', deployAlerts);

            expect(latestItem.time).toEqual('2019-10-21T14:49:50.1567707Z');
        });
    });
});

function getDatedList() {
    return [
        {
            id: 'c7ae8fe8-8f7a-4460-95bf-849d8b0238b8',
            firstOccurred: '2019-10-21T14:48:36.3543392Z',
            time: '2019-10-21T14:48:36.1567707Z',
            policy: {
                id: '74cfb824-2e65-46b7-b1b4-ba897e53af1f'
            },
            state: 'ACTIVE',
            processViolation: null
        },
        {
            id: '4422cd23-24c9-48ec-a5f9-c8c159f3602f',
            firstOccurred: '2019-10-19T14:49:36.3543392Z',
            time: '2019-10-21T14:49:36.1567707Z',
            policy: {
                id: '886c3c94-3a6a-4f2b-82fc-d6bf5a310840'
            },
            state: 'ACTIVE',
            processViolation: null
        },
        {
            id: '0b30af1b-7ad3-4da9-8e15-e56d362f3658',
            firstOccurred: '2019-10-20T14:49:36.3543392Z',
            time: '2019-10-20T14:49:36.1567707Z',
            policy: {
                id: '2db9a279-2aec-4618-a85d-7f1bdf4911b1'
            },
            state: 'ACTIVE',
            processViolation: null
        },
        {
            id: 'e0adc514-183d-4ce6-9ef5-6d97a935b8c5',
            firstOccurred: '2019-10-18T14:50:36.3543392Z',
            time: '2019-10-21T14:49:50.1567707Z',
            policy: {
                id: 'f09f8da1-6111-4ca0-8f49-294a76c65115'
            },
            state: 'ACTIVE',
            processViolation: null
        }
    ];
}
