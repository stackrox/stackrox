import React, { ReactElement, useContext } from 'react';
import { Link } from 'react-router-dom';

import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';

type CVESummaryLinkProps = {
    cve: string;
    id: string;
};

function CVESummaryLink({ cve, id }: CVESummaryLinkProps): ReactElement {
    const entityType = entityTypes.IMAGE_CVE;

    const workflowState = useContext(workflowStateContext);
    const url = workflowState.pushRelatedEntity(entityType, id).toUrl();

    return <Link to={url}>{cve}</Link>;
}

export default CVESummaryLink;
