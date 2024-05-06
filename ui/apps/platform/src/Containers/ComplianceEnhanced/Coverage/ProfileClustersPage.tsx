import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CoveragesToggleGroup from './CoveragesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';

function ProfileClustersPage() {
    const history = useHistory();
    const { profileName } = useParams();

    return (
        <>
            <CoveragesPageHeader />
            <PageSection>
                <CoveragesToggleGroup tableView="clusters" />
            </PageSection>
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
                <Button
                    onClick={() => {
                        history.push(
                            `${complianceEnhancedCoveragePath}/profiles/${profileName}/clusters/test`
                        );
                    }}
                    variant="primary"
                >
                    Go to single cluster (ClusterDetailsPage)
                </Button>
            </PageSection>
        </>
    );
}

export default ProfileClustersPage;
