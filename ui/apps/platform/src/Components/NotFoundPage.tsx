import type { ReactElement } from 'react';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

function NotFoundPage(): ReactElement {
    return (
        <PageSection hasBodyWrapper={false}>
            <PageTitle title="Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

export default NotFoundPage;
