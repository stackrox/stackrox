import React from 'react';
import { generatePath, useHistory, useParams } from 'react-router-dom';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { ComplianceProfileScanStats } from 'services/ComplianceResultsStatsService';

type ProfilesToggleGroupProps = {
    profiles: ComplianceProfileScanStats[];
    route: string;
};

function ProfilesToggleGroup({ profiles, route }: ProfilesToggleGroupProps) {
    const { profileName: profileNameParam } = useParams();
    const history = useHistory();

    const handleToggleChange = (selectedProfile) => {
        const path = generatePath(route, { profileName: selectedProfile });
        history.push(path);
    };

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
