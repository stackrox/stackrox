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

const ServiceAccount = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
    query serviceAccount($id: ID!) {
        serviceAccount(id: $id) {
            id
            name
            namespace
            saNamespace {
                metadata {
                    id
                    name
                }
            }
            deployments {
                ${entityListType === entityTypes.DEPLOYMENT ? '...deploymentFields' : 'id'}
            }
            k8sroles: roles {
                ${entityListType === entityTypes.ROLE ? '...roleFields' : 'id'}
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
                if (loading) return <Loader />;
                const { serviceAccount: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.SERVICE_ACCOUNT} />;

                const {
                    automountToken = false,
                    createdAt,
                    labels = [],
                    secrets = [],
                    deployments = [],
                    k8sroles = [],
                    saNamespace: { metadata = {} },
                    scopedPermissions = [],
                    annotations
                } = entity;

                const { name: namespaceName, id: namespaceId } = metadata;

                const metadataKeyValuePairs = [
                    { key: 'Automounted', value: automountToken.toString() },
                    {
                        key: 'Created',
                        value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A'
                    }
                ];

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={getSubListFromEntity(entity, entityListType)}
                            query={query}
                        />
                    );
                }

                return (
                    <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Service Account Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                    secrets={secrets}
                                />
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.NAMESPACE}
                                    name="Namespace"
                                    value={namespaceName}
                                    entityId={namespaceId}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Deployments"
                                    value={deployments.length}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Roles"
                                    value={k8sroles.length}
                                    entityType={entityTypes.ROLE}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Service Account Permissions">
                            <div className="flex mb-4 pdf-page pdf-stretch">
                                <ClusterScopedPermissions
                                    scopedPermissions={scopedPermissions}
                                    className="mx-4 bg-base-100"
                                />
                                <NamespaceScopedPermissions
                                    scopedPermissions={scopedPermissions}
                                    className="flex-grow mx-4 bg-base-100"
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
