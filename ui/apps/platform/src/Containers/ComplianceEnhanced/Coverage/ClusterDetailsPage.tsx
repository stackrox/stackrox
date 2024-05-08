import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

function ClusterDetailsPage() {
    const history = useHistory();
    const { profileName } = useParams();
    return (
        <>
            <PageSection variant="light">
                <div>ClusterDetailsPage</div>
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

export default ClusterDetailsPage;
