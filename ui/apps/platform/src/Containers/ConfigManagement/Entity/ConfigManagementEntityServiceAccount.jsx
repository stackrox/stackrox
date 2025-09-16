import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';
import searchContext from 'Containers/searchContext';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import { getDateTime } from 'utils/dateUtils';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import EntityList from '../List/EntityList';

const ConfigManagementEntityServiceAccount = ({
    id,
    entityListType,
    entityId1,
    query,
    entityContext,
    pagination,
}) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Lifecycle Stage': 'DEPLOY',
        }),
        pagination,
    };

    const defaultQuery = gql`
        query getServiceAccount($id: ID!) {
            serviceAccount(id: $id) {
                id
                name
                saNamespace {
                    metadata {
                        id
                        name
                    }
                }
                clusterId
                clusterName
                deploymentCount
                k8sRoleCount
                automountToken
                createdAt
                labels {
                    key
                    value
                }
                annotations {
                    key
                    value
                }
                secrets: imagePullSecretObjects {
                    id
                    name
                }
                scopedPermissions {
                    scope
                    permissions {
                        key
                        values
                    }
                }
            }
        }
    `;

    function getQuery() {
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            'SERVICE_ACCOUNT',
            entityListType,
            'configmanagement'
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);

        return gql`
            query getServiceAccount_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                serviceAccount(id: $id) {
                    id
                    ${listFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
                    ${countQuery}
                }
            }
            ${fragment}
        `;
    }

    return (
        <Query query={getQuery()} variables={variables} fetchPolicy="network-only">
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }
                const { serviceAccount: entity } = data;
                if (!entity) {
                    return (
                        <PageNotFound resourceType="SERVICE_ACCOUNT" useCase="configmanagement" />
                    );
                }

                if (entityListType) {
                    const listData =
                        entityListType === 'ROLE'
                            ? entity.k8sRoles
                            : getSubListFromEntity(entity, entityListType);
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            entityContext={{ ...entityContext, SERVICE_ACCOUNT: id }}
                            data={listData}
                            totalResults={data?.serviceAccount?.count}
                            query={query}
                        />
                    );
                }

                const {
                    automountToken = false,
                    createdAt,
                    labels = [],
                    secrets = [],
                    deploymentCount,
                    k8sRoleCount,
                    saNamespace,
                    scopedPermissions = [],
                    annotations,
                    clusterName = '',
                    clusterId = '',
                } = entity;

                let namespaceName = '';
                let namespaceId;
                if (saNamespace) {
                    const { metadata } = saNamespace;
                    namespaceName = metadata.name;
                    namespaceId = metadata.id;
                }

                const metadataKeyValuePairs = [
                    { key: 'Automounted', value: automountToken.toString() },
                    {
                        key: 'Created',
                        value: createdAt ? getDateTime(createdAt) : 'N/A',
                    },
                ];

                const scopedPermissionsByCluster = [{ clusterId, clusterName, scopedPermissions }];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Service Account Summary">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                    secrets={secrets}
                                />
                                {!(entityContext && entityContext.CLUSTER) && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType="CLUSTER"
                                        name="Cluster"
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                                {!(entityContext && entityContext.NAMESPACE) && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType="NAMESPACE"
                                        name="Namespace"
                                        value={namespaceName}
                                        entityId={namespaceId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Deployments"
                                    value={deploymentCount}
                                    entityType="DEPLOYMENT"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Roles"
                                    value={k8sRoleCount}
                                    entityType="ROLE"
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Service Account Permissions">
                            <div className="flex mb-4 pdf-page pdf-stretch">
                                <ClusterScopedPermissions
                                    scopedPermissionsByCluster={scopedPermissionsByCluster}
                                    className="mx-4 bg-base-100 w-full"
                                />
                                <NamespaceScopedPermissions
                                    scopedPermissionsByCluster={scopedPermissionsByCluster}
                                    className="flex-grow mx-4 bg-base-100 w-full"
                                />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};
ConfigManagementEntityServiceAccount.propTypes = entityComponentPropTypes;
ConfigManagementEntityServiceAccount.defaultProps = entityComponentDefaultProps;

export default ConfigManagementEntityServiceAccount;
