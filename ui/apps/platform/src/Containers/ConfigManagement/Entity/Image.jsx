import React, { useContext } from 'react';
import { gql } from '@apollo/client';
import { format } from 'date-fns';
import cloneDeep from 'lodash/cloneDeep';

import Query from 'Components/ThrowingQuery';
import NoResultsMessage from 'Components/NoResultsMessage';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityToColumns } from 'constants/listColumns';
import entityTypes from 'constants/entityTypes';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import CVETable from 'Containers/Images/CVETable';
import searchContext from 'Containers/searchContext';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import TableWidget from './widgets/TableWidget';
import EntityList from '../List/EntityList';

const Image = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
    const searchParam = useContext(searchContext);
    const safeImageId = decodeURIComponent(id);

    const variables = {
        id: safeImageId,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Lifecycle Stage': 'DEPLOY',
        }),
        pagination,
    };

    const defaultQuery = gql`
        query getImage($id: ID!${entityListType ? ', $query: String' : ''}) {
            image(id: $id) {
                id
                lastUpdated
                ${entityContext[entityTypes.DEPLOYMENT] ? '' : 'deploymentCount'}
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
    `;

    function getQuery() {
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.IMAGE,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);

        return gql`
            query getImage_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                image(id: $id) {
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
                const { image: entity } = data;
                if (!entity) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.IMAGE}
                            useCase={useCases.CONFIG_MANAGEMENT}
                        />
                    );
                }

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(entity, entityListType)}
                            totalResults={data?.image?.count}
                            entityContext={{ ...entityContext, [entityTypes.IMAGE]: id }}
                            query={query}
                        />
                    );
                }

                const { lastUpdated, metadata, scan, deploymentCount } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Last Scanned',
                        value: lastUpdated ? format(lastUpdated, dateTimeFormat) : 'N/A',
                    },
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
                    scan.components.forEach((component) => {
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
                        <CollapsibleSection title="Image Summary">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                />
                                {deploymentCount && (
                                    <RelatedEntityListCount
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        name="Deployments"
                                        value={deploymentCount}
                                        entityType={entityTypes.DEPLOYMENT}
                                    />
                                )}
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Dockerfile">
                            <div className="flex pdf-page pdf-stretch shadow relative rounded bg-base-100 mb-4 ml-4 mr-4">
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
