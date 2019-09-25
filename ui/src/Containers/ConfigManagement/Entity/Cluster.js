import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import gql from 'graphql-tag';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import isGQLLoading from 'utils/gqlLoading';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';
import getControlsWithStatus from '../List/utilities/getControlsWithStatus';
import NodesWithFailedControls from './widgets/NodesWithFailedControls';
import DeploymentsWithFailedPolicies from './widgets/DeploymentsWithFailedPolicies';
import EntityList from '../List/EntityList';

const Cluster = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const queryObject = { ...query[searchParam] };

    if (entityListType === entityTypes.POLICY) queryObject['Lifecycle Stage'] = 'DEPLOY';
    if (!queryObject.Standard) queryObject.Standard = 'CIS';

    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        id,
        query: queryService.objectToWhereClause(queryObject)
    };

    const defaultQuery = gql`
        query getCluster($id: ID!) {
            cluster(id: $id) {
                id
                name
                admissionController
                centralApiEndpoint
                imageCount
                nodeCount
                deploymentCount
                namespaceCount
                subjectCount
                k8sroleCount
                secretCount
                policyCount(query: "Lifecycle Stage:DEPLOY")
                serviceAccountCount
                complianceControlCount(query: "Standard:CIS") {
                    passingCount
                    failingCount
                    unknownCount
                }
                status {
                    orchestratorMetadata {
                        version
                        buildDate
                    }
                }
            }
        }
    `;

    function getQuery() {
        if (!entityListType) return defaultQuery;
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.CLUSTER,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );

        return gql`
            query getCluster_${entityListType}($id: ID!, $query: String) {
                cluster(id: $id) {
                    id
                    ${listFieldName}(query: $query) { ...${fragmentName} }
                }
            }
            ${fragment}
        `;
    }

    return (
        <Query query={getQuery()} variables={variables}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) return <Loader transparent />;
                const { cluster: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.CLUSTER} />;

                const { complianceResults = [] } = entity;

                if (entityListType) {
                    let listData = getSubListFromEntity(entity, entityListType);
                    if (entityListType === entityTypes.CONTROL) {
                        listData = getControlsWithStatus(complianceResults);
                    }
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={listData}
                            entityContext={{ ...entityContext, [entityTypes.CLUSTER]: id }}
                            query={query}
                        />
                    );
                }
                if (!entity.status) return null;

                const {
                    name,
                    nodeCount,
                    deploymentCount,
                    namespaceCount,
                    subjectCount,
                    serviceAccountCount,
                    k8sroleCount,
                    secretCount,
                    imageCount,
                    complianceControlCount,
                    status: { orchestratorMetadata = null }
                } = entity;

                const { version = 'N/A' } = orchestratorMetadata;

                const metadataKeyValuePairs = [
                    {
                        key: 'K8s version',
                        value: version
                    }
                ];

                const { passingCount, failingCount, unknownCount } = complianceControlCount;
                const totalControlCount = passingCount + failingCount + unknownCount;

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Cluster Details">
                            <div className="flex flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 min-w-48 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Nodes"
                                    value={nodeCount}
                                    entityType={entityTypes.NODE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Namespaces"
                                    value={namespaceCount}
                                    entityType={entityTypes.NAMESPACE}
                                />
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
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Roles"
                                    value={k8sroleCount}
                                    entityType={entityTypes.ROLE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="CIS Controls"
                                    value={totalControlCount}
                                    entityType={entityTypes.CONTROL}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Cluster Findings">
                            <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                                <Tabs
                                    hasTabSpacing
                                    headers={[{ text: 'Policies' }, { text: 'CIS Controls' }]}
                                >
                                    <TabContent>
                                        <DeploymentsWithFailedPolicies
                                            query={queryService.objectToWhereClause({
                                                Cluster: name
                                            })}
                                            message="No deployments violating policies in this cluster"
                                            entityContext={{
                                                ...entityContext,
                                                [entityTypes.CLUSTER]: id
                                            }}
                                        />
                                    </TabContent>
                                    <TabContent>
                                        <NodesWithFailedControls
                                            entityType={entityTypes.CLUSTER}
                                            entityContext={entityContext}
                                        />
                                    </TabContent>
                                </Tabs>
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Cluster.propTypes = entityComponentPropTypes;
Cluster.defaultProps = entityComponentDefaultProps;

export default Cluster;
