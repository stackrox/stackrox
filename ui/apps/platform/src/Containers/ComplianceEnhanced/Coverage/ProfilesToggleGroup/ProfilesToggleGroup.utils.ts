import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

export const NON_STANDARD_TAB = 'Other';
export const TAILORED_PROFILES_TAB = 'Tailored Profiles';

// Compliance profiles currently have a single standard, but the API returns an array.
// Return the first non-empty shortName in case multiple standards are ever present.
export function getFirstStandardShortName(profile: ComplianceProfileSummary): string | undefined {
    return profile.standards.find((standard) => standard.shortName.length > 0)?.shortName;
}

// Tailored profiles use their own tab; all other kinds use the first standard shortName or Other.
export function getProfileTab(profile: ComplianceProfileSummary): string {
    if (profile.operatorKind === 'TAILORED_PROFILE') {
        return TAILORED_PROFILES_TAB;
    }

    return getFirstStandardShortName(profile) ?? NON_STANDARD_TAB;
}

// Tab keys: unique standard short names (non-tailored with a shortName), Tailored if any, Other if any non-tailored lacks a shortName.
export function getStandardTabs(profiles: ComplianceProfileSummary[]): string[] {
    const uniqueStandardShortNames = new Set<string>();
    let isTailoredTabApplicable = false;
    let isOtherTabApplicable = false;

    profiles.forEach((profile) => {
        if (profile.operatorKind === 'TAILORED_PROFILE') {
            isTailoredTabApplicable = true;
            return;
        }

        const standardShortName = getFirstStandardShortName(profile);
        if (standardShortName) {
            uniqueStandardShortNames.add(standardShortName);
        } else {
            isOtherTabApplicable = true;
        }
    });

    return [
        ...Array.from(uniqueStandardShortNames).sort(),
        ...(isTailoredTabApplicable ? [TAILORED_PROFILES_TAB] : []),
        ...(isOtherTabApplicable ? [NON_STANDARD_TAB] : []),
    ];
}
