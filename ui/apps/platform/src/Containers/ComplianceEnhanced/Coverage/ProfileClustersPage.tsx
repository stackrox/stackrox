import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';
import { ListComplianceProfileScanStatsResponse } from 'services/ComplianceResultsService';

import CoveragesToggleGroup from './CoveragesToggleGroup';

function ProfileClustersPage({
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
        return <div>Profile {profileName} does have results</div>;
    }

    return (
        <>
            <CoveragesToggleGroup tableView="clusters" profileScanStats={profileScanStats} />
            <PageSection variant="light">
                <div>ProfileClustersPage</div>
                <Button
                    onClick={() => {
                        history.push(
                            `${complianceEnhancedCoveragePath}/profiles/${profileName}/checks`
                        );
                    }}
                    variant="primary"
                >
                    Go to all checks (ProfileChecksPage)
                </Button>
            </PageSection>
        </>
    );
}

export default ProfileClustersPage;
