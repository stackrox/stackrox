import React, { ReactElement, useContext } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '@patternfly/react-core';

import workflowStateContext from 'Containers/workflowStateContext';
import useFeatureFlags from 'hooks/useFeatureFlags';
import entityTypes from 'constants/entityTypes';

type CVESummaryLinkProps = {
    cve: string;
    id: string;
};

function CVESummaryLink({ cve, id }: CVESummaryLinkProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');
    const entityType = showVMUpdates ? entityTypes.IMAGE_CVE : entityTypes.CVE;

    const workflowState = useContext(workflowStateContext);
    const url = workflowState.pushRelatedEntity(entityType, id).toUrl();

    return (
        <Button variant="link" isInline>
            <Link to={url}>{cve}</Link>
        </Button>
    );
}

export default CVESummaryLink;
