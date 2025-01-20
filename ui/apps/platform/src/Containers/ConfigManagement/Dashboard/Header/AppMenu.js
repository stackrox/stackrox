import React from 'react';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';
import entityLabels from 'messages/entity';
import URLService from 'utils/URLService';
import { useLocation, useMatch } from 'react-router-dom';
import { workflowPaths } from 'routePaths';

import DashboardMenu from 'Components/DashboardMenu';

const getLabel = (entityType) => pluralize(entityLabels[entityType]);

const createOptions = (urlBuilder, types) => {
    return types.map((type) => {
        return {
            label: getLabel(type),
            link: urlBuilder.base(type).url(),
        };
    });
};

const AppMenu = () => {
    const types = [
        entityTypes.CLUSTER,
        entityTypes.NAMESPACE,
        entityTypes.NODE,
        entityTypes.DEPLOYMENT,
        entityTypes.IMAGE,
        entityTypes.SECRET,
    ];

    const match = useMatch(workflowPaths.DASHBOARD);
    const location = useLocation();
    const urlBuilder = URLService.getURL(match, location);
    const options = createOptions(urlBuilder, types);

    return <DashboardMenu text="Application & Infrastructure" options={options} />;
};

export default AppMenu;
