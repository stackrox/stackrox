import React, { ReactElement, useContext } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '@patternfly/react-core';

import workflowStateContext from 'Containers/workflowStateContext';

type CVESummaryLinkProps = {
    cve: string;
};

function CVESummaryLink({ cve }: CVESummaryLinkProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    const url = workflowState.pushRelatedEntity('CVE', cve).toUrl();

    return (
        <Button variant="link" isInline>
            <Link to={url}>{cve}</Link>
        </Button>
    );
}

export default CVESummaryLink;
