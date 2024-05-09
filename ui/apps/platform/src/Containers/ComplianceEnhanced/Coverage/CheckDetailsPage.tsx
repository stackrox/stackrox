import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Button, PageSection } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

function CheckDetailsPage() {
    const history = useHistory();
    const { profileName } = useParams();
    return (
        <>
            <PageSection variant="light">
                <div>CheckDetailsPage</div>
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

export default CheckDetailsPage;
