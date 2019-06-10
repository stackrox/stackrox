import React from 'react';
import PropTypes from 'prop-types';
import { SECRET as QUERY } from 'queries/secret';
import entityTypes from 'constants/entityTypes';
import clusterIcon from 'images/cluster.svg';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';

const Secret = ({ id, onRelatedEntityClick, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { secret: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.SECRET} />;

            const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                onRelatedEntityClick(entityType, entityId);
            };

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                createdAt,
                labels = [],
                annotations = [],
                deployments = [],
                clusterName,
                clusterId
            } = entity;

            const metadataKeyValuePairs = [
                {
                    key: 'Created',
                    value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A'
                }
            ];
            const metadataCounts = [
                { value: labels.length, text: 'Labels' },
                { value: annotations.length, text: 'Annotations' }
            ];

            return (
                <div className="bg-primary-100 w-full">
                    <CollapsibleSection title="Secret Details">
                        <div className="flex mb-4 flex-wrap">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Cluster"
                                icon={clusterIcon}
                                value={clusterName}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.CLUSTER,
                                    clusterId
                                )}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Deployments"
                                value={deployments.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.DEPLOYMENT)}
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Secret.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Secret;
