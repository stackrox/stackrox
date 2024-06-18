import React from 'react';
import { useParams } from 'react-router-dom';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { ComplianceProfileScanStats } from 'services/ComplianceResultsStatsService';

type ProfilesToggleGroupProps = {
    profiles: ComplianceProfileScanStats[];
    handleToggleChange: (selectedProfile: string) => void;
};

function ProfilesToggleGroup({ profiles, handleToggleChange }: ProfilesToggleGroupProps) {
    const { profileName: profileNameParam } = useParams();

    return (
        <ToggleGroup aria-label="Toggle for selected profile view">
            {profiles.map(({ profileName }) => (
                <ToggleGroupItem
                    key={profileName}
                    text={profileName}
                    buttonId="compliance-profiles-toggle-group"
                    isSelected={profileNameParam === profileName}
                    onChange={() => handleToggleChange(profileName)}
                />
            ))}
        </ToggleGroup>
    );
}

export default ProfilesToggleGroup;
