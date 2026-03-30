import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

// A standard short name, 'Tailored Profiles', or 'Other'.
export type ProfileTab = string;

export const NON_STANDARD_TAB: ProfileTab = 'Other';
export const TAILORED_PROFILES_TAB: ProfileTab = 'Tailored Profiles';

// Compliance profiles currently have a single standard, but the API returns an array.
// Return the first non-empty shortName in case multiple standards are ever present.
export function getFirstStandardShortName(profile: ComplianceProfileSummary): string | undefined {
    return profile.standards.find((standard) => standard.shortName.length > 0)?.shortName;
}

// Tailored profiles use their own tab; all other kinds use the first standard shortName or Other.
export function getProfileTab(profile: ComplianceProfileSummary): ProfileTab {
    if (profile.operatorKind === 'TAILORED_PROFILE') {
        return TAILORED_PROFILES_TAB;
    }

    return getFirstStandardShortName(profile) ?? NON_STANDARD_TAB;
}

// Tab keys: unique standard short names (non-tailored with a shortName), Tailored if any, Other if any non-tailored lacks a shortName.
export function getTabsFromProfiles(profiles: ComplianceProfileSummary[]): ProfileTab[] {
    const uniqueStandardShortNames = new Set<ProfileTab>();
    let hasTailoredProfilesTab = false;
    let hasOtherTab = false;

    profiles.forEach((profile) => {
        if (profile.operatorKind === 'TAILORED_PROFILE') {
            hasTailoredProfilesTab = true;
            return;
        }

        const standardShortName = getFirstStandardShortName(profile);
        if (standardShortName) {
            uniqueStandardShortNames.add(standardShortName);
        } else {
            hasOtherTab = true;
        }
    });

    return [
        ...Array.from(uniqueStandardShortNames).sort(),
        ...(hasTailoredProfilesTab ? [TAILORED_PROFILES_TAB] : []),
        ...(hasOtherTab ? [NON_STANDARD_TAB] : []),
    ];
}

// Profiles for a given tab, sorted by name (same order as the toggle group).
export function getProfilesByTab(
    profiles: ComplianceProfileSummary[],
    tab: ProfileTab
): ComplianceProfileSummary[] {
    return profiles
        .filter((profile) => getProfileTab(profile) === tab)
        .sort((a, b) => a.name.localeCompare(b.name));
}

// Gets the first profile by name from the first tab. Used as fallback when opening coverage or URL is stale.
export function getDefaultProfile(
    profiles: ComplianceProfileSummary[]
): ComplianceProfileSummary | undefined {
    const tab = getTabsFromProfiles(profiles)[0];
    return tab ? getProfilesByTab(profiles, tab)[0] : undefined;
}
