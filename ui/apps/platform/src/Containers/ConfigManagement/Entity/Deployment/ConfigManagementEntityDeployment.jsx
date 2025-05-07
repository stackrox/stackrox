import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import { getDateTime } from 'utils/dateUtils';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import EntityList from '../../List/EntityList';
import DeploymentFindings from './DeploymentFindings';

const ConfigManagementEntityDeployment = ({
    id,
    entityContext,
    entityListType,
    query,
    pagination,
}) => {
    const searchParam = useContext(searchContext);
    const variables = {
        id,
        query: queryService.objectToWhereClause(query[searchParam]),
        pagination,
    };

    const defaultQuery = gql`
        query getDeployment($id: ID!, $query: String) {
            deployment(id: $id) {
                id
                annotations {
                    key
                    value
                }
                ${entityContext.CLUSTER ? '' : 'cluster { id name}'}
                hostNetwork: id
                imagePullSecrets
                inactive
                labels {
                    key
                    value
                }
                name
                ${entityContext.NAMESPACE ? '' : 'namespace namespaceId'}
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
                ${entityContext.SERVICE_ACCOUNT ? '' : 'serviceAccount serviceAccountID'}
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
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            'DEPLOYMENT',
            entityListType,
            'configmanagement'
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);

        return gql`
            query getDeployment_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                deployment(id: $id) {
                    id
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
                if (!data || !data.deployment) {
                    return <PageNotFound resourceType="DEPLOYMENT" useCase="configmanagement" />;
                }
                const { deployment: entity } = data;

                if (entityListType) {
                    const listData =
                        entityListType === 'POLICY'
                            ? entity.failingPolicies
                            : getSubListFromEntity(entity, entityListType);

                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={listData}
                            totalResults={data?.deployment?.count}
                            query={query}
                            entityContext={{ ...entityContext, DEPLOYMENT: id }}
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
                    secretCount,
                } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Created',
                        value: created ? getDateTime(created) : 'N/A',
                    },
                    {
                        key: 'Deployment Type',
                        value: type,
                    },
                    {
                        key: 'Replicas',
                        value: replicas,
                    },
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
                                        entityType="CLUSTER"
                                        entityId={cluster.id}
                                        name="Cluster"
                                        value={cluster.name}
                                    />
                                )}
                                {namespace && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType="NAMESPACE"
                                        entityId={namespaceId}
                                        name="Namespace"
                                        value={namespace}
                                    />
                                )}
                                {serviceAccount && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType="SERVICE_ACCOUNT"
                                        name="Service Account"
                                        value={serviceAccount}
                                        entityId={serviceAccountID}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType="IMAGE"
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Secrets"
                                    value={secretCount}
                                    entityType="SECRET"
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

ConfigManagementEntityDeployment.propTypes = entityComponentPropTypes;
ConfigManagementEntityDeployment.defaultProps = entityComponentDefaultProps;

export default ConfigManagementEntityDeployment;
