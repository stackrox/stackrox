import React, { useContext } from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

import { ComplianceProfilesContext } from './ComplianceProfilesProvider';

function CoveragesToggleGroup({ tableView = 'checks' }: { tableView: string }) {
    const { profileName } = useParams();
    const history = useHistory();

    const context = useContext(ComplianceProfilesContext);
    if (!context) {
        return null;
    }
    const { profileScanStats } = context;

    const handleToggleChange = (selectedProfile) => {
        history.push(`${complianceEnhancedCoveragePath}/profiles/${selectedProfile}/${tableView}`);
    };
    return (
        <ToggleGroup aria-label="Toggle for selected profile view">
            {profileScanStats &&
                profileScanStats.scanStats.map((profile) => (
                    <ToggleGroupItem
                        key={profile.profileName}
                        text={profile.profileName}
                        buttonId="compliance-profiles-toggle-group"
                        isSelected={profileName === profile.profileName}
                        onChange={() => handleToggleChange(profile.profileName)}
                    />
                ))}
        </ToggleGroup>
    );
}

export default CoveragesToggleGroup;
