import React, { useContext } from 'react';
import PropTypes from 'prop-types';

import PageNotFound from 'Components/PageNotFound';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import configMgmtPaginationContext from 'Containers/configMgmtPaginationContext';
import workflowStateContext from 'Containers/workflowStateContext';
import { getConfigMgmtDefaultSort } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import queryService from 'utils/queryService';
import ServiceAccount from './Entity/ServiceAccount';
import Secret from './Entity/Secret';
import Deployment from './Entity/Deployment/Deployment';
import Node from './Entity/Node';
import Cluster from './Entity/Cluster';
import Namespace from './Entity/Namespace';
import Role from './Entity/Role';
import Control from './Entity/Control';
import Image from './Entity/Image';
import Policy from './Entity/Policy/Policy';
import Subject from './Entity/Subject';

const entityComponentMap = {
    [entityTypes.SERVICE_ACCOUNT]: ServiceAccount,
    [entityTypes.ROLE]: Role,
    [entityTypes.SECRET]: Secret,
    [entityTypes.DEPLOYMENT]: Deployment,
    [entityTypes.CLUSTER]: Cluster,
    [entityTypes.NAMESPACE]: Namespace,
    [entityTypes.NODE]: Node,
    [entityTypes.CONTROL]: Control,
    [entityTypes.NODE]: Node,
    [entityTypes.IMAGE]: Image,
    [entityTypes.POLICY]: Policy,
    [entityTypes.SUBJECT]: Subject,
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
        return <PageNotFound resourceType={entityType} useCase={useCases.CONFIG_MANAGEMENT} />;
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
