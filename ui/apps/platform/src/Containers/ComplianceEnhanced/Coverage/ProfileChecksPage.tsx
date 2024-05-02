import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';
import { ListComplianceProfileScanStatsResponse } from 'services/ComplianceResultsService';

import CoveragesToggleGroup from './CoveragesToggleGroup';

function ProfileChecksPage({
    profileScanStats,
}: {
    profileScanStats: ListComplianceProfileScanStatsResponse;
}) {
    const history = useHistory();
    const { profileName } = useParams();

    const profileParamExists = profileScanStats.scanStats.some(
        (profile) => profile.profileName === profileName
    );

    if (!profileParamExists) {
        return <div>No results for {profileName}</div>;
    }

    return (
        <>
            <CoveragesToggleGroup tableView="checks" profileScanStats={profileScanStats} />
            <PageSection variant="light">
                <div>ProfileChecksPage</div>
                <Button
                    onClick={() => {
                        history.push(
                            `${complianceEnhancedCoveragePath}/profiles/${profileName}/clusters`
                        );
                    }}
                    variant="primary"
                >
                    Go to all clusters (ProfileClustersPage)
                </Button>
                <Button
                    onClick={() => {
                        history.push(
                            `${complianceEnhancedCoveragePath}/profiles/${profileName}/checks/test`
                        );
                    }}
                    variant="primary"
                >
                    Go to single check (CheckDetailsPage)
                </Button>
            </PageSection>
        </>
    );
}

export default ProfileChecksPage;
