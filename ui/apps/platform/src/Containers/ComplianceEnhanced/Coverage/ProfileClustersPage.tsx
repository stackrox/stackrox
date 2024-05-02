import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CoveragesToggleGroup from './CoveragesToggleGroup';

function ProfileClustersPage() {
    const history = useHistory();
    const { profileName } = useParams();

    return (
        <>
            <CoveragesToggleGroup tableView="clusters" />
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

export default ProfileClustersPage;
