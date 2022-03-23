// system under test (SUT)
import { getLatestDatedItemByKey, addBrandedTimestampToString, getDayList } from './dateUtils';

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

            expect(latestItem).toHaveProperty('time', '2019-10-21T14:49:50.1567707Z');
        });
    });

    describe('addBrandedTimestampToString', () => {
        it('should return string with branding prepended, and current data appended', () => {
            const currentDate = new Date();
            const month = `0${currentDate.getMonth() + 1}`.slice(-2);
            const dayOfMonth = `0${currentDate.getDate()}`.slice(-2);
            const year = currentDate.getFullYear();

            const baseName = `Vulnerability Management CVES Report`;

            const fileName = addBrandedTimestampToString(baseName);

            expect(fileName).toEqual(`StackRox:${baseName}-${month}/${dayOfMonth}/${year}`);
        });
    });

    describe('getDayList', () => {
        it('should return array with one weekly day', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [1];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['Monday']);
        });

        it('should return array with two contiguous weekly days', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [1, 2];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['Monday', 'Tuesday']);
        });

        it('should return array with two non-contiguous weekly days', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [1, 5];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['Monday', 'Friday']);
        });

        it('should return array with three contiguous weekly days', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [2, 3, 4];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['Tuesday', 'Wednesday', 'Thursday']);
        });

        it('should return array with three non-contiguous weekly days', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [1, 5, 0];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['Monday', 'Friday', 'Sunday']);
        });

        it('should return array with all, but one, weekly days', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [2, 3, 4, 5, 6, 0];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual([
                'Tuesday',
                'Wednesday',
                'Thursday',
                'Friday',
                'Saturday',
                'Sunday',
            ]);
        });

        it('should return array with all weekly days', () => {
            const dayListType = 'WEEKLY';
            const daysArray = [1, 2, 3, 4, 5, 6, 0];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual([
                'Monday',
                'Tuesday',
                'Wednesday',
                'Thursday',
                'Friday',
                'Saturday',
                'Sunday',
            ]);
        });

        it('should return array with first monthly day', () => {
            const dayListType = 'MONTHLY';
            const daysArray = [1];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['the first of the month']);
        });

        it('should return array with other monthly days', () => {
            const dayListType = 'MONTHLY';
            const daysArray = [15];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['the middle of the month']);
        });

        it('should return array with both monthly days', () => {
            const dayListType = 'MONTHLY';
            const daysArray = [1, 15];

            const daysList = getDayList(dayListType, daysArray);

            expect(daysList).toEqual(['the first of the month', 'the middle of the month']);
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
                id: '74cfb824-2e65-46b7-b1b4-ba897e53af1f',
            },
            state: 'ACTIVE',
            processViolation: null,
        },
        {
            id: '4422cd23-24c9-48ec-a5f9-c8c159f3602f',
            firstOccurred: '2019-10-19T14:49:36.3543392Z',
            time: '2019-10-21T14:49:36.1567707Z',
            policy: {
                id: '886c3c94-3a6a-4f2b-82fc-d6bf5a310840',
            },
            state: 'ACTIVE',
            processViolation: null,
        },
        {
            id: '0b30af1b-7ad3-4da9-8e15-e56d362f3658',
            firstOccurred: '2019-10-20T14:49:36.3543392Z',
            time: '2019-10-20T14:49:36.1567707Z',
            policy: {
                id: '2db9a279-2aec-4618-a85d-7f1bdf4911b1',
            },
            state: 'ACTIVE',
            processViolation: null,
        },
        {
            id: 'e0adc514-183d-4ce6-9ef5-6d97a935b8c5',
            firstOccurred: '2019-10-18T14:50:36.3543392Z',
            time: '2019-10-21T14:49:50.1567707Z',
            policy: {
                id: 'f09f8da1-6111-4ca0-8f49-294a76c65115',
            },
            state: 'ACTIVE',
            processViolation: null,
        },
    ];
}
