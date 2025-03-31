import React from 'react';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';
import entityLabels from 'messages/entity';
import URLService from 'utils/URLService';
import { useLocation } from 'react-router-dom';

import DashboardMenu from 'Components/DashboardMenu';
import useWorkflowMatch from 'hooks/useWorkflowMatch';

const getLabel = (entityType) => pluralize(entityLabels[entityType]);

const createOptions = (urlBuilder, types) => {
    return types.map((type) => {
        return {
            label: getLabel(type),
            link: urlBuilder.base(type).url(),
        };
    });
};

const RBACMenu = () => {
    const types = [entityTypes.SUBJECT, entityTypes.SERVICE_ACCOUNT, entityTypes.ROLE];

    const match = useWorkflowMatch();
    const location = useLocation();
    const urlBuilder = URLService.getURL(match, location);
    const options = createOptions(urlBuilder, types);

    return <DashboardMenu text="Role-Based Access Control" options={options} />;
};

export default RBACMenu;
