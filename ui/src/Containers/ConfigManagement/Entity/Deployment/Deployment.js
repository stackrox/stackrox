import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import { POLICY_FRAGMENT } from 'queries/policy';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import gql from 'graphql-tag';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import { IMAGE_FRAGMENT } from 'queries/image';
import EntityList from '../../List/EntityList';
import getSubListFromEntity from '../../List/utilities/getSubListFromEntity';
import DeploymentFindings from './DeploymentFindings';

const Deployment = ({ id, entityContext, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
        query getDeployment($id: ID!) {
            deployment(id: $id) {
                id
                annotations {
                    key
                    value
                }
                cluster {
                    id
                    name
                }
                hostNetwork: id
                imagePullSecrets
                inactive
                labels {
                    key
                    value
                }
                name
                namespace
                namespaceId
                ports {
                    containerPort
                    exposedPort
                    exposure
                    exposureInfos {
                        externalHostnames
                        externalIps
                        level
                        nodePort
                        serviceClusterIp
                        serviceId
                        serviceName
                        servicePort
                    }
                    name
                    protocol
                }
                priority
                replicas
                serviceAccount
                serviceAccountID
                policyStatus {
                    status
                    failingPolicies {${
                        entityListType === entityTypes.POLICY ? '...policyFields' : 'id'
                    }}
                }
                tolerations {
                    key
                    operator
                    taintEffect
                    value
                }
                type
                updatedAt
                ${entityListType === entityTypes.IMAGE ? 'images { ...imageFields }' : 'imageCount'}
            }
        }
    ${entityListType === entityTypes.POLICY ? POLICY_FRAGMENT : ''}
    ${entityListType === entityTypes.IMAGE ? IMAGE_FRAGMENT : ''}

    `;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                const { deployment: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.DEPLOYMENT} />;

                if (entityListType) {
                    const listData =
                        entityListType === entityTypes.POLICY
                            ? entity.policyStatus.failingPolicies
                            : getSubListFromEntity(entity, entityListType);

                    return (
                        <EntityList entityListType={entityListType} data={listData} query={query} />
                    );
                }

                const {
                    cluster,
                    updatedAt,
                    type,
                    replicas,
                    labels = [],
                    annotations = [],
                    namespace,
                    namespaceId,
                    serviceAccount,
                    serviceAccountID,
                    imageCount,
                    policyStatus
                } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Updated',
                        value: updatedAt ? format(updatedAt, dateTimeFormat) : 'N/A'
                    },
                    {
                        key: 'Deployment Type',
                        value: type
                    },
                    {
                        key: 'Replicas',
                        value: replicas
                    }
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Deployment Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                />
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.CLUSTER}
                                    entityId={cluster.id}
                                    name="Cluster"
                                    value={cluster.name}
                                />
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.NAMESPACE}
                                    entityId={namespaceId}
                                    name="Namespace"
                                    value={namespace}
                                />
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.SERVICE_ACCOUNT}
                                    name="Service Account"
                                    value={serviceAccount}
                                    entityId={serviceAccountID}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType={entityTypes.IMAGE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Failing Policies"
                                    value={policyStatus.failingPolicies.length}
                                    entityType={entityTypes.POLICY}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Deployment Findings">
                            <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                                <DeploymentFindings
                                    entityContext={entityContext}
                                    deploymentID={id}
                                />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Deployment.propTypes = entityComponentPropTypes;
Deployment.defaultProps = entityComponentDefaultProps;

export default Deployment;
