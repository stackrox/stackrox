import React from 'react';
import PropTypes from 'prop-types';

import PageNotFound from 'Components/PageNotFound';

import ConfigManagementListClusters from './ConfigManagementListClusters';
import ConfigManagementListControls from './ConfigManagementListControls';
import ConfigManagementListDeployments from './ConfigManagementListDeployments';
import ConfigManagementListImages from './ConfigManagementListImages';
import ConfigManagementListNamespaces from './ConfigManagementListNamespaces';
import ConfigManagementListNodes from './ConfigManagementListNodes';
import ConfigManagementListPolicies from './ConfigManagementListPolicies';
import ConfigManagementListRoles from './ConfigManagementListRoles';
import ConfigManagementListSecrets from './ConfigManagementListSecrets';
import ConfigManagementListServiceAccounts from './ConfigManagementListServiceAccounts';
import ConfigManagementListSubjects from './ConfigManagementListSubjects';

const entityComponentMap = {
    CLUSTER: ConfigManagementListClusters,
    CONTROL: ConfigManagementListControls,
    DEPLOYMENT: ConfigManagementListDeployments,
    IMAGE: ConfigManagementListImages,
    NAMESPACE: ConfigManagementListNamespaces,
    NODE: ConfigManagementListNodes,
    POLICY: ConfigManagementListPolicies,
    ROLE: ConfigManagementListRoles,
    SECRET: ConfigManagementListSecrets,
    SERVICE_ACCOUNT: ConfigManagementListServiceAccounts,
    SUBJECT: ConfigManagementListSubjects,
};

const EntityList = ({ entityListType, entityId, ...rest }) => {
    const Component = entityComponentMap[entityListType];
    if (!Component) {
        return <PageNotFound resourceType={entityListType} useCase="configmanagement" />;
    }
    return <Component selectedRowId={entityId} {...rest} />;
};

EntityList.propTypes = {
    entityListType: PropTypes.string.isRequired,
    entityId: PropTypes.string,
};

EntityList.defaultProps = {
    entityId: null,
};

export default EntityList;
