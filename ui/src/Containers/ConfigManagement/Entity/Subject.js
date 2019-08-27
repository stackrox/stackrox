import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { ROLE_FRAGMENT } from 'queries/role';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
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
            const {
                subject,
                type,
                clusterAdmin,
                clusterID,
                clusterName,
                roles,
                ...rest
            } = curr.subject;
            return {
                subject,
                type,
                clusterAdmin,
                clusterID,
                clusterName,
                roles,
                clusters: [...acc.clusters, { ...rest }]
            };
        },
        { clusters: [] }
    );
    return entity;
};

const Subject = ({ id, entityListType, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
        query subject($id: String!, $query: String) {
            clusters {
                id
                subject(name: $id) {
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
                    clusterName
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
                const { type, clusterAdmin, clusterID, clusterName, clusters = [] } = entity;

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
                            data={listData}
                            query={query}
                            entityContext={{ ...entityContext, [entityTypes.SUBJECT]: id }}
                        />
                    );
                }

                const scopedPermissionsAcrossAllClusters = clusters.reduce(
                    (acc, { scopedPermissions }) => {
                        return [...acc, ...scopedPermissions];
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
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.CLUSTER}
                                    name="Cluster"
                                    value={clusterName}
                                    entityId={clusterID}
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
                                    scopedPermissions={scopedPermissionsAcrossAllClusters}
                                    className="mx-4 bg-base-100"
                                />
                                <NamespaceScopedPermissions
                                    scopedPermissions={scopedPermissionsAcrossAllClusters}
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
