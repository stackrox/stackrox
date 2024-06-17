import React, { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Tab, Tabs, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { ComplianceProfileSummary } from 'services/ComplianceCommon';

const NON_STANDARD_TAB = 'Other';

// Extract unique standards from profiles and add 'Other' standard if there are profiles with no standards
function getUniqueStandards(profiles: ComplianceProfileSummary[]): string[] {
    const standards = new Set(
        profiles.flatMap((profile) => profile.standards.map((standard) => standard.shortName))
    );
    if (profiles.some((profile) => profile.standards.length === 0)) {
        standards.add(NON_STANDARD_TAB);
    }
    return Array.from(standards);
}

function getInitialStandard(
    profiles: ComplianceProfileSummary[],
    profileNameParam: string
): string {
    const profile = profiles.find((profile) => profile.name === profileNameParam);
    if (profile && profile.standards.length > 0) {
        return profile.standards[0].shortName;
    }
    return NON_STANDARD_TAB;
}

function isStandardInProfile(standardShortName: string, profile: ComplianceProfileSummary) {
    return (
        profile.standards.some((standard) => standard.shortName === standardShortName) ||
        (standardShortName === 'Other' && profile.standards.length === 0)
    );
}

type ProfilesToggleGroupProps = {
    profiles: ComplianceProfileSummary[];
    handleToggleChange: (selectedProfile: string) => void;
};

function ProfilesToggleGroup({ profiles, handleToggleChange }: ProfilesToggleGroupProps) {
    const { profileName: profileNameParam } = useParams();

    const uniqueStandards = useMemo(() => getUniqueStandards(profiles), [profiles]);
    const initialStandard = useMemo(
        () => getInitialStandard(profiles, profileNameParam),
        [profileNameParam, profiles]
    );

    const [selectedStandard, setSelectedStandard] = useState(initialStandard);

    useEffect(() => {
        // Sets the selected standard based on the profile name in the URL.
        // Currently picks the first standard found since no profile should have multiple standards, however
        // if this changes in the future, we'll want to find all matches and only update selectedStandard if the
        // current selectedStandard doesn't exist in the match
        if (profileNameParam) {
            const standardShortName =
                profiles.find((profile) => profile.name === profileNameParam)?.standards[0]
                    ?.shortName || NON_STANDARD_TAB;
            setSelectedStandard(standardShortName);
        }
    }, [profileNameParam, profiles]);

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
                    />
                ))}
            </Tabs>
            <ToggleGroup
                aria-label="Toggle for selected profile view"
                className="pf-v5-u-background-color-100 pf-v5-u-p-md"
            >
                {filteredProfiles.map(({ name }) => (
                    <ToggleGroupItem
                        key={name}
                        text={name}
                        buttonId="compliance-profiles-toggle-group"
                        isSelected={profileNameParam === name}
                        onChange={() => handleToggleChange(name)}
                    />
                ))}
            </ToggleGroup>
        </>
    );
}

export default ProfilesToggleGroup;
