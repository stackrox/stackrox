import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import queryService from 'modules/queryService';
import { SECRET_FRAGMENT } from 'queries/secret';
import { POLICY_FRAGMENT } from 'queries/policy';

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
        query: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
    query getNamespace($id: ID!) {
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
            deployments {
                ${entityListType === entityTypes.DEPLOYMENT ? '...deploymentFields' : 'id'}

            }
            numDeployments
            numNetworkPolicies
            numSecrets
            imageCount
            policyCount
            images {
                ${entityListType === entityTypes.IMAGE ? '...imageFields' : 'id'}
            }
            secrets {
                ${entityListType === entityTypes.SECRET ? '...secretFields' : 'id'}
            }
            policies {
                ${entityListType === entityTypes.POLICY ? '...policyFields' : 'id'}
            }
        }
    }
    ${entityListType === entityTypes.DEPLOYMENT ? DEPLOYMENT_FRAGMENT : ''}
    ${entityListType === entityTypes.IMAGE ? IMAGE_FRAGMENT : ''}
    ${entityListType === entityTypes.SECRET ? SECRET_FRAGMENT : ''}
    ${entityListType === entityTypes.POLICY ? POLICY_FRAGMENT : ''}


`;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.NAMESPACE} />;
                const {
                    metadata = {},
                    cluster,
                    numDeployments = 0,
                    numSecrets = 0,
                    policyCount = 0,
                    imageCount = 0
                } = entity;

                const { name, creationTime, labels = [] } = metadata;

                const metadataKeyValuePairs = [
                    {
                        key: 'Created',
                        value: creationTime ? format(creationTime, dateTimeFormat) : 'N/A'
                    }
                ];

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={getSubListFromEntity(entity, entityListType)}
                        />
                    );
                }

                const metadataCounts = [{ value: labels.length, text: 'Labels' }];

                return (
                    <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Namespace Details">
                            <div className="flex flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    counts={metadataCounts}
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
                                    value={numDeployments}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Secrets"
                                    value={numSecrets}
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
