import React from 'react';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';
import entityLabels from 'messages/entity';
import URLService from 'utils/URLService';
import { withRouter } from 'react-router-dom';

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

const RBACMenu = ({ match, location }) => {
    const types = [entityTypes.SUBJECT, entityTypes.SERVICE_ACCOUNT, entityTypes.ROLE];

    const urlBuilder = URLService.getURL(match, location);
    const options = createOptions(urlBuilder, types);

    return <DashboardMenu text="Role-Based Access Control" options={options} />;
};

export default withRouter(RBACMenu);
