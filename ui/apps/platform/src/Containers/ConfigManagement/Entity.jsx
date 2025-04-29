import React, { useContext } from 'react';
import PropTypes from 'prop-types';

import PageNotFound from 'Components/PageNotFound';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import configMgmtPaginationContext from 'Containers/configMgmtPaginationContext';
import workflowStateContext from 'Containers/workflowStateContext';
import { getConfigMgmtDefaultSort } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import queryService from 'utils/queryService';

import ConfigManagementEntityCluster from './Entity/ConfigManagementEntityCluster';
import ConfigManagementEntityControl from './Entity/ConfigManagementEntityControl';
import ConfigManagementEntityDeployment from './Entity/Deployment/ConfigManagementEntityDeployment';
import ConfigManagementEntityImage from './Entity/ConfigManagementEntityImage';
import ConfigManagementEntityNamespace from './Entity/ConfigManagementEntityNamespace';
import ConfigManagementEntityNode from './Entity/ConfigManagementEntityNode';
import ConfigManagementEntityPolicy from './Entity/Policy/ConfigManagementEntityPolicy';
import ConfigManagementEntityRole from './Entity/ConfigManagementEntityRole';
import ConfigManagementEntitySecret from './Entity/ConfigManagementEntitySecret';
import ConfigManagementEntityServiceAccount from './Entity/ConfigManagementEntityServiceAccount';
import ConfigManagementEntitySubject from './Entity/ConfigManagementEntitySubject';

const entityComponentMap = {
    CLUSTER: ConfigManagementEntityCluster,
    CONTROL: ConfigManagementEntityControl,
    DEPLOYMENT: ConfigManagementEntityDeployment,
    IMAGE: ConfigManagementEntityImage,
    NAMESPACE: ConfigManagementEntityNamespace,
    NODE: ConfigManagementEntityNode,
    POLICY: ConfigManagementEntityPolicy,
    ROLE: ConfigManagementEntityRole,
    SECRET: ConfigManagementEntitySecret,
    SERVICE_ACCOUNT: ConfigManagementEntityServiceAccount,
    SUBJECT: ConfigManagementEntitySubject,
};

const Entity = ({ entityType, entityId, entityListType, ...rest }) => {
    const workflowState = useContext(workflowStateContext);
    const configMgmtPagination = useContext(configMgmtPaginationContext);
    const page = workflowState.paging[configMgmtPagination.pageParam];
    const pageSort = workflowState.sort[configMgmtPagination.sortParam];

    const defaultSorted = getConfigMgmtDefaultSort(entityListType);
    const tableSort = pageSort || defaultSorted;

    const pagination = queryService.getPagination(tableSort, page, LIST_PAGE_SIZE);

    const Component = entityComponentMap[entityType];
    if (!Component) {
        return <PageNotFound resourceType={entityType} useCase="configmanagement" />;
    }
    return (
        <div className={`flex w-full h-full ${entityListType ? 'bg-base-100' : 'bg-base-200'}`}>
            <Component
                id={entityId}
                entityListType={entityListType}
                pagination={pagination}
                {...rest}
            />
        </div>
    );
};

Entity.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    entityId: PropTypes.string.isRequired,
    query: PropTypes.shape({}),
};
Entity.defaultProps = {
    query: null,
    entityListType: undefined,
};

export default Entity;
