import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';
import entityLabels from 'messages/entity';

import workflowStateContext from 'Containers/workflowStateContext';
import { generateURLTo } from 'modules/URLReadWrite';

import DashboardMenu from 'Components/DashboardMenu';

const getURL = (workflowState, entityType) => {
    const url = generateURLTo(workflowState, entityType);
    return url;
};

const getLabel = entityType => pluralize(entityLabels[entityType]);

const createOptions = (workflowState, types) => {
    return types.map(type => {
        return {
            label: getLabel(type),
            link: getURL(workflowState, type)
        };
    });
};

const ApplicationDashboardMenu = () => {
    const types = [
        entityTypes.CLUSTER,
        entityTypes.NAMESPACE,
        entityTypes.DEPLOYMENT,
        entityTypes.IMAGE,
        entityTypes.COMPONENT
    ];

    const workflowState = useContext(workflowStateContext);
    const options = createOptions(workflowState, types);

    return <DashboardMenu text="Application & Infrastructure" options={options} />;
};

export default ApplicationDashboardMenu;
