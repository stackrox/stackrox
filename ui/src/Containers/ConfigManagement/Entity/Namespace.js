import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import queryService from 'modules/queryService';
import appContexts from 'constants/appContextTypes';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import DeploymentsWithFailedPolicies from 'Containers/ConfigManagement/Entity/widgets/DeploymentsWithFailedPolicies';
import gql from 'graphql-tag';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';
import EntityList from '../List/EntityList';

const Namespace = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Lifecycle Stage': 'DEPLOY'
        })
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
                k8sroleCount
                serviceAccountCount
                secretCount
                policyCount(query: $query)
            }
        }
    `;

    function getQuery() {
        if (!entityListType) return defaultQuery;
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.NAMESPACE,
            entityListType,
            appContexts.CONFIG_MANAGEMENT
        );

        return gql`
            query getNamespace_${entityListType}($id: ID!, $query: String) {
                namespace(id: $id) {
                    metadata {
                        id
                    }
                    ${listFieldName}(query: $query) { ...${fragmentName} }
                }
            }
            ${fragment}
        `;
    }

    return (
        <Query query={getQuery()} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                const { namespace } = data;
                if (!namespace) return <PageNotFound resourceType={entityTypes.NAMESPACE} />;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(namespace, entityListType)}
                            entityContext={{ ...entityContext, [entityTypes.NAMESPACE]: id }}
                        />
                    );
                }

                const {
                    metadata = {},
                    cluster,
                    deploymentCount,
                    secretCount,
                    imageCount,
                    serviceAccountCount
                } = namespace;

                const { name, creationTime, labels = [] } = metadata;

                const metadataKeyValuePairs = [
                    {
                        key: 'Created',
                        value: creationTime ? format(creationTime, dateTimeFormat) : 'N/A'
                    }
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Namespace Details">
                            <div className="flex flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                />
                                {cluster && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 h-48 mb-4"
                                        entityType={entityTypes.CLUSTER}
                                        name="Cluster"
                                        value={cluster.name}
                                        entityId={cluster.id}
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
                                    name="Secrets"
                                    value={secretCount}
                                    entityType={entityTypes.SECRET}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType={entityTypes.IMAGE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Service Accounts"
                                    value={serviceAccountCount}
                                    entityType={entityTypes.SERVICE_ACCOUNT}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Namespace Findings">
                            <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                                <DeploymentsWithFailedPolicies
                                    query={queryService.objectToWhereClause({
                                        Cluster: cluster.name,
                                        Namespace: name
                                    })}
                                    message="No deployments violating policies in this namespace"
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
