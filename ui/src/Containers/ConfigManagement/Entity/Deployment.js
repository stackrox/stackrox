import React from 'react';
import PropTypes from 'prop-types';
import { DEPLOYMENT_QUERY as QUERY } from 'queries/deployment';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import FailedPoliciesAcrossDeployment from 'Containers/ConfigManagement/Entity/widgets/FailedPoliciesAcrossDeployment';

const Deployment = ({ id, onRelatedEntityClick, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { deployment: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.DEPLOYMENT} />;

            const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                onRelatedEntityClick(entityType, entityId);
            };

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                updatedAt,
                type,
                replicas,
                labels = [],
                annotations = [],
                namespace,
                namespaceId,
                serviceAccount,
                serviceAccountID,
                secretCount,
                imagesCount
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
            const metadataCounts = [
                { value: labels.length, text: 'Labels' },
                { value: annotations.length, text: 'Annotations' }
            ];

            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Deployment Details">
                        <div className="flex mb-4 flex-wrap pdf-page">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                entityType={entityTypes.NAMESPACE}
                                name="Namespace"
                                value={namespace}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.NAMESPACE,
                                    namespaceId
                                )}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                entityType={entityTypes.SERVICE_ACCOUNT}
                                name="Service Account"
                                value={serviceAccount}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.SERVICE_ACCOUNT,
                                    serviceAccountID
                                )}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Secrets"
                                value={secretCount}
                                onClick={onRelatedEntityListClickHandler(entityTypes.SECRET)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Images"
                                value={imagesCount}
                                onClick={onRelatedEntityListClickHandler(entityTypes.IMAGE)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Deployment Findings">
                        <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                            <FailedPoliciesAcrossDeployment deploymentID={id} />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Deployment.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Deployment;
