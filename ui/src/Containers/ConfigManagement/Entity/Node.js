import React from 'react';
import PropTypes from 'prop-types';
import { NODE_QUERY as QUERY } from 'queries/node';
import entityTypes from 'constants/entityTypes';
import cluster from 'images/cluster.svg';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import entityToColumns from 'constants/listColumns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import TableWidget from './TableWidget';

const Node = ({ id, onRelatedEntityClick, onRelatedEntityListClick }) => {
    return (
        <Query query={QUERY} variables={{ id }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { node: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.NODE} />;

                const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                    onRelatedEntityClick(entityType, entityId);
                };

                const onRelatedEntityListClickHandler = entityListType => () => {
                    onRelatedEntityListClick(entityListType);
                };

                const {
                    kernelVersion,
                    osImage,
                    labels = [],
                    containerRuntimeVersion,
                    joinedAt,
                    clusterName,
                    clusterId,
                    annotations,
                    complianceResults = []
                } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'K8s Version',
                        value: kernelVersion
                    },
                    {
                        key: 'Node OS',
                        value: osImage
                    },
                    {
                        key: 'Runtime',
                        value: containerRuntimeVersion
                    },
                    {
                        key: 'Join time',
                        value: joinedAt ? format(joinedAt, dateTimeFormat) : 'N/A'
                    }
                ];
                const metadataCounts = [
                    { value: labels.length, text: 'Labels' },
                    { value: annotations.length, text: 'Annotations' }
                ];

                function onRowClick() {}

                const failedComplianceResults = complianceResults.filter(
                    cr => cr.value.overallState === 'COMPLIANCE_STATE_FAILURE'
                );

                return (
                    <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Node Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    counts={metadataCounts}
                                />
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Cluster"
                                    icon={cluster}
                                    value={clusterName}
                                    onClick={onRelatedEntityClickHandler(
                                        entityTypes.CLUSTER,
                                        clusterId
                                    )}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="CIS Controls"
                                    value={complianceResults.length}
                                    onClick={onRelatedEntityListClickHandler(entityTypes.CONTROL)}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Node Findings">
                            <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                                <TableWidget
                                    header={`${
                                        failedComplianceResults.length
                                    } failed controls accross this node`}
                                    rows={failedComplianceResults}
                                    noDataText="No Controls"
                                    className="bg-base-100"
                                    columns={entityToColumns[entityTypes.CONTROL]}
                                    onRowClick={onRowClick}
                                />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Node.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Node;
