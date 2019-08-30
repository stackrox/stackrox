import React, { useContext } from 'react';
import { ROLE_FRAGMENT } from 'queries/role';

import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import gql from 'graphql-tag';
import searchContext from 'Containers/searchContext';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';
import queryService from 'modules/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import EntityList from '../List/EntityList';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';

const ServiceAccount = ({ id, entityListType, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Lifecycle Stage': 'DEPLOY'
        })
    };

    const QUERY = gql`
        query getServiceAccount($id: ID!${entityListType ? ', $query: String' : ''}) {
            serviceAccount(id: $id) {
                id
                name
                ${
                    entityContext[entityTypes.NAMESPACE]
                        ? ''
                        : `saNamespace {
                    metadata {
                        id
                        name
                    }
                }`
                }
                
                ${entityContext[entityTypes.CLUSTER] ? '' : 'clusterId clusterName'}
                ${
                    entityListType === entityTypes.DEPLOYMENT
                        ? 'deployments(query: $query) { ...deploymentFields }'
                        : 'deploymentCount'
                }
                ${
                    entityListType === entityTypes.ROLE
                        ? 'k8sroles: roles(query: $query) { ...k8roleFields }'
                        : 'roleCount'
                }
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
        ${entityListType === entityTypes.DEPLOYMENT ? DEPLOYMENT_FRAGMENT : ''}
        ${entityListType === entityTypes.ROLE ? ROLE_FRAGMENT : ''}
    `;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                const { serviceAccount: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.SERVICE_ACCOUNT} />;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityContext={{ ...entityContext, [entityTypes.SERVICE_ACCOUNT]: id }}
                            data={getSubListFromEntity(entity, entityListType)}
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
                    roleCount,
                    saNamespace,
                    scopedPermissions = [],
                    annotations,
                    clusterName,
                    clusterId
                } = entity;

                let namespaceName;
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
                        value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A'
                    }
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Service Account Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                    secrets={secrets}
                                />
                                {clusterName && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 h-48 mb-4"
                                        entityType={entityTypes.CLUSTER}
                                        name="Cluster"
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                                {saNamespace && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 h-48 mb-4"
                                        entityType={entityTypes.NAMESPACE}
                                        name="Namespace"
                                        value={namespaceName}
                                        entityId={namespaceId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Deployments"
                                    value={deploymentCount}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Roles"
                                    value={roleCount}
                                    entityType={entityTypes.ROLE}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Service Account Permissions">
                            <div className="flex mb-4 pdf-page pdf-stretch">
                                <ClusterScopedPermissions
                                    scopedPermissions={scopedPermissions}
                                    clusterName={clusterName}
                                    className="mx-4 bg-base-100 w-full"
                                />
                                <NamespaceScopedPermissions
                                    scopedPermissions={scopedPermissions}
                                    namespace={namespaceName}
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
ServiceAccount.propTypes = entityComponentPropTypes;
ServiceAccount.defaultProps = entityComponentDefaultProps;

export default ServiceAccount;
