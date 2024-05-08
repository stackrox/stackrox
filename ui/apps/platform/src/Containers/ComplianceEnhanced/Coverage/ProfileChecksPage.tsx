import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CoveragesToggleGroup from './CoveragesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';

function ProfileChecksPage() {
    const history = useHistory();
    const { profileName } = useParams();

    return (
        <>
            <CoveragesPageHeader />
            <PageSection>
                <CoveragesToggleGroup tableView="checks" />
            </PageSection>
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
