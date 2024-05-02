import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CoveragesToggleGroup from './CoveragesToggleGroup';

function ProfileChecksPage() {
    const history = useHistory();
    const { profileName } = useParams();

    return (
        <>
            <CoveragesToggleGroup tableView="checks" />
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
            </PageSection>
        </>
    );
}

export default ProfileChecksPage;
