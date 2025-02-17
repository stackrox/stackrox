import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';
import isGQLLoading from 'utils/gqlLoading';
import { gql } from '@apollo/client';
import useCases from 'constants/useCaseTypes';
import queryService from 'utils/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import EntityList from '../List/EntityList';

const Subject = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id: decodeURIComponent(id),
        query: queryService.objectToWhereClause(query[searchParam]),
        pagination,
    };

    const defaultQuery = gql`
        query getSubject($id: ID) {
            subject(id: $id) {
                id
                name
                kind
                namespace
                type
                scopedPermissions {
                    scope
                    permissions {
                        key
                        values
                    }
                }
                clusterName
                clusterId
                clusterAdmin
                k8sRoleCount
            }
        }
    `;

    function getQuery() {
        if (!entityListType) {
            return defaultQuery;
        }
        const { fragment } = queryService.getFragmentInfo(
            entityTypes.SUBJECT,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );

        return gql`
            query getSubject_${entityListType}($id: ID, $query: String, $pagination: Pagination) {
                subject(id: $id) {
                    id
                    name
                    kind
                    namespace
                    type
                    scopedPermissions {
                        scope
                        permissions {
                            key
                            values
                        }
                    }
                    k8sRoles(query: $query, pagination: $pagination) {
                       ...k8RoleFields
                    }
                    clusterAdmin
                    k8sRoleCount
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

                const entity = data.subject;
                const {
                    clusterId,
                    clusterName,
                    scopedPermissions,
                    type,
                    clusterAdmin,
                    k8sRoles,
                    k8sRoleCount,
                } = entity;

                if (entityListType) {
                    let listData;
                    let listCount;
                    switch (entityListType) {
                        case entityTypes.ROLE:
                            listData = k8sRoles;
                            listCount = k8sRoleCount;
                            break;
                        default:
                            listData = [];
                            listCount = 0;
                    }
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={listData}
                            totalResults={listCount}
                            query={query}
                            entityContext={{ ...entityContext, [entityTypes.SUBJECT]: id }}
                        />
                    );
                }

                const scopedPermissionsAcrossAllClusters = [
                    { clusterId, clusterName, scopedPermissions },
                ];
                const metadataKeyValuePairs = [
                    { key: 'Role type', value: type },
                    {
                        key: 'Cluster Admin Role',
                        value: clusterAdmin ? 'Enabled' : 'Disabled',
                    },
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Subject Summary">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Roles"
                                    value={k8sRoleCount}
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
