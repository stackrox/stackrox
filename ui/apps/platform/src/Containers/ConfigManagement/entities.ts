import { configManagementPath, urlEntityListTypes } from 'routePaths';

export type ConfigMgmtEntityType =
    | 'CLUSTER'
    | 'CONTROL'
    | 'DEPLOYMENT'
    | 'IMAGE'
    | 'NAMESPACE'
    | 'NODE'
    | 'POLICY'
    | 'ROLE'
    | 'SECRET'
    | 'SERVICE_ACCOUNT'
    | 'SUBJECT';

export function getConfigMgmtPathForEntitiesAndId(
    entityListType: ConfigMgmtEntityType,
    id: string
) {
    return `${configManagementPath}/${urlEntityListTypes[entityListType]}/${id}`;
}
