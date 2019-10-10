import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';
import isGQLLoading from 'utils/gqlLoading';
import gql from 'graphql-tag';
import useCases from 'constants/useCaseTypes';
import queryService from 'modules/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import EntityList from '../List/EntityList';

const processSubjectDataByClusters = data => {
    const entity = data.clusters.reduce(
        (acc, curr) => {
            if (!curr.subject) return acc;
            const { subject, type, clusterAdmin, roles, roleCount = 0, ...rest } = curr.subject;
            const { id: clusterId, name: clusterName } = curr;
            let allRoles = [...acc.roles];
            if (roles) allRoles = allRoles.concat(roles);
            const totalRoles = acc.roleCount + roleCount;
            return {
                subject,
                type,
                clusterAdmin,
                roles: allRoles,
                roleCount: totalRoles,
                clusters: [...acc.clusters, { ...rest, clusterId, clusterName }]
            };
        },
        { roles: [], clusters: [], roleCount: 0 }
    );
    return entity;
};

const getClustersQuery = entityContext => {
    if (entityContext && entityContext[entityTypes.CLUSTER]) {
        return queryService.objectToWhereClause({
            [`${entityTypes.CLUSTER} ID`]: entityContext[entityTypes.CLUSTER]
        });
    }
    return null;
};

const Subject = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        clustersQuery: getClustersQuery(entityContext),
        name: id,
        query: queryService.objectToWhereClause(query[searchParam])
    };

    const defaultQuery = gql`
        query subject($clustersQuery: String, $name: String!) {
            clusters(query: $clustersQuery) {
                id
                name
                subject(name: $name) {
                    id: name
                    subject {
                        name
                        kind
                        namespace
                    }
                    type
                    scopedPermissions {
                        scope
                        permissions {
                            key
                            values
                        }
                    }
                    clusterAdmin
                    roleCount
                }
            }
        }
    `;

    function getQuery() {
        if (!entityListType) return defaultQuery;
        const { fragment } = queryService.getFragmentInfo(
            entityTypes.SUBJECT,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );

        return gql`
            query subject($clustersQuery: String, $name: String!, $query: String) {
                clusters(query: $clustersQuery) {
                    id
                    subject(name: $name) {
                        id: name
                        name
                        roles(query: $query) {
                            ...k8roleFields
                        }
                    }
                }
            }
            ${fragment}
        `;
    }

    return (
        <Query query={getQuery()} variables={variables} fetchPolicy="no-cache">
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) return <Loader transparent />;

                if (!data.clusters || !data.clusters.length)
                    return <PageNotFound resourceType={entityTypes.SUBJECT} />;
                const entity = processSubjectDataByClusters(data);
                const { type, clusterAdmin, clusters = [], roleCount } = entity;

                if (entityListType) {
                    let listData;
                    switch (entityListType) {
                        case entityTypes.ROLE:
                            listData = entity.roles;
                            break;
                        default:
                            listData = [];
                    }
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={listData}
                            query={query}
                            entityContext={{ ...entityContext, [entityTypes.SUBJECT]: id }}
                        />
                    );
                }

                const scopedPermissionsAcrossAllClusters = clusters.reduce(
                    (acc, { clusterId = '', clusterName = '', scopedPermissions = [] }) => {
                        return [...acc, { clusterId, clusterName, scopedPermissions }];
                    },
                    []
                );
                const metadataKeyValuePairs = [
                    { key: 'Role type', value: type },
                    {
                        key: 'Cluster Admin Role',
                        value: clusterAdmin ? 'Enabled' : 'Disabled'
                    }
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Subject Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Roles"
                                    value={roleCount}
                                    entityType={entityTypes.ROLE}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Subject Permissions">
                            <div className="flex mb-4 pdf-page pdf-stretch">
                                <ClusterScopedPermissions
                                    scopedPermissionsByCluster={scopedPermissionsAcrossAllClusters}
                                    className="mx-4 bg-base-100"
                                />
                                <NamespaceScopedPermissions
                                    scopedPermissionsByCluster={scopedPermissionsAcrossAllClusters}
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

Subject.propTypes = entityComponentPropTypes;
Subject.defaultProps = entityComponentDefaultProps;

export default Subject;
