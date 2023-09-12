import { isDefaultEmailSubject } from './emailTemplateFormUtils';

describe('emailTemplateFormUtils', () => {
    describe('isDefaultEmailSubject', () => {
        it('should be a default email subject', () => {
            expect(
                isDefaultEmailSubject('RHACS Workload CVE Report for sc-test-1; Scope: scope-1')
            ).toEqual(true);

            expect(
                isDefaultEmailSubject('RHACS Workload CVE Report for sc test 1; Scope: scope 1')
            ).toEqual(true);
        });

        it('should not be a default email subject', () => {
            const emailSubject = 'RHACS Workload CVE Report for sc-test-1 / Scope: scope-1';

            expect(isDefaultEmailSubject(emailSubject)).toEqual(false);
        });
    });
});
