import React, { ReactElement } from 'react';

import { violationsPFBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';

const ViolationNotFoundPage = (): ReactElement => (
    <NotFoundMessage
        title="404: We couldn't find that page"
        message="Violation not found. This violation may have been deleted due to data retention settings."
        actionText="Go to Violations"
        url={violationsPFBasePath}
    />
);

export default ViolationNotFoundPage;
