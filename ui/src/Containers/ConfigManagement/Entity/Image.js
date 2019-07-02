import React from 'react';
import PropTypes from 'prop-types';

import { DEPLOYMENTS_WITH_IMAGE } from 'queries/deployment';
import { IMAGE as QUERY } from 'queries/image';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityToColumns } from 'constants/listColumns';
import cloneDeep from 'lodash/cloneDeep';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import CVETable from 'Containers/Images/CVETable';
import TableWidget from './widgets/TableWidget';

const DeploymentsCount = ({ variables, onClick }) => {
    return (
        <Query query={DEPLOYMENTS_WITH_IMAGE} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { deployments } = data;
                return (
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Deployments"
                        value={deployments.length}
                        onClick={onClick}
                    />
                );
            }}
        </Query>
    );
};

DeploymentsCount.propTypes = {
    variables: PropTypes.shape({}).isRequired,
    onClick: PropTypes.func.isRequired
};

const Image = ({ id, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { image: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.IMAGE} />;

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };
            const { lastUpdated, metadata, scan } = entity;

            const metadataKeyValuePairs = [
                {
                    key: 'Last Scanned',
                    value: lastUpdated ? format(lastUpdated, dateTimeFormat) : 'N/A'
                }
            ];
            const metadataCounts = [];

            const variables = {
                query: queryService.objectToWhereClause({
                    'Image Sha': id
                })
            };

            function renderCVEsTable(row) {
                const layer = row.original;
                if (!layer.components || layer.components.length === 0) {
                    return null;
                }
                return (
                    <CVETable
                        scan={layer}
                        containsFixableCVEs={false}
                        className="cve-table my-3 ml-4 px-2 border-0 border-l-4 border-base-300"
                    />
                );
            }

            const layers = cloneDeep(metadata.v1.layers);

            // If we have a scan, then we can try and assume we have layers
            if (scan) {
                layers.forEach((layer, i) => {
                    layers[i].components = [];
                });
                scan.components.forEach(component => {
                    if (component.layerIndex !== undefined) {
                        layers[component.layerIndex].components.push(component);
                    }
                });

                layers.forEach((layer, i) => {
                    layers[i].cvesCount = layer.components.reduce(
                        (cnt, o) => cnt + o.vulns.length,
                        0
                    );
                });
            }

            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Image Details">
                        <div className="flex mb-4 flex-wrap pdf-page">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <DeploymentsCount
                                variables={variables}
                                onClick={onRelatedEntityListClickHandler(entityTypes.DEPLOYMENT)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Dockerfile">
                        <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                            <TableWidget
                                header={`${layers.length} layers accross this image`}
                                rows={layers}
                                noDataText="No Layers"
                                className="bg-base-100"
                                columns={entityToColumns[entityTypes.IMAGE]}
                                SubComponent={renderCVEsTable}
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Image.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Image;
