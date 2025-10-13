import React, { useEffect, useMemo, useState } from 'react';
import { Tab, Tabs, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

const NON_STANDARD_TAB = 'Other';

// Extract unique standards from profiles and add 'Other' standard if there are profiles with no standards
function getUniqueStandards(profiles: ComplianceProfileSummary[]): string[] {
    const standards = new Set(
        profiles.flatMap((profile) => profile.standards.map((standard) => standard.shortName))
    );

    const standardsArray = Array.from(standards).sort();

    if (profiles.some((profile) => profile.standards.length === 0)) {
        standardsArray.push(NON_STANDARD_TAB);
    }

    return standardsArray;
}

function getInitialStandard(profiles: ComplianceProfileSummary[], profileName: string): string {
    const profile = profiles.find((profile) => profile.name === profileName);
    if (profile && profile.standards.length > 0) {
        return profile.standards[0].shortName;
    }
    return NON_STANDARD_TAB;
}

function isStandardInProfile(
    standardShortName: string,
    profile: ComplianceProfileSummary
): boolean {
    return (
        profile.standards.some((standard) => standard.shortName === standardShortName) ||
        (standardShortName === NON_STANDARD_TAB && profile.standards.length === 0)
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
        // Currently picks the first standard found since no profile should have multiple standards, however
        // if this changes in the future, we'll want to find all matches and only update selectedStandard if the
        // current selectedStandard doesn't exist in the match
        if (profileName && profiles.some((profile) => profile.name === profileName)) {
            const standardShortName =
                profiles.find((profile) => profile.name === profileName)?.standards[0]?.shortName ||
                NON_STANDARD_TAB;
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
