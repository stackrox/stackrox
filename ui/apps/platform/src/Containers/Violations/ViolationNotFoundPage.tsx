import React, { ReactElement } from 'react';

import { violationsBasePath } from 'routePaths';
import useFilteredWorkflowViewURLState, {
    filteredWorkflowViewKey,
} from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import NotFoundMessage from 'Components/NotFoundMessage';

const ViolationNotFoundPage = (): ReactElement => {
    const { filteredWorkflowView } = useFilteredWorkflowViewURLState('Full view');

    return (
        <NotFoundMessage
            title="404: We couldn't find that page"
            message="Violation not found. This violation may have been deleted due to data retention settings."
            actionText="Go to Violations"
            url={`${violationsBasePath}?${filteredWorkflowViewKey}=${filteredWorkflowView}`}
        />
    );
};

export default ViolationNotFoundPage;
