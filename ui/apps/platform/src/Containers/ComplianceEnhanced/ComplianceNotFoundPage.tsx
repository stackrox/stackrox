import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

function ComplianceNotFoundPage() {
    return (
        <PageSection hasBodyWrapper={false}>
            <PageTitle title="Compliance - Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

export default ComplianceNotFoundPage;
