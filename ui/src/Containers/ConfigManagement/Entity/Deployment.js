import React from 'react';
import PropTypes from 'prop-types';
import { DEPLOYMENT_QUERY as QUERY } from 'queries/deployment';
import entityTypes from 'constants/entityTypes';
import nsIcon from 'images/ns-icon.svg';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';

const Deployment = ({ id, onRelatedEntityClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { deployment: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.DEPLOYMENT} />;

            const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                onRelatedEntityClick(entityType, entityId);
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
                serviceAccountId
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
                <div className="bg-primary-100 w-full">
                    <CollapsibleSection title="Deployment Details">
                        <div className="flex mb-4 flex-wrap">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Namespace"
                                icon={nsIcon}
                                value={namespace}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.NAMESPACE,
                                    namespaceId
                                )}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Service Account"
                                icon={nsIcon}
                                value={serviceAccount}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.SERVICE_ACCOUNT,
                                    serviceAccountId
                                )}
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Deployment.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired
};

export default Deployment;
