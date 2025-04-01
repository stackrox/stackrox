import React from 'react';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

function ComplianceNotFoundPage() {
    return (
        <PageSection variant="light">
            <PageTitle title="Compliance - Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

export default ComplianceNotFoundPage;
