import React from 'react';
import entityTypes from 'constants/entityTypes';

import PageNotFound from 'Components/PageNotFound';
import Namespaces from '../List/Namespaces';
import ServiceAccounts from '../List/ServiceAccounts';
import Clusters from '../List/Clusters';
import Deployments from '../List/Deployments';
import Secrets from '../List/Secrets';
import Roles from '../List/Roles';

const EntityList = ({ entityListType, onRowClick }) => {
    switch (entityListType) {
        case entityTypes.SERVICE_ACCOUNT:
            return <ServiceAccounts onRowClick={onRowClick} />;
        case entityTypes.CLUSTER:
            return <Clusters onRowClick={onRowClick} />;
        case entityTypes.NAMESPACE:
            return <Namespaces onRowClick={onRowClick} />;
        case entityTypes.DEPLOYMENT:
            return <Deployments onRowClick={onRowClick} />;
        case entityTypes.SECRET:
            return <Secrets onRowClick={onRowClick} />;
        case entityTypes.ROLE:
            return <Roles onRowClick={onRowClick} />;
        default:
            return <PageNotFound resourceType={entityListType} />;
    }
};

export default EntityList;
