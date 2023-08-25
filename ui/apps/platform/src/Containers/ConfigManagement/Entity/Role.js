import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import Rules from 'Containers/ConfigManagement/Entity/widgets/Rules';
import RulePermissions from 'Containers/ConfigManagement/Entity/widgets/RulePermissions';
import isGQLLoading from 'utils/gqlLoading';
import { gql } from '@apollo/client';
import queryService from 'utils/queryService';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import EntityList from '../List/EntityList';

const Role = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause(query[searchParam]),
        pagination,
    };

    const defaultQuery = gql`
        query getRole($id: ID!${entityListType ? ', $query: String' : ''}) {
            k8sRole(id: $id) {
                id
                name
                type
                verbs
                createdAt
                ${
                    entityContext[entityTypes.NAMESPACE]
                        ? ''
                        : `roleNamespace {
                    metadata {
                        id
                        name
                    }
                }`
                }
                serviceAccountCount
                subjectCount
                rules {
                    apiGroups
                    nonResourceUrls
                    resourceNames
                    resources
                    verbs
                }
                ${entityContext[entityTypes.CLUSTER] ? '' : 'clusterId clusterName'}
            }
        }
    `;

    function getQuery() {
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.ROLE,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );

        const countQuery = getConfigMgmtCountQuery(entityListType);
        return gql`
            query getRole_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                    k8sRole(id: $id) {
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

                const { k8sRole: entity } = data;
                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(entity, entityListType)}
                            totalResults={entity.count}
                            entityContext={{ ...entityContext, [entityTypes.ROLE]: id }}
                            query={query}
                        />
                    );
                }

                const {
                    type,
                    createdAt,
                    roleNamespace,
                    serviceAccountCount,
                    subjectCount,
                    labels = [],
                    annotations = [],
                    rules,
                    clusterName,
                    clusterId,
                } = entity;

                let namespaceName;
                let namespaceId;
                if (roleNamespace) {
                    namespaceName = roleNamespace.metadata.name;
                    namespaceId = roleNamespace.metadata.id;
                }

                const metadataKeyValuePairs = [
                    { key: 'Role Type', value: type },
                    {
                        key: 'Created',
                        value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A',
                    },
                ];

                return (
                    <div className="w-full">
                        <CollapsibleSection title="Role Summary">
                            <div className="flex mb-4 flex-wrap">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                />
                                {clusterName && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.CLUSTER}
                                        name="Cluster"
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                                {roleNamespace && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.NAMESPACE}
                                        name="Namespace Scope"
                                        value={namespaceName}
                                        entityId={namespaceId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Users & Groups"
                                    value={subjectCount}
                                    entityType={entityTypes.SUBJECT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Service Accounts"
                                    value={serviceAccountCount}
                                    entityType={entityTypes.SERVICE_ACCOUNT}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Role Permissions And Rules">
                            <div className="flex mb-4">
                                <RulePermissions rules={rules} className="mx-4 bg-base-100" />
                                <Rules rules={rules} className="mx-4 bg-base-100" />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Role.propTypes = entityComponentPropTypes;
Role.defaultProps = entityComponentDefaultProps;

export default Role;
