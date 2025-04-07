import React, { useContext } from 'react';
import { format } from 'date-fns';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import RelatedEntity from 'Components/RelatedEntity';
import Metadata from 'Components/Metadata';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import DeploymentsWithFailedPolicies from 'Containers/ConfigManagement/Entity/widgets/DeploymentsWithFailedPolicies';
import searchContext from 'Containers/searchContext';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import EntityList from '../List/EntityList';

const Namespace = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
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
        query getNamespace($id: ID!, $query: String) {
            namespace(id: $id) {
                metadata {
                    name
                    id
                    labels {
                        key
                        value
                    }
                    creationTime
                }
                cluster {
                    id
                    name
                }
                imageCount
                deploymentCount
                subjectCount
                k8sRoleCount
                serviceAccountCount
                secretCount
                policyCount(query: $query)
            }
        }
    `;

    function getQuery() {
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.NAMESPACE,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);

        return gql`
            query getNamespace_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                namespace(id: $id) {
                    metadata {
                        id
                    }
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
                const { namespace } = data;
                if (!namespace) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.NAMESPACE}
                            useCase={useCases.CONFIG_MANAGEMENT}
                        />
                    );
                }

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(namespace, entityListType)}
                            totalResults={data?.namespace?.count}
                            entityContext={{ ...entityContext, [entityTypes.NAMESPACE]: id }}
                        />
                    );
                }

                const {
                    metadata = {},
                    cluster = {},
                    deploymentCount,
                    secretCount,
                    imageCount,
                    serviceAccountCount,
                    k8sRoleCount,
                } = namespace;

                const { name, creationTime, labels = [] } = metadata;

                const metadataKeyValuePairs = [
                    {
                        key: 'Created',
                        value: creationTime ? format(creationTime, dateTimeFormat) : 'N/A',
                    },
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Namespace Summary">
                            <div className="flex flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                />
                                {cluster && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.CLUSTER}
                                        name="Cluster"
                                        value={cluster.name}
                                        entityId={cluster.id}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Deployments"
                                    value={deploymentCount}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Secrets"
                                    value={secretCount}
                                    entityType={entityTypes.SECRET}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType={entityTypes.IMAGE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Service Accounts"
                                    value={serviceAccountCount}
                                    entityType={entityTypes.SERVICE_ACCOUNT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Roles"
                                    value={k8sRoleCount}
                                    entityType={entityTypes.ROLE}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Namespace Findings">
                            <div className="flex pdf-page pdf-stretch relative rounded mb-4 ml-4 mr-4">
                                <DeploymentsWithFailedPolicies
                                    query={queryService.objectToWhereClause({
                                        Cluster: cluster.name,
                                        Namespace: name,
                                    })}
                                    message="No deployments violating policies in this namespace"
                                    entityContext={{
                                        ...entityContext,
                                        [entityTypes.NAMESPACE]: id,
                                    }}
                                />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};
Namespace.propTypes = entityComponentPropTypes;
Namespace.defaultProps = entityComponentDefaultProps;

export default Namespace;
