import { getVersionMajorMinor, getVersionedDocs } from './versioning';

describe('versioning utilities', () => {
    describe('getVersionMajorMinor', () => {
        it('only returns the major and minor version as a string when the given a valid version', () => {
            expect(getVersionMajorMinor('3.73.x')).toBe('3.73');
            expect(getVersionMajorMinor('3.73.123')).toBe('3.73');
            expect(getVersionMajorMinor('3.73.123.1')).toBe('3.73');
            expect(getVersionMajorMinor('3.73-beta.123')).toBe('3.73');
            expect(getVersionMajorMinor('3.73.123-beta')).toBe('3.73');
            expect(getVersionMajorMinor('3.73')).toBe('3.73');
            expect(getVersionMajorMinor('4.0.0')).toBe('4.0');
        });

        it('returns an empty string when given an invalid version', () => {
            expect(getVersionMajorMinor('3')).toBe('');
            expect(getVersionMajorMinor('a.b')).toBe('');
            expect(getVersionMajorMinor('a.b.c-d')).toBe('');
            expect(getVersionMajorMinor('')).toBe('');
            expect(getVersionMajorMinor('3a.4b')).toBe('');
        });
    });

    describe('getVersionedDocs', () => {
        it('returns the correct url for acs documentation', () => {
            expect(getVersionedDocs('3.73', 'sub-path')).toBe(
                'https://docs.openshift.com/acs/3.73/sub-path'
            );
        });

        it('returns only the major and minor version in the url', () => {
            expect(getVersionedDocs('3.73.123')).toMatch(/.*\/3\.73\//);
        });

        it('the url ends with the given subPath', () => {
            expect(getVersionedDocs('3.73.123', 'sub-path#anchor')).toMatch(/.*\/sub-path#anchor/);
        });

        it('the url ends with the default subpath welcome/index.html when the subpath is not given', () => {
            expect(getVersionedDocs('3.73.123')).toMatch(/.*\/welcome\/index\.html/);
        });
    });
});
