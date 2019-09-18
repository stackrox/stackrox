import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import Rules from 'Containers/ConfigManagement/Entity/widgets/Rules';
import RulePermissions from 'Containers/ConfigManagement/Entity/widgets/RulePermissions';
import isGQLLoading from 'utils/gqlLoading';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import appContexts from 'constants/appContextTypes';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';
import EntityList from '../List/EntityList';

const Role = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        id,
        query: queryService.objectToWhereClause(query[searchParam])
    };

    const defaultQuery = gql`
        query k8sRole($id: ID!${entityListType ? ', $query: String' : ''}) {
            clusters {
                id
                k8srole(role: $id) {
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
        }
    `;

    function getQuery() {
        if (!entityListType) return defaultQuery;
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.ROLE,
            entityListType,
            appContexts.CONFIG_MANAGEMENT
        );

        return gql`
            query getRole_${entityListType}($id: ID!, $query: String) {
                clusters {
                    id
                    k8srole(role: $id) {
                        id
                        ${listFieldName}(query: $query) { ...${fragmentName} }
                    }
                }
            }
            ${fragment}
        `;
    }
    return (
        <Query query={getQuery()} variables={variables}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) return <Loader transparent />;
                const { clusters } = data;
                if (!clusters || !clusters.length)
                    return <PageNotFound resourceType={entityTypes.ROLE} />;

                const { k8srole: entity } = clusters.find(cluster => cluster.k8srole);

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(entity, entityListType)}
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
                    clusterId
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
                        value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A'
                    }
                ];

                return (
                    <div className="w-full">
                        <CollapsibleSection title="Role Details">
                            <div className="flex mb-4 flex-wrap">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
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
                                {roleNamespace && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 h-48 mb-4"
                                        entityType={entityTypes.NAMESPACE}
                                        name="Namespace Scope"
                                        value={namespaceName}
                                        entityId={namespaceId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Users & Groups"
                                    value={subjectCount}
                                    entityType={entityTypes.SUBJECT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
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
