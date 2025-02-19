import { getDistroLinkText } from './textUtils';

describe('getDistroLinkText', () => {
    const testCases = [
        { link: 'https://www.cve.org/CVE-2021-1234', expected: 'View at cve.org' },
        {
            link: 'https://security.alpinelinux.org/vuln/CVE-2021-1234',
            expected: 'View in Alpine CVE database',
        },
        {
            link: 'https://alas.aws.amazon.com/ALAS-2021-1234',
            expected: 'View in Amazon CVE database',
        },
        {
            link: 'https://security-tracker.debian.org/tracker/CVE-2021-1234',
            expected: 'View in Debian CVE database',
        },
        {
            link: 'https://access.redhat.com/security/cve/CVE-2021-1234',
            expected: 'View in Red Hat CVE database',
        },
        {
            link: 'https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-1234',
            expected: 'View in MITRE CVE database',
        },
        {
            link: 'https://linux.oracle.com/cve/CVE-2021-1234',
            expected: 'View in Oracle CVE database',
        },
        {
            link: 'https://ubuntu.com/security/CVE-2021-1234',
            expected: 'View in Ubuntu CVE database',
        },
        { link: 'https://osv.dev/CVE-2021-1234', expected: 'View in OSV CVE database' },
    ];

    testCases.forEach(({ link, expected }) => {
        test(`returns '${expected}' for link: ${link}`, () => {
            expect(getDistroLinkText({ link })).toBe(expected);
        });
    });

    test('returns generic message for unknown domain', () => {
        expect(getDistroLinkText({ link: 'https://example.com/cve/1234' })).toBe(
            'View additional information at example.com'
        );
    });

    test('returns generic message when given an invalid URL', () => {
        expect(getDistroLinkText({ link: 'invalid-url' })).toBe('View additional information');
    });
});
