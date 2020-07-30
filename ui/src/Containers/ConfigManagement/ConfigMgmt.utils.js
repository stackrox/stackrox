import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import { defaultClusterSort } from 'Containers/ConfigManagement/List/Clusters';
import { defaultNamespaceSort } from 'Containers/ConfigManagement/List/Namespaces';
import { defaultDeploymentSort } from 'Containers/ConfigManagement/List/Deployments';
import { defaultImageSort } from 'Containers/ConfigManagement/List/Images';
import { defaultNodeSort } from 'Containers/ConfigManagement/List/Nodes';
import { defaultPolicyrSort } from 'Containers/ConfigManagement/List/Policies';
import { defaultRoleSort } from 'Containers/ConfigManagement/List/Roles';
import { defaultSecretSort } from 'Containers/ConfigManagement/List/Secrets';
import { defaultServiceAccountSort } from 'Containers/ConfigManagement/List/ServiceAccounts';
import { defaultSubjectSort } from 'Containers/ConfigManagement/List/Subjects';

const defaultSortFieldMap = {
    [entityTypes.CLUSTER]: defaultClusterSort,
    [entityTypes.DEPLOYMENT]: defaultDeploymentSort,
    [entityTypes.NAMESPACE]: defaultNamespaceSort,
    [entityTypes.IMAGE]: defaultImageSort,
    [entityTypes.NODE]: defaultNodeSort,
    [entityTypes.POLICY]: defaultPolicyrSort,
    [entityTypes.ROLE]: defaultRoleSort,
    [entityTypes.SECRET]: defaultSecretSort,
    [entityTypes.SERVICE_ACCOUNT]: defaultServiceAccountSort,
    [entityTypes.SUBJECT]: defaultSubjectSort,
};

export function getConfigMgmtDefaultSort(entityListType = '') {
    const defaultSort = defaultSortFieldMap[entityListType];

    return defaultSort || [];
}

export function getConfigMgmtCountQuery(entityListType = '') {
    const parsedEntityListTypeCount = defaultCountKeyMap[entityListType];

    return !parsedEntityListTypeCount ||
        entityListType === entityTypes.CONTROL ||
        entityListType === entityTypes.POLICY
        ? ''
        : `count: ${parsedEntityListTypeCount}(query: $query)`;
}

export default {
    getConfigMgmtCountQuery,
};
