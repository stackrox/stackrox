import React, { ReactElement } from 'react';

import { violationsBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';

const ViolationNotFoundPage = (): ReactElement => (
    <NotFoundMessage
        title="404: We couldn't find that page"
        message="Violation not found. This violation may have been deleted due to data retention settings."
        actionText="Go to Violations"
        url={violationsBasePath}
    />
);

export default ViolationNotFoundPage;
