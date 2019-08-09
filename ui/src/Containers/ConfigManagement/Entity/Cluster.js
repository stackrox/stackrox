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
import { NODE_FRAGMENT } from 'queries/node';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';
import { NAMESPACE_FRAGMENT } from 'queries/namespace';
import { SUBJECT_WITH_CLUSTER_FRAGMENT } from 'queries/subject';
import { ROLE_FRAGMENT } from 'queries/role';
import { SECRET_FRAGMENT } from 'queries/secret';
import { SERVICE_ACCOUNT_FRAGMENT } from 'queries/serviceAccount';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import { POLICY_FRAGMENT } from 'queries/policy';
import { IMAGE_FRAGMENT } from 'queries/image';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';
import NodesWithFailedControls from './widgets/NodesWithFailedControls';
import DeploymentsWithFailedPolicies from './widgets/DeploymentsWithFailedPolicies';
import EntityList from '../List/EntityList';

const Cluster = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
    query getCluster($id: ID!) {
        cluster(id: $id) {
            id
            name
            admissionController
            centralApiEndpoint
            alertsCount
            images {
                ${entityListType === entityTypes.IMAGE ? '...imageFields' : 'id'}
            }
            nodes {
                ${entityListType === entityTypes.NODE ? '...nodeFields' : 'id'}
            }
            deployments {
                ${entityListType === entityTypes.DEPLOYMENT ? '...deploymentFields' : 'id'}
            }
            namespaces {
                ${entityListType === entityTypes.NAMESPACE ? '...namespaceFields' : 'metadata{id}'}
            }
            subjects {
                ${entityListType === entityTypes.SUBJECT ? '...subjectWithClusterFields' : 'name'}
            }
            k8sroles {
                ${entityListType === entityTypes.ROLE ? '...roleFields' : 'id'}
            }
            secrets {
                ${entityListType === entityTypes.SECRET ? '...secretFields' : 'id'}
            }
            policies(query: "Lifecycle Stage:DEPLOY") {
                ${entityListType === entityTypes.POLICY ? '...policyFields' : 'id'}
            }
            serviceAccounts {
                ${
                    entityListType === entityTypes.SERVICE_ACCOUNT
                        ? '...serviceAccountFields'
                        : 'id'
                }                
            }
            status {
                orchestratorMetadata {
                    version
                    buildDate
                }
            }
        }
    }
    ${entityListType === entityTypes.IMAGE ? IMAGE_FRAGMENT : ''}
    ${entityListType === entityTypes.NODE ? NODE_FRAGMENT : ''}
    ${entityListType === entityTypes.DEPLOYMENT ? DEPLOYMENT_FRAGMENT : ''}
    ${entityListType === entityTypes.NAMESPACE ? NAMESPACE_FRAGMENT : ''}
    ${entityListType === entityTypes.SUBJECT ? SUBJECT_WITH_CLUSTER_FRAGMENT : ''}
    ${entityListType === entityTypes.ROLE ? ROLE_FRAGMENT : ''}
    ${entityListType === entityTypes.SERVICE_ACCOUNT ? SERVICE_ACCOUNT_FRAGMENT : ''}
    ${entityListType === entityTypes.SECRET ? SECRET_FRAGMENT : ''}
    ${entityListType === entityTypes.POLICY ? POLICY_FRAGMENT : ''}
`;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading || !data) return <Loader />;
                const { cluster: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.CLUSTER} />;

                const {
                    name,
                    nodes = [],
                    namespaces = [],
                    deployments = [],
                    subjects = [],
                    serviceAccounts = [],
                    k8sroles = [],
                    secrets = [],
                    images = [],
                    policies = [],
                    status: { orchestratorMetadata = null }
                } = entity;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={getSubListFromEntity(entity, entityListType)}
                            query={query}
                        />
                    );
                }

                const { version = 'N/A' } = orchestratorMetadata;

                const metadataKeyValuePairs = [
                    {
                        key: 'K8s version',
                        value: version
                    }
                ];

                return (
                    <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Cluster Details">
                            <div className="flex flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    counts={null}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Nodes"
                                    value={nodes.length}
                                    entityType={entityTypes.NODE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Namespaces"
                                    value={namespaces.length}
                                    entityType={entityTypes.NAMESPACE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Deployments"
                                    value={deployments.length}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Secrets"
                                    value={secrets.length}
                                    entityType={entityTypes.SECRET}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Images"
                                    value={images.length}
                                    entityType={entityTypes.IMAGE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Users & Groups"
                                    value={subjects.length}
                                    entityType={entityTypes.SUBJECT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Service Accounts"
                                    value={serviceAccounts.length}
                                    entityType={entityTypes.SERVICE_ACCOUNT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Roles"
                                    value={k8sroles.length}
                                    entityType={entityTypes.ROLE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Policies"
                                    value={policies.length}
                                    entityType={entityTypes.POLICY}
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
                                        />
                                    </TabContent>
                                    <TabContent>
                                        <NodesWithFailedControls />
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
