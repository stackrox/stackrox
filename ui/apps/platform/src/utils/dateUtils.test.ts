// system under test (SUT)
import { addBrandedTimestampToString } from './dateUtils';

describe('dateUtils', () => {
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
});
