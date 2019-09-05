import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityToColumns } from 'constants/listColumns';
import cloneDeep from 'lodash/cloneDeep';
import { format } from 'date-fns';
import gql from 'graphql-tag';
import Query from 'Components/ThrowingQuery';
import NoResultsMessage from 'Components/NoResultsMessage';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import CVETable from 'Containers/Images/CVETable';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';
import EntityList from '../List/EntityList';
import TableWidget from './widgets/TableWidget';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';

const Image = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Lifecycle Stage': 'DEPLOY'
        })
    };

    const deploymentsQuery = `${
        entityListType === entityTypes.DEPLOYMENT
            ? 'deployments(query: $query) {...deploymentFields}'
            : 'deploymentCount'
    }`;

    const QUERY = gql`
        query image($id: ID!${entityListType ? ', $query: String' : ''}) {
            image(sha: $id) {
                id
                lastUpdated
                ${entityContext[entityTypes.DEPLOYMENT] ? '' : deploymentsQuery}
                metadata {
                    layerShas
                    v1 {
                        created
                        layers {
                            instruction
                            created
                            value
                        }
                    }
                    v2 {
                        digest
                    }
                }
                name {
                    fullName
                    registry
                    remote
                    tag
                }
                scan {
                    components {
                        name
                        layerIndex
                        version
                        license {
                            name
                            type
                            url
                        }
                        vulns {
                            cve
                            cvss
                            link
                            summary
                        }
                    }
                }
            }
        }
        ${entityListType === entityTypes.DEPLOYMENT ? DEPLOYMENT_FRAGMENT : ''}    
    `;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                const { image: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.IMAGE} />;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(entity, entityListType)}
                            entityContext={{ ...entityContext, [entityTypes.IMAGE]: id }}
                            query={query}
                        />
                    );
                }

                const { lastUpdated, metadata, scan, deploymentCount } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Last Scanned',
                        value: lastUpdated ? format(lastUpdated, dateTimeFormat) : 'N/A'
                    }
                ];

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

                const layers = metadata ? cloneDeep(metadata.v1.layers) : [];

                // If we have a scan, then we can try and assume we have layers
                if (scan) {
                    layers.forEach((layer, i) => {
                        layers[i].components = [];
                    });
                    scan.components.forEach(component => {
                        if (component.layerIndex !== undefined && layers[component.layerIndex]) {
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
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Image Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                />
                                {deploymentCount && (
                                    <RelatedEntityListCount
                                        className="mx-4 min-w-48 h-48 mb-4"
                                        name="Deployments"
                                        value={deploymentCount}
                                        entityType={entityTypes.DEPLOYMENT}
                                    />
                                )}
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Dockerfile">
                            <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                                {layers.length === 0 && (
                                    <NoResultsMessage
                                        message="No layers available in this image"
                                        className="p-6"
                                    />
                                )}
                                {layers.length > 0 && (
                                    <TableWidget
                                        header={`${layers.length} layers across this image`}
                                        rows={layers}
                                        noDataText="No Layers"
                                        className="bg-base-100"
                                        columns={entityToColumns[entityTypes.IMAGE]}
                                        SubComponent={renderCVEsTable}
                                        idAttribute="id"
                                    />
                                )}
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Image.propTypes = entityComponentPropTypes;
Image.defaultProps = entityComponentDefaultProps;

export default Image;
