import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';
import { ListComplianceProfileScanStatsResponse } from 'services/ComplianceResultsService';

function CoveragesToggleGroup({
    tableView,
    profileScanStats,
}: {
    tableView: string;
    profileScanStats: ListComplianceProfileScanStatsResponse;
}) {
    const { profileName } = useParams();
    const history = useHistory();

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
