import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import isEmpty from 'lodash/isEmpty';
import { ROLE_FRAGMENT } from 'queries/role';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import EntityList from '../List/EntityList';

const processSubjectDataByClusters = data => {
    const entity = data.clusters.reduce(
        (acc, curr) => {
            if (!curr.subjects.length) return acc;
            const { subject, type, clusterAdmin, roles, ...rest } = curr.subjects[0];
            const { id: clusterId, name: clusterName } = curr;
            return {
                subject,
                type,
                clusterAdmin,
                roles: [...acc.roles, ...roles],
                clusters: [...acc.clusters, { ...rest, clusterId, clusterName }]
            };
        },
        { roles: [], clusters: [] }
    );
    return entity;
};

const getSubjectQuery = (id, entityContext) => {
    const queryObject = {
        Subject: id
    };
    if (!isEmpty(entityContext)) {
        Object.keys(entityContext).forEach(entityType => {
            queryObject[`${entityType} ID`] = entityContext[entityType];
        });
    }
    return queryService.objectToWhereClause(queryObject);
};

const Subject = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        subjectQuery: getSubjectQuery(id, entityContext),
        query: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
        query subject($subjectQuery: String!, $query: String) {
            clusters {
                id
                name
                subjects(query: $subjectQuery) {
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
                    clusterID
                    roles(query: $query) {
                        ${entityListType === entityTypes.ROLE ? '...k8roleFields' : 'id'}
                    }
                }
            }
        }
    ${entityListType === entityTypes.ROLE ? ROLE_FRAGMENT : ''}
    `;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                if (!data.clusters || !data.clusters.length)
                    return <PageNotFound resourceType={entityTypes.SUBJECT} />;
                const entity = processSubjectDataByClusters(data);
                const { type, clusterAdmin, clusters = [] } = entity;

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
                    (acc, { clusterId, clusterName, scopedPermissions }) => {
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
                                    value={entity.roles.length}
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
