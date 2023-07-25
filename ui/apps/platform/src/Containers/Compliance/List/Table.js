import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import orderBy from 'lodash/orderBy';
import { useQuery } from '@apollo/client';
import { Alert } from '@patternfly/react-core';

import entityTypes, { standardTypes } from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import Table from 'Components/Table';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import Loader from 'Components/Loader';
import TablePagination from 'Components/TablePagination';
import TableGroup from 'Components/TableGroup';
import { getColumnsByEntity, getColumnsByStandard } from 'constants/tableColumns';
import Query from 'Components/CacheFirstQuery';
import NoResultsMessage from 'Components/NoResultsMessage';

import createPDFTable from 'utils/pdfUtils';
import { CLUSTERS_QUERY, NAMESPACES_QUERY, NODES_QUERY, DEPLOYMENTS_QUERY } from 'queries/table';
import { LIST_STANDARD, STANDARDS_QUERY } from 'queries/standard';
import queryService from 'utils/queryService';

import { complianceEntityTypes, entityCountNounOrdinaryCase } from '../entitiesForCompliance';

function getQuery(entityType) {
    switch (entityType) {
        case entityTypes.CLUSTER:
            return CLUSTERS_QUERY;
        case entityTypes.NAMESPACE:
            return NAMESPACES_QUERY;
        case entityTypes.NODE:
            return NODES_QUERY;
        case entityTypes.DEPLOYMENT:
            return DEPLOYMENTS_QUERY;
        case entityTypes.CONTROL:
            return LIST_STANDARD;
        default:
            return null;
    }
}

function getVariables(entityType, query) {
    const groupBy =
        entityType === entityTypes.CONTROL
            ? ['CONTROL', 'CATEGORY', ...(query.groupBy ? [query.groupBy] : [])]
            : null;
    return {
        where: queryService.objectToWhereClause(query),
        groupBy,
    };
}

function complianceRate(numPassing, numFailing) {
    return numPassing + numFailing > 0
        ? `${Math.round((numPassing / (numPassing + numFailing)) * 100)}%`
        : 'N/A';
}

function formatResourceData(data, resourceType) {
    if (!data || !data.results || data.results.results.length === 0) {
        return null;
    }
    const formattedData = { results: [] };
    const entityMap = {};
    let standardKeyIndex = 0;
    let entityKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === 'STANDARD') {
            standardKeyIndex = idx;
        }
        if (scope === resourceType) {
            entityKeyIndex = idx;
        }
    });
    data.results.results.forEach(({ aggregationKeys, keys, numPassing, numFailing }) => {
        const curEntity = aggregationKeys[entityKeyIndex].id;
        const curStandard = aggregationKeys[standardKeyIndex].id;
        const entity = keys[entityKeyIndex];
        // eslint-disable-next-line no-underscore-dangle
        if (entity.__typename === '') {
            return;
        }
        const entityMetaData = entity.metadata || {};

        entityMap[curEntity] = entityMap[curEntity] || {
            name: entity?.name || entity?.metadata?.name,
            cluster: entity?.clusterName || entityMetaData?.clusterName || entity?.name,
            namespace: entity?.namespace,
            id: curEntity,
            overall: {
                numPassing: 0,
                numFailing: 0,
                average: 0,
            },
        };

        if (numPassing + numFailing > 0) {
            entityMap[curEntity][curStandard] = complianceRate(numPassing, numFailing);
        }
        entityMap[curEntity].overall.numPassing += numPassing;
        entityMap[curEntity].overall.numFailing += numFailing;
    });

    Object.keys(entityMap).forEach((cluster) => {
        const overallCluster = { ...entityMap[cluster] };
        const { numPassing, numFailing } = overallCluster.overall;
        overallCluster.overall.average = complianceRate(numPassing, numFailing);
        formattedData.results.push(overallCluster);
    });

    return formattedData;
}

function formatStandardData(data) {
    if (!data.results || !data.results.results || data.results.results.length === 0) {
        return null;
    }
    const formattedData = { results: [], totalRows: 0 };
    const groups = {};
    let controlKeyIndex = null;
    let categoryKeyIndex = null;
    let groupByKeyIndex = null;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === 'CONTROL') {
            controlKeyIndex = idx;
        }
        if (scope === 'CATEGORY') {
            categoryKeyIndex = idx;
        }
        if (scope !== 'CATEGORY' && scope !== 'CONTROL') {
            groupByKeyIndex = idx;
        }
    });
    data.results.results.forEach(({ keys, numPassing, numFailing }) => {
        const groupKey = groupByKeyIndex === null ? categoryKeyIndex : groupByKeyIndex;
        const {
            id: standard,
            name,
            clusterName,
            description: groupDescription,
            metadata,
            __typename,
        } = keys[groupKey];
        // the check below is to address ROX-1420
        if (__typename !== '') {
            let groupName = name || standardLabels[standard] || standard;
            if (__typename === 'Node') {
                groupName = `${clusterName}/${name}`;
            } else if (__typename === 'Namespace') {
                groupName = `${metadata?.clusterName}/${metadata?.name}`;
            }
            if (!groups[groupName]) {
                const groupId = parseInt(groupName, 10) || groupName;
                groups[groupName] = {
                    groupId,
                    name: `${groupName} ${groupDescription ? `- ${groupDescription}` : ''}`,
                    rows: [],
                };
            }
            if (controlKeyIndex) {
                const { id, name: controlName, description, standardId } = keys[controlKeyIndex];
                groups[groupName].rows.push({
                    id,
                    description,
                    standardId,
                    standard: standardLabels[standardId],
                    control: controlName,
                    compliance: complianceRate(numPassing, numFailing),
                    group: groupName,
                });
            }
        }
    });
    Object.keys(groups).forEach((group) => {
        formattedData.results.push(groups[group]);
        formattedData.totalRows += groups[group].rows.length;
    });
    formattedData.results = orderBy(formattedData.results, ['groupId', 'name'], ['asc', 'asc']);
    return formattedData;
}

function canFilterByComplianceState(filterQuery, complianceStateKey) {
    return (
        filterQuery[complianceStateKey] &&
        (!Array.isArray(filterQuery[complianceStateKey]) ||
            filterQuery[complianceStateKey].length <= 1)
    );
}

const ListTable = ({
    searchComponent,
    entityType,
    query,
    selectedRowId,
    updateSelectedRow,
    pdfId,
}) => {
    const [page, setPage] = useState(0);
    // This is a client-side implementation of filtering by the "Compliance State" Search Option
    function filterByComplianceState(data, filterQuery, isControlList) {
        const complianceStateKey = SEARCH_OPTIONS.COMPLIANCE.STATE;
        if (!canFilterByComplianceState(filterQuery, complianceStateKey)) {
            return data.results;
        }
        const val = filterQuery[complianceStateKey].toLowerCase();
        const isPassing = val === 'pass';
        const isFailing = val === 'fail';
        const { results } = data;
        if (isControlList) {
            return results
                .map((result) => {
                    const newResult = { ...result };
                    newResult.rows = result.rows.filter((row) => {
                        const intValue = parseInt(row.compliance, 10); // strValue comes in the format "100.00%"
                        if (Number.isNaN(intValue)) {
                            return false;
                        }
                        if (isPassing) {
                            return intValue === 100;
                        }
                        if (isFailing) {
                            return intValue !== 100;
                        }
                        return true;
                    });
                    return newResult;
                })
                .filter((result) => result.rows.length);
        }

        return results.filter((item) =>
            Object.values(standardTypes).reduce((acc, standardId) => {
                if (!item[standardId]) {
                    return acc;
                }
                const intValue = parseInt(item[standardId], 10);
                if (isPassing) {
                    if (acc === false) {
                        return acc;
                    }
                    return intValue === 100;
                }
                if (isFailing) {
                    if (acc === true) {
                        return acc;
                    }
                    return intValue !== 100;
                }
                return acc;
            }, null)
        );
    }

    function getTotalRows(data, isStandard) {
        if (!isStandard) {
            return data.length;
        }
        return data.reduce((acc, group) => acc + group.rows.length, 0);
    }

    const { loading: loadingStandards, data: standardsData } = useQuery(STANDARDS_QUERY);
    if (loadingStandards) {
        return <Loader />;
    }
    const { standardId } = query;
    const gqlQuery = getQuery(entityType);
    const variables = getVariables(entityType, query);
    const isControlList = entityType === entityTypes.CONTROL;
    const formatData = isControlList ? formatStandardData : formatResourceData;
    let tableColumns;
    if (standardId) {
        tableColumns = getColumnsByStandard(standardId);
    } else {
        tableColumns = getColumnsByEntity(entityType, standardsData.results);
    }
    let tableData;

    return (
        <Query query={gqlQuery} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                let headerComponent;
                let headerText;
                let totalRows = 0;

                if (!loading || (data && data.results)) {
                    const formattedData = formatData(data, entityType);
                    if (!formattedData) {
                        headerText = entityCountNounOrdinaryCase(0, entityType);
                        contents = <NoResultsMessage message="No data matched your search." />;
                    } else {
                        tableData = filterByComplianceState(formattedData, query, isControlList);
                        totalRows = getTotalRows(tableData, isControlList);
                        const entityCountNoun = entityCountNounOrdinaryCase(totalRows, entityType);
                        const { groupBy } = query;

                        // Resouces: CLUSTER, NAMESPACE, NODE, DEPLOYMENT.
                        // Or CATEGORY from View Standard link of sunburst graph on dashboard.
                        // Or STANDARD on Controls tab of resource single page.
                        // Otherwise undefined.
                        const { length } = tableData;
                        const groupedByText = groupBy
                            ? ` across ${
                                  complianceEntityTypes.includes(groupBy)
                                      ? entityCountNounOrdinaryCase(length, groupBy)
                                      : `${length} ${pluralize(groupBy.toLowerCase(), length)}`
                              }`
                            : '';
                        headerText = `${entityCountNoun}${groupedByText}`;

                        if (tableData && tableData.length) {
                            createPDFTable(tableData, entityType, query, pdfId, tableColumns);
                        }

                        const tableElement = isControlList ? (
                            <TableGroup
                                groups={tableData}
                                totalRows={totalRows}
                                tableColumns={tableColumns}
                                onRowClick={updateSelectedRow}
                                entityType={entityType}
                                idAttribute="id"
                                selectedRowId={selectedRowId}
                            />
                        ) : (
                            <Table
                                rows={tableData}
                                columns={tableColumns}
                                onRowClick={updateSelectedRow}
                                idAttribute="id"
                                selectedRowId={selectedRowId}
                                noDataText="No results found. Please refine your search."
                                page={page}
                                defaultSorted={[
                                    {
                                        id: 'name',
                                        desc: false,
                                    },
                                ]}
                            />
                        );
                        contents = (
                            <>
                                {data.results.errorMessage && (
                                    <Alert variant="danger" isInline title="Unable to get data">
                                        {data.results.errorMessage}
                                    </Alert>
                                )}
                                {tableElement}
                            </>
                        );
                    }
                    headerComponent = isControlList ? null : (
                        <>
                            <div className="flex flex-1 justify-start">{searchComponent}</div>
                            <TablePagination page={page} dataLength={totalRows} setPage={setPage} />
                        </>
                    );
                }
                return (
                    <PanelNew testid="panel">
                        <PanelHead>
                            <PanelTitle testid="panel-header" text={headerText} />
                            <PanelHeadEnd>{headerComponent}</PanelHeadEnd>
                        </PanelHead>
                        <PanelBody>{contents}</PanelBody>
                    </PanelNew>
                );
            }}
        </Query>
    );
};

ListTable.propTypes = {
    searchComponent: PropTypes.node,
    entityType: PropTypes.string,
    query: PropTypes.shape({
        standardId: PropTypes.string,
        groupBy: PropTypes.string,
    }),
    selectedRowId: PropTypes.string,
    updateSelectedRow: PropTypes.func.isRequired,
    pdfId: PropTypes.string,
};

ListTable.defaultProps = {
    searchComponent: null,
    selectedRowId: null,
    pdfId: null,
    entityType: null,
    query: null,
};

export default ListTable;
