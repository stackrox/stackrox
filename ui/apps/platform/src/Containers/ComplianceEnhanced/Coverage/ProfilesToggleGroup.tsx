import { useEffect, useMemo, useState } from 'react';
import { Tab, Tabs, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

const NON_STANDARD_TAB = 'Other';
const TAILORED_PROFILES_TAB = 'Tailored Profiles';

// Check if a profile is a tailored profile
function isTailoredProfile(profile: ComplianceProfileSummary): boolean {
    return profile.tailoredDetails !== undefined && profile.tailoredDetails !== null;
}

// Extract unique standards from profiles and add special tabs for tailored profiles and profiles with no standards
function getUniqueStandards(profiles: ComplianceProfileSummary[]): string[] {
    // Get standards only from non-tailored profiles, filtering out empty shortNames
    const standards = new Set(
        profiles
            .filter((profile) => !isTailoredProfile(profile))
            .flatMap((profile) =>
                profile.standards
                    .map((standard) => standard.shortName)
                    .filter((shortName) => shortName && shortName.trim() !== '')
            )
    );

    const standardsArray = Array.from(standards).sort();

    // Add "Tailored Profiles" tab if there are any tailored profiles
    if (profiles.some(isTailoredProfile)) {
        standardsArray.push(TAILORED_PROFILES_TAB);
    }

    // Add "Other" tab if there are non-tailored profiles with no valid standards
    const hasProfilesWithNoValidStandards = profiles.some(
        (profile) =>
            !isTailoredProfile(profile) &&
            profile.standards.every((s) => !s.shortName || s.shortName.trim() === '')
    );
    if (hasProfilesWithNoValidStandards) {
        standardsArray.push(NON_STANDARD_TAB);
    }

    return standardsArray;
}

function getInitialStandard(profiles: ComplianceProfileSummary[], profileName: string): string {
    const profile = profiles.find((profile) => profile.name === profileName);
    if (profile) {
        // Tailored profiles go to the Tailored Profiles tab
        if (isTailoredProfile(profile)) {
            return TAILORED_PROFILES_TAB;
        }
        // Non-tailored profiles with standards go to their first standard tab
        if (profile.standards.length > 0) {
            return profile.standards[0].shortName;
        }
    }
    return NON_STANDARD_TAB;
}

function hasValidStandards(profile: ComplianceProfileSummary): boolean {
    return profile.standards.some((s) => s.shortName && s.shortName.trim() !== '');
}

function isStandardInProfile(
    standardShortName: string,
    profile: ComplianceProfileSummary
): boolean {
    // Tailored profiles only appear in the Tailored Profiles tab
    if (isTailoredProfile(profile)) {
        return standardShortName === TAILORED_PROFILES_TAB;
    }
    // Non-tailored profiles with no valid standards go to Other tab
    if (standardShortName === NON_STANDARD_TAB) {
        return !hasValidStandards(profile);
    }
    // Non-tailored profiles appear in their matching standard tabs
    return profile.standards.some(
        (standard) => standard.shortName && standard.shortName === standardShortName
    );
}

const tabContentId = 'profiles-toggle-group';

type ProfilesToggleGroupProps = {
    profileName: string;
    profiles: ComplianceProfileSummary[];
    handleToggleChange: (selectedProfile: string) => void;
};

function ProfilesToggleGroup({
    profileName,
    profiles,
    handleToggleChange,
}: ProfilesToggleGroupProps) {
    const uniqueStandards = useMemo(() => getUniqueStandards(profiles), [profiles]);
    const initialStandard = useMemo(
        () => getInitialStandard(profiles, profileName),
        [profileName, profiles]
    );

    const [selectedStandard, setSelectedStandard] = useState(initialStandard);

    useEffect(() => {
        // Sets the selected standard based on the profile name in the URL.
        // Tailored profiles go to the Tailored Profiles tab, otherwise use the first standard or Other tab
        if (profileName && profiles.some((profile) => profile.name === profileName)) {
            const currentProfile = profiles.find((profile) => profile.name === profileName);
            let standardShortName: string;
            if (currentProfile && isTailoredProfile(currentProfile)) {
                standardShortName = TAILORED_PROFILES_TAB;
            } else {
                standardShortName = currentProfile?.standards[0]?.shortName || NON_STANDARD_TAB;
            }
            setSelectedStandard(standardShortName);
        } else {
            if (profiles[0]?.name) {
                // useful when scan schedule filter changes and current profile is not in the list
                handleToggleChange(profiles[0].name);
            }
        }
    }, [profileName, profiles, handleToggleChange]);

    function handleStandardSelection(standardShortName) {
        setSelectedStandard(standardShortName);
        const firstProfileInStandard = profiles.find((profile) =>
            isStandardInProfile(standardShortName, profile)
        );
        if (firstProfileInStandard) {
            handleToggleChange(firstProfileInStandard.name);
        }
    }

    const filteredProfiles: ComplianceProfileSummary[] = useMemo(() => {
        return profiles.filter((profile) => isStandardInProfile(selectedStandard, profile));
    }, [profiles, selectedStandard]);

    return (
        <>
            <Tabs
                activeKey={selectedStandard}
                onSelect={(_e, key) => {
                    handleStandardSelection(key);
                }}
                isBox
            >
                {Array.from(uniqueStandards).map((standardShortName) => (
                    <Tab
                        key={standardShortName}
                        eventKey={standardShortName}
                        title={standardShortName}
                        tabContentId={tabContentId}
                    />
                ))}
            </Tabs>
            <ToggleGroup
                id={tabContentId}
                aria-label="Toggle for selected profile view"
                className="pf-v5-u-background-color-100 pf-v5-u-p-md"
            >
                {filteredProfiles.map(({ name }) => (
                    <ToggleGroupItem
                        key={name}
                        text={name}
                        buttonId="compliance-profiles-toggle-group"
                        isSelected={profileName === name}
                        onChange={() => handleToggleChange(name)}
                    />
                ))}
            </ToggleGroup>
        </>
    );
}

export default ProfilesToggleGroup;
