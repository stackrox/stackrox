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
import { GraphQLSortOption } from 'types/search';
import { ConfigurationManagementEntityType } from 'utils/entityRelationships';

const defaultSortFieldMap: Record<ConfigurationManagementEntityType, GraphQLSortOption[]> = {
    CLUSTER: defaultClusterSort,
    CONTROL: [],
    DEPLOYMENT: defaultDeploymentSort,
    IMAGE: defaultImageSort,
    NAMESPACE: defaultNamespaceSort,
    NODE: defaultNodeSort,
    POLICY: defaultPolicyrSort,
    ROLE: defaultRoleSort,
    SECRET: defaultSecretSort,
    SERVICE_ACCOUNT: defaultServiceAccountSort,
    SUBJECT: defaultSubjectSort,
};

export function getConfigMgmtDefaultSort(
    entityListType: ConfigurationManagementEntityType
): GraphQLSortOption[] {
    const defaultSort = defaultSortFieldMap[entityListType];

    return defaultSort ?? [];
}

export function getConfigMgmtCountQuery(entityListType: ConfigurationManagementEntityType) {
    const parsedEntityListTypeCount = defaultCountKeyMap[entityListType];

    return !parsedEntityListTypeCount || entityListType === 'CONTROL' || entityListType === 'POLICY'
        ? ''
        : `count: ${parsedEntityListTypeCount}(query: $query)`;
}
