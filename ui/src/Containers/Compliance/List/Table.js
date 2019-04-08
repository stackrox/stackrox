import React, { Component } from 'react';
import PropTypes from 'prop-types';
import entityTypes, { standardTypes } from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';

import pluralize from 'pluralize';
import toLower from 'lodash/toLower';
import startCase from 'lodash/startCase';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Loader from 'Components/Loader';

import TablePagination from 'Components/TablePagination';
import TableGroup from 'Components/TableGroup';
import entityToColumns from 'constants/tableColumns';
import Query from 'Components/ThrowingQuery';
import NoResultsMessage from 'Components/NoResultsMessage';
import {
    CLUSTERS_LIST_QUERY,
    NAMESPACES_LIST_QUERY,
    NODES_QUERY,
    DEPLOYMENTS_QUERY
} from 'queries/table';
import { LIST_STANDARD } from 'queries/standard';
import queryService from 'modules/queryService';
import orderBy from 'lodash/orderBy';

function getQuery(entityType) {
    if (standardTypes[entityType] || entityType === entityTypes.CONTROL) {
        return LIST_STANDARD;
    }
    switch (entityType) {
        case entityTypes.CLUSTER:
            return CLUSTERS_LIST_QUERY;
        case entityTypes.NAMESPACE:
            return NAMESPACES_LIST_QUERY;
        case entityTypes.NODE:
            return NODES_QUERY;
        case entityTypes.DEPLOYMENT:
            return DEPLOYMENTS_QUERY;
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
        groupBy
    };
}

function complianceRate(numPassing, numFailing) {
    return numPassing + numFailing > 0
        ? `${Math.round((numPassing / (numPassing + numFailing)) * 100)}%`
        : 'N/A';
}

function formatResourceData(data, resourceType) {
    if (!data.results || data.results.results.length === 0) return null;
    const formattedData = { results: [] };
    const entityMap = {};
    let standardKeyIndex = 0;
    let entityKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === 'STANDARD') standardKeyIndex = idx;
        if (scope === resourceType) entityKeyIndex = idx;
    });
    data.results.results.forEach(({ aggregationKeys, keys, numPassing, numFailing }) => {
        const curEntity = aggregationKeys[entityKeyIndex].id;
        const curStandard = aggregationKeys[standardKeyIndex].id;
        const entity = keys[entityKeyIndex];
        // eslint-disable-next-line no-underscore-dangle
        if (entity.__typename === '') return;
        const entityMetaData = entity.metadata || {};

        entityMap[curEntity] = entityMap[curEntity] || {
            name: entity.name || (entity.metadata && entity.metadata.name),
            cluster: entity.clusterName || entityMetaData.clusterName || entity.name,
            namespace: entity.namespace,
            id: curEntity,
            overall: {
                numPassing: 0,
                numFailing: 0,
                average: 0
            }
        };

        if (numPassing + numFailing > 0)
            entityMap[curEntity][curStandard] = complianceRate(numPassing, numFailing);
        entityMap[curEntity].overall.numPassing += numPassing;
        entityMap[curEntity].overall.numFailing += numFailing;
    });

    Object.keys(entityMap).forEach(cluster => {
        const overallCluster = Object.assign({}, entityMap[cluster]);
        const { numPassing, numFailing } = overallCluster.overall;
        overallCluster.overall.average = complianceRate(numPassing, numFailing);
        formattedData.results.push(overallCluster);
    });
    return formattedData;
}

function formatStandardData(data) {
    if (!data.results || data.results.results.length === 0) return null;
    const formattedData = { results: [], totalRows: 0 };
    const groups = {};
    let controlKeyIndex = null;
    let categoryKeyIndex = null;
    let groupByKeyIndex = null;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === 'CONTROL') controlKeyIndex = idx;
        if (scope === 'CATEGORY') categoryKeyIndex = idx;
        if (scope !== 'CATEGORY' && scope !== 'CONTROL') groupByKeyIndex = idx;
    });
    data.results.results
        .filter(datum => datum.numFailing + datum.numPassing)
        .forEach(({ keys, numPassing, numFailing }) => {
            const groupKey = groupByKeyIndex === null ? categoryKeyIndex : groupByKeyIndex;
            const {
                id: standard,
                name,
                clusterName,
                description: groupDescription,
                metadata,
                __typename
            } = keys[groupKey];
            // the check below is to address ROX-1420
            if (__typename !== '') {
                let groupName = name || standardLabels[standard];
                if (__typename === 'Node') {
                    groupName = `${clusterName}/${name}`;
                } else if (__typename === 'Namespace') {
                    groupName = `${metadata.clusterName}/${metadata.name}`;
                }
                if (!groups[groupName]) {
                    const groupId = parseInt(groupName, 10) || groupName;
                    groups[groupName] = {
                        groupId,
                        name: `${groupName} ${groupDescription ? `- ${groupDescription}` : ''}`,
                        rows: []
                    };
                }
                if (controlKeyIndex) {
                    const { id, name: controlName, description, standardId } = keys[
                        controlKeyIndex
                    ];
                    groups[groupName].rows.push({
                        id,
                        description,
                        standardId,
                        standard: standardLabels[standardId],
                        control: controlName,
                        compliance: complianceRate(numPassing, numFailing),
                        group: groupName
                    });
                }
            }
        });
    Object.keys(groups).forEach(group => {
        formattedData.results.push(groups[group]);
        formattedData.totalRows += groups[group].rows.length;
    });
    formattedData.results = orderBy(formattedData.results, ['groupId', 'name'], ['asc', 'asc']);
    return formattedData;
}

const createPDFTable = (tableData, entityType, query, pdfId) => {
    const { standardId } = query;
    const table = document.getElementById('pdf-table');
    const parent = document.getElementById(pdfId);
    if (table) {
        parent.removeChild(table);
    }
    let type = null;
    if (query.groupBy) {
        type = startCase(toLower(query.groupBy));
    } else if (standardId) {
        type = 'Standard';
    }
    const columns = entityToColumns[standardId || entityType];
    if (tableData.length) {
        const headers = columns.map(col => col.Header).filter(header => header !== 'id');
        const headerKeys = columns.map(col => col.accessor).filter(header => header !== 'id');

        if (tableData[0].rows && type) {
            headers.unshift(type);
            headerKeys.unshift(type);
        }
        const tbl = document.createElement('table');
        tbl.style.width = '100%';
        tbl.setAttribute('border', '1');
        const tbdy = document.createElement('tbody');
        const trh = document.createElement('tr');

        headers.forEach(val => {
            const th = document.createElement('th');
            th.appendChild(document.createTextNode(val));
            trh.appendChild(th);
        });
        tbdy.appendChild(trh);
        const addRows = val => {
            const tr = document.createElement('tr');
            headerKeys.forEach(key => {
                const td = document.createElement('td');
                const trimmedStr = val[key] && val[key].replace(/\s+/g, ' ').trim();
                td.appendChild(document.createTextNode(trimmedStr || 'N/A'));
                tr.appendChild(td);
            });
            tbdy.appendChild(tr);
        };
        tableData.forEach(val => {
            if (val.rows) {
                val.rows.forEach(row => {
                    Object.assign(row, { [type]: val.name });
                    addRows(row);
                });
            } else {
                addRows(val);
            }
        });
        tbl.appendChild(tbdy);
        tbl.id = 'pdf-table';
        tbl.className = 'hidden';
        if (parent) parent.appendChild(tbl);
    }
};

class ListTable extends Component {
    static propTypes = {
        entityType: PropTypes.string,
        query: PropTypes.shape({}),
        selectedRow: PropTypes.shape({}),
        updateSelectedRow: PropTypes.func.isRequired,
        pdfId: PropTypes.string
    };

    static defaultProps = {
        selectedRow: null,
        pdfId: null,
        entityType: null,
        query: null
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    setTablePage = page => this.setState({ page });

    // This is a client-side implementation of filtering by the "Compliance State" Search Option
    filterByComplianceState = (data, query, isControlList) => {
        const complianceStateKey = SEARCH_OPTIONS.COMPLIANCE.STATE;
        if (!query[complianceStateKey]) return data.results;
        const val = query[complianceStateKey].toLowerCase();
        const isPassing = val === 'pass';
        const isFailing = val === 'fail';
        const { results } = data;
        if (isControlList) {
            return results
                .map(result => {
                    const newResult = { ...result };
                    newResult.rows = result.rows.filter(row => {
                        const intValue = parseInt(row.compliance, 10); // strValue comes in the format "100.00%"
                        if (Number.isNaN(intValue)) return false;
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
                .filter(result => result.rows.length);
        }
        return results.filter(result => {
            const { id, name, cluster, overall, ...standards } = result;
            return Object.values(standards).reduce((acc, strValue) => {
                const intValue = parseInt(strValue, 10); // strValue comes in the format "100.00%"
                if (isPassing) {
                    if (acc === false) return acc;
                    return intValue === 100;
                }
                if (isFailing) {
                    if (acc === true) return acc;
                    return intValue !== 100;
                }
                return acc;
            }, null);
        });
    };

    getTotalRows = (data, isStandard) => {
        if (!isStandard) {
            return data.length;
        }
        return data.reduce((acc, group) => acc + group.rows.length, 0);
    };

    render() {
        const { entityType, query, selectedRow, updateSelectedRow, pdfId } = this.props;
        const { standardId } = query;
        const { page } = this.state;
        const gqlQuery = getQuery(entityType);
        const variables = getVariables(entityType, query);
        const isControlList = entityType === entityTypes.CONTROL;
        const formatData = isControlList ? formatStandardData : formatResourceData;
        const tableColumns = entityToColumns[standardId || entityType];
        return (
            <Query query={gqlQuery} variables={variables}>
                {({ loading, data }) => {
                    let tableData;
                    let contents = <Loader />;
                    let paginationComponent;
                    let headerText;
                    if (!loading || (data && data.results)) {
                        const formattedData = formatData(data, entityType);
                        if (!formattedData)
                            return (
                                <NoResultsMessage message="No compliance data available. Please run a scan." />
                            );

                        tableData = this.filterByComplianceState(
                            formattedData,
                            query,
                            isControlList
                        );

                        if (tableData.length) {
                            createPDFTable(tableData, entityType, query, pdfId);
                        }
                        const totalRows = this.getTotalRows(tableData, isControlList);
                        const { groupBy } = query;

                        const groupedByText = groupBy
                            ? `across ${tableData.length} ${pluralize(groupBy, tableData.length)}`
                            : '';
                        headerText = `${totalRows} ${pluralize(
                            entityType,
                            totalRows
                        )} ${groupedByText}`;

                        contents = isControlList ? (
                            <TableGroup
                                groups={tableData}
                                totalRows={totalRows}
                                tableColumns={tableColumns}
                                onRowClick={updateSelectedRow}
                                entityType={entityType}
                                idAttribute="id"
                                selectedRowId={selectedRow ? selectedRow.id : null}
                            />
                        ) : (
                            <Table
                                rows={tableData}
                                columns={tableColumns}
                                onRowClick={updateSelectedRow}
                                idAttribute="id"
                                selectedRowId={selectedRow ? selectedRow.id : null}
                                noDataText="No results found. Please refine your search."
                                page={page}
                                defaultSorted={[
                                    {
                                        id: 'name',
                                        desc: false
                                    }
                                ]}
                            />
                        );
                        paginationComponent = (
                            <TablePagination
                                page={page}
                                dataLength={totalRows}
                                setPage={this.setTablePage}
                            />
                        );
                    }
                    return (
                        <Panel header={headerText} headerComponents={paginationComponent}>
                            {contents}
                        </Panel>
                    );
                }}
            </Query>
        );
    }
}

export default ListTable;
