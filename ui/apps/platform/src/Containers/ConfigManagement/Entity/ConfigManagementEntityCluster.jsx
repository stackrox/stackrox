import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import BinderTabs from 'Components/BinderTabs';
import Tab from 'Components/Tab';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import isGQLLoading from 'utils/gqlLoading';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import queryService from 'utils/queryService';
import getControlsWithStatus from '../List/utilities/getControlsWithStatus';
import NodesWithFailedControls from './widgets/NodesWithFailedControls';
import DeploymentsWithFailedPolicies from './widgets/DeploymentsWithFailedPolicies';
import EntityList from '../List/EntityList';

const ConfigManagementEntityCluster = ({
    id,
    entityListType,
    entityId1,
    query,
    entityContext,
    pagination,
}) => {
    const searchParam = useContext(searchContext);

    const queryObject = { ...query[searchParam] };

    if (entityListType === 'POLICY') {
        queryObject['Lifecycle Stage'] = 'DEPLOY';
    }
    if (!queryObject.Standard && entityListType === 'CONTROL') {
        queryObject.Standard = 'CIS';
    }

    const variables = {
        id,
        query: queryService.objectToWhereClause(queryObject),
        pagination,
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
                k8sRoleCount
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
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            'CLUSTER',
            entityListType,
            'configmanagement'
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);
        const availableVars =
            entityListType === 'CONTROL'
                ? '$id: ID!, $query: String'
                : '$id: ID!, $query: String, $pagination: Pagination';
        const listQueryVars =
            entityListType === 'CONTROL'
                ? 'query: $query'
                : 'query: $query, pagination: $pagination';

        return gql`
            query getCluster_${entityListType}(${availableVars}) {
                cluster(id: $id) {
                    id
                    ${listFieldName}(${listQueryVars}) { ...${fragmentName} }
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
                const { cluster: entity } = data;
                if (!entity) {
                    return <PageNotFound resourceType="CLUSTER" useCase="configmanagement" />;
                }

                const { complianceResults = [] } = entity;

                if (entityListType) {
                    let listData = getSubListFromEntity(entity, entityListType);
                    if (entityListType === 'CONTROL') {
                        listData = getControlsWithStatus(complianceResults);
                    } else if (entityListType === 'SUBJECT') {
                        listData = listData.map((listItem) => {
                            return {
                                ...listItem,
                                subjectWithClusterID: listItem?.subject?.subjectWithClusterID || [],
                            };
                        });
                    }

                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={listData}
                            totalResults={data?.cluster?.count}
                            entityContext={{ ...entityContext, CLUSTER: id }}
                            query={query}
                        />
                    );
                }
                if (!entity.status) {
                    return null;
                }

                const {
                    name,
                    nodeCount,
                    deploymentCount,
                    namespaceCount,
                    subjectCount,
                    serviceAccountCount,
                    k8sRoleCount,
                    secretCount,
                    imageCount,
                    complianceControlCount,
                    status: { orchestratorMetadata = null },
                } = entity;

                const { version = 'N/A' } = orchestratorMetadata;

                const metadataKeyValuePairs = [
                    {
                        key: 'K8s version',
                        value: version,
                    },
                ];

                const { passingCount, failingCount, unknownCount } = complianceControlCount;
                const totalControlCount = passingCount + failingCount + unknownCount;

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Cluster Summary">
                            <div className="flex flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 min-w-48 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Nodes"
                                    value={nodeCount}
                                    entityType="NODE"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Namespaces"
                                    value={namespaceCount}
                                    entityType="NAMESPACE"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Deployments"
                                    value={deploymentCount}
                                    entityType="DEPLOYMENT"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Secrets"
                                    value={secretCount}
                                    entityType="SECRET"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType="IMAGE"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Users & Groups"
                                    value={subjectCount}
                                    entityType="SUBJECT"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Service Accounts"
                                    value={serviceAccountCount}
                                    entityType="SERVICE_ACCOUNT"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Roles"
                                    value={k8sRoleCount}
                                    entityType="ROLE"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="CIS Controls"
                                    value={totalControlCount}
                                    entityType="CONTROL"
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Cluster Findings">
                            <div className="flex pdf-page pdf-stretch relative rounded mb-4 ml-4 mr-4">
                                <BinderTabs>
                                    <Tab title="Policies">
                                        <DeploymentsWithFailedPolicies
                                            query={queryService.objectToWhereClause({
                                                Cluster: name,
                                            })}
                                            message="No deployments violating policies in this cluster"
                                            entityContext={{
                                                ...entityContext,
                                                CLUSTER: id,
                                            }}
                                        />
                                    </Tab>
                                    <Tab title="CIS Controls">
                                        <NodesWithFailedControls
                                            entityType="CLUSTER"
                                            entityContext={{
                                                ...entityContext,
                                                CLUSTER: id,
                                            }}
                                        />
                                    </Tab>
                                </BinderTabs>
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

ConfigManagementEntityCluster.propTypes = entityComponentPropTypes;
ConfigManagementEntityCluster.defaultProps = entityComponentDefaultProps;

export default ConfigManagementEntityCluster;
