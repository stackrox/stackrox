import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import queryService from 'modules/queryService';
import { SECRET_FRAGMENT } from 'queries/secret';
import { POLICY_FRAGMENT } from 'queries/policy';
import { SUBJECT_WITH_CLUSTER_FRAGMENT } from 'queries/subject';
import { ROLE_FRAGMENT } from 'queries/role';
import { SERVICE_ACCOUNT_FRAGMENT } from 'queries/serviceAccount';

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
import { IMAGE_FRAGMENT } from 'queries/image';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';
import EntityList from '../List/EntityList';

const Namespace = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Lifecycle Stage': 'DEPLOY'
        })
    };

    const QUERY = gql`
    query getNamespace($id: ID!, $query: String) {
        entity: namespace(id: $id) {
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
            ${
                entityListType === entityTypes.IMAGE
                    ? 'images(query: $query) {...imageFields}'
                    : 'imageCount'
            }
            ${
                entityListType === entityTypes.DEPLOYMENT
                    ? 'deployments(query: $query) { ...deploymentFields }'
                    : 'deploymentCount'
            }
            ${
                entityListType === entityTypes.SUBJECT
                    ? 'subjects {...subjectWithClusterFields}'
                    : 'subjectCount'
            }
            ${
                entityListType === entityTypes.ROLE
                    ? 'k8sroles(query: $query) {...k8roleFields}'
                    : 'k8sroleCount'
            }
            ${
                entityListType === entityTypes.SERVICE_ACCOUNT
                    ? 'serviceAccounts(query: $query) {...serviceAccountFields}'
                    : 'serviceAccountCount'
            }
            ${
                entityListType === entityTypes.SECRET
                    ? 'secrets(query: $query) { ...secretFields }'
                    : 'secretCount'
            }
            ${
                entityListType === entityTypes.POLICY
                    ? 'policies(query: $query) {...policyFields}'
                    : 'policyCount(query: $query)'
            }
        }
    }
    ${entityListType === entityTypes.DEPLOYMENT ? DEPLOYMENT_FRAGMENT : ''}
    ${entityListType === entityTypes.IMAGE ? IMAGE_FRAGMENT : ''}
    ${entityListType === entityTypes.SECRET ? SECRET_FRAGMENT : ''}
    ${entityListType === entityTypes.POLICY ? POLICY_FRAGMENT : ''}
    ${entityListType === entityTypes.SUBJECT ? SUBJECT_WITH_CLUSTER_FRAGMENT : ''}
    ${entityListType === entityTypes.ROLE ? ROLE_FRAGMENT : ''}
    ${entityListType === entityTypes.SERVICE_ACCOUNT ? SERVICE_ACCOUNT_FRAGMENT : ''}
`;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                const { entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.NAMESPACE} />;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={getSubListFromEntity(entity, entityListType)}
                        />
                    );
                }

                const {
                    metadata = {},
                    cluster,
                    deploymentCount,
                    secretCount,
                    policyCount,
                    imageCount
                } = entity;

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
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.CLUSTER}
                                    name="Cluster"
                                    value={cluster.name}
                                    entityId={cluster.id}
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
                                    name="Policies"
                                    value={policyCount}
                                    entityType={entityTypes.POLICY}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Images"
                                    value={imageCount}
                                    entityType={entityTypes.IMAGE}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Namespace Findings">
                            <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                                <DeploymentsWithFailedPolicies
                                    query={queryService.objectToWhereClause({ Namespace: name })}
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
