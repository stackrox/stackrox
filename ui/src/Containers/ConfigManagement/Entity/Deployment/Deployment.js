import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import isGQLLoading from 'utils/gqlLoading';
import gql from 'graphql-tag';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import useCases from 'constants/useCaseTypes';
import EntityList from '../../List/EntityList';
import DeploymentFindings from './DeploymentFindings';

const Deployment = ({ id, entityContext, entityListType, query }) => {
    const searchParam = useContext(searchContext);
    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        id,
        query: queryService.objectToWhereClause(query[searchParam])
    };

    const defaultQuery = gql`
        query getDeployment($id: ID!, $query: String) {
            deployment(id: $id) {
                id
                annotations {
                    key
                    value
                }
                ${entityContext[entityTypes.CLUSTER] ? '' : 'cluster { id name}'}
                hostNetwork: id
                imagePullSecrets
                inactive
                labels {
                    key
                    value
                }
                name
                ${entityContext[entityTypes.NAMESPACE] ? '' : 'namespace namespaceId'}
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
                ${
                    entityContext[entityTypes.SERVICE_ACCOUNT]
                        ? ''
                        : 'serviceAccount serviceAccountID'
                }
                failingPolicyCount(query: $query)
            
                tolerations {
                    key
                    operator
                    taintEffect
                    value
                }
                type
                created
                secretCount
                imageCount
            }
        }
    `;

    function getQuery() {
        if (!entityListType) return defaultQuery;
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.DEPLOYMENT,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );

        return gql`
            query getDeployment${entityListType}($id: ID!, $query: String) {
                deployment(id: $id) {
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
                if (isGQLLoading(loading, data)) return <Loader />;
                if (!data || !data.deployment)
                    return <PageNotFound resourceType={entityTypes.DEPLOYMENT} />;
                const { deployment: entity } = data;

                if (entityListType) {
                    const listData =
                        entityListType === entityTypes.POLICY
                            ? entity.failingPolicies
                            : getSubListFromEntity(entity, entityListType);

                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={listData}
                            query={query}
                            entityContext={{ ...entityContext, [entityTypes.DEPLOYMENT]: id }}
                        />
                    );
                }

                const {
                    cluster,
                    created,
                    type,
                    replicas,
                    labels = [],
                    annotations = [],
                    namespace,
                    namespaceId,
                    serviceAccount,
                    serviceAccountID,
                    imageCount,
                    secretCount
                } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Created',
                        value: created ? format(created, dateTimeFormat) : 'N/A'
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
                        <CollapsibleSection title="Deployment Summary">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                />
                                {cluster && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.CLUSTER}
                                        entityId={cluster.id}
                                        name="Cluster"
                                        value={cluster.name}
                                    />
                                )}
                                {namespace && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.NAMESPACE}
                                        entityId={namespaceId}
                                        name="Namespace"
                                        value={namespace}
                                    />
                                )}
                                {serviceAccount && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.SERVICE_ACCOUNT}
                                        name="Service Account"
                                        value={serviceAccount}
                                        entityId={serviceAccountID}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType={entityTypes.IMAGE}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Secrets"
                                    value={secretCount}
                                    entityType={entityTypes.SECRET}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Deployment Findings">
                            <div className="flex mb-4 pdf-page pdf-stretch">
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
