import type { ComplianceBenchmark, ComplianceProfileSummary } from 'services/ComplianceCommon';

import {
    NON_STANDARD_TAB,
    TAILORED_PROFILES_TAB,
    getDefaultProfile,
    getFirstStandardShortName,
    getProfileTab,
    getProfilesByTab,
    getTabsFromProfiles,
} from './ProfilesToggleGroup.utils';

function createComplianceStandard(shortName: string): ComplianceBenchmark {
    return { name: '', version: '', description: '', provider: '', shortName };
}

function createProfile(
    name: string,
    partial: Partial<ComplianceProfileSummary> = {}
): ComplianceProfileSummary {
    return {
        name,
        productType: 'Platform',
        description: '',
        title: '',
        ruleCount: 0,
        profileVersion: '',
        standards: [],
        ...partial,
    };
}

describe('ProfilesToggleGroup.utils', () => {
    describe('getFirstStandardShortName', () => {
        test('returns the first non-empty shortName when several standards exist', () => {
            const profile = createProfile('profile-one', {
                standards: [
                    createComplianceStandard(''),
                    createComplianceStandard('CIS'),
                    createComplianceStandard('BSI'),
                ],
            });
            expect(getFirstStandardShortName(profile)).toBe('CIS');
        });

        test('skips leading empty shortNames', () => {
            const profile = createProfile('profile-one', {
                standards: [
                    createComplianceStandard(''),
                    createComplianceStandard(''),
                    createComplianceStandard('NIST'),
                ],
            });
            expect(getFirstStandardShortName(profile)).toBe('NIST');
        });

        test('returns undefined when all shortNames are empty', () => {
            const profile = createProfile('profile-one', {
                standards: [createComplianceStandard(''), createComplianceStandard('')],
            });
            expect(getFirstStandardShortName(profile)).toBeUndefined();
        });

        test('returns undefined when standards is empty', () => {
            expect(getFirstStandardShortName(createProfile('profile-one'))).toBeUndefined();
        });
    });

    describe('getProfileTab', () => {
        test('TAILORED_PROFILE maps to Tailored Profiles tab only', () => {
            const profile = createProfile('tailored-profile', {
                operatorKind: 'TAILORED_PROFILE',
                standards: [createComplianceStandard('CIS')],
            });
            expect(getProfileTab(profile)).toBe(TAILORED_PROFILES_TAB);
        });

        test('PROFILE with a non-empty standard shortName maps to that tab', () => {
            const profile = createProfile('built-in-profile', {
                operatorKind: 'PROFILE',
                standards: [createComplianceStandard('BSI')],
            });
            expect(getProfileTab(profile)).toBe('BSI');
        });

        test('PROFILE without a named standard maps to Other', () => {
            const profile = createProfile('profile-without-standard', {
                operatorKind: 'PROFILE',
                standards: [],
            });
            expect(getProfileTab(profile)).toBe(NON_STANDARD_TAB);
        });

        test('OPERATOR_KIND_UNSPECIFIED with a standard uses first non-empty shortName', () => {
            const profile = createProfile('legacy-profile', {
                operatorKind: 'OPERATOR_KIND_UNSPECIFIED',
                standards: [createComplianceStandard('PCI')],
            });
            expect(getProfileTab(profile)).toBe('PCI');
        });

        test('OPERATOR_KIND_UNSPECIFIED without a standard maps to Other', () => {
            const profile = createProfile('legacy-profile-empty-standard', {
                operatorKind: 'OPERATOR_KIND_UNSPECIFIED',
                standards: [createComplianceStandard('')],
            });
            expect(getProfileTab(profile)).toBe(NON_STANDARD_TAB);
        });

        test('missing operatorKind behaves like non-tailored (shortName or Other)', () => {
            const withShortName = createProfile('unspecified-with-short-name', {
                standards: [createComplianceStandard('GDPR')],
            });
            expect(getProfileTab(withShortName)).toBe('GDPR');
            expect(getProfileTab(createProfile('unspecified-without-standard'))).toBe(
                NON_STANDARD_TAB
            );
        });
    });

    describe('getTabsFromProfiles', () => {
        test('empty profile list yields no tabs', () => {
            expect(getTabsFromProfiles([])).toEqual([]);
        });

        test('only tailored profiles: Tailored tab, no standard shortName tabs or Other', () => {
            const profiles = [
                createProfile('tailored-one', {
                    operatorKind: 'TAILORED_PROFILE',
                    standards: [createComplianceStandard('CIS')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual([TAILORED_PROFILES_TAB]);
        });

        test('tailored profiles do not create standard shortName tabs from embedded standards', () => {
            const profiles = [
                createProfile('tailored-profile', {
                    operatorKind: 'TAILORED_PROFILE',
                    standards: [createComplianceStandard('CIS')],
                }),
                createProfile('built-in-profile', {
                    operatorKind: 'PROFILE',
                    standards: [createComplianceStandard('CIS')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual(['CIS', TAILORED_PROFILES_TAB]);
        });

        test('PROFILE with standard: standard shortName tab rendered', () => {
            const profiles = [
                createProfile('profile-openshift-cis', {
                    operatorKind: 'PROFILE',
                    standards: [createComplianceStandard('CIS-OCP')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual(['CIS-OCP']);
        });

        test('PROFILE without standard: Other tab only when something maps there', () => {
            const profiles = [
                createProfile('other-only-profile', { operatorKind: 'PROFILE', standards: [] }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual([NON_STANDARD_TAB]);
        });

        test('Other is omitted when no profile maps to Other', () => {
            const profiles = [
                createProfile('profile-pci-only', {
                    standards: [createComplianceStandard('PCI')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual(['PCI']);
            expect(getTabsFromProfiles(profiles)).not.toContain(NON_STANDARD_TAB);
        });

        test('Tailored tab omitted when no tailored profiles', () => {
            const profiles = [
                createProfile('profile-pci-only', {
                    standards: [createComplianceStandard('PCI')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).not.toContain(TAILORED_PROFILES_TAB);
        });

        test('mixed list: sorted standard shortName tabs, then Tailored, then Other when applicable', () => {
            const profiles = [
                createProfile('profile-cis', { standards: [createComplianceStandard('CIS')] }),
                createProfile('tailored-profile', {
                    operatorKind: 'TAILORED_PROFILE',
                    standards: [],
                }),
                createProfile('profile-bsi', { standards: [createComplianceStandard('BSI')] }),
                createProfile('profile-other-bucket', { standards: [] }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual([
                'BSI',
                'CIS',
                TAILORED_PROFILES_TAB,
                NON_STANDARD_TAB,
            ]);
        });

        test('multiple profiles under same standard shortName: single tab key', () => {
            const profiles = [
                createProfile('profile-nist-one', {
                    standards: [createComplianceStandard('NIST')],
                }),
                createProfile('profile-nist-two', {
                    standards: [createComplianceStandard('NIST')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual(['NIST']);
        });

        test('OPERATOR_KIND_UNSPECIFIED with shortName contributes to standard shortName tabs', () => {
            const profiles = [
                createProfile('unspecified-profile', {
                    operatorKind: 'OPERATOR_KIND_UNSPECIFIED',
                    standards: [createComplianceStandard('HIPAA')],
                }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual(['HIPAA']);
        });

        test('only Other profiles: single tab labeled Other', () => {
            const profiles = [
                createProfile('other-one', { standards: [] }),
                createProfile('other-two', { standards: [createComplianceStandard('')] }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual([NON_STANDARD_TAB]);
        });

        test('only profiles with standard shortNames: no Tailored or Other', () => {
            const profiles = [
                createProfile('profile-nist', { standards: [createComplianceStandard('NIST')] }),
                createProfile('profile-cis', { standards: [createComplianceStandard('CIS')] }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual(['CIS', 'NIST']);
        });

        test('only tailored: single Tailored tab', () => {
            const profiles = [
                createProfile('tailored-one', { operatorKind: 'TAILORED_PROFILE' }),
                createProfile('tailored-two', { operatorKind: 'TAILORED_PROFILE' }),
            ];
            expect(getTabsFromProfiles(profiles)).toEqual([TAILORED_PROFILES_TAB]);
        });
    });

    describe('getProfilesByTab', () => {
        test('filters and sorts by name', () => {
            const profiles = [
                createProfile('z-nist', { standards: [createComplianceStandard('NIST')] }),
                createProfile('a-nist', { standards: [createComplianceStandard('NIST')] }),
                createProfile('cis-only', { standards: [createComplianceStandard('CIS')] }),
            ];
            expect(getProfilesByTab(profiles, 'NIST').map((p) => p.name)).toEqual([
                'a-nist',
                'z-nist',
            ]);
        });
    });

    describe('getDefaultProfile', () => {
        test('empty profiles yields undefined', () => {
            expect(getDefaultProfile([])).toBeUndefined();
        });

        test('uses first getTabsFromProfiles tab and first profile by name there', () => {
            const profiles = [
                createProfile('z-cis', { standards: [createComplianceStandard('CIS')] }),
                createProfile('a-cis', { standards: [createComplianceStandard('CIS')] }),
                createProfile('b-nist', { standards: [createComplianceStandard('NIST')] }),
            ];
            expect(getDefaultProfile(profiles)?.name).toBe('a-cis');
        });
    });
});
