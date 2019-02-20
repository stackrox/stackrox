import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { standardBaseTypes } from 'constants/entityTypes';
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
import componentTypes from 'constants/componentTypes';
import AppQuery from 'Components/AppQuery';
import NoResultsMessage from 'Components/NoResultsMessage';

const createPDFTable = (tableData, params, pdfId) => {
    const { entityType } = params;
    const table = document.getElementById('pdf-table');
    const parent = document.getElementById(pdfId);
    if (table) {
        parent.removeChild(table);
    }
    let type = null;
    if (params.query.groupBy) {
        type = startCase(toLower(params.query.groupBy));
    } else if (standardBaseTypes[params.entityType]) {
        type = 'Standard';
    }
    if (tableData.length) {
        const headers = entityToColumns[entityType]
            .map(col => col.Header)
            .filter(header => header !== 'id');
        const headerKeys = entityToColumns[entityType]
            .map(col => col.accessor)
            .filter(header => header !== 'id');

        if (tableData[0].rows) {
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
        params: PropTypes.shape({}).isRequired,
        selectedRow: PropTypes.shape({}),
        updateSelectedRow: PropTypes.func.isRequired,
        pdfId: PropTypes.string
    };

    static defaultProps = {
        selectedRow: null,
        pdfId: null
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    setTablePage = page => this.setState({ page });

    // This is a client-side implementation of filtering by the "Compliance State" Search Option
    filterByComplianceState = (data, params, isStandard) => {
        const complianceStateKey = SEARCH_OPTIONS.COMPLIANCE.STATE;
        if (!params.query[complianceStateKey]) return data.results;
        const val = params.query[complianceStateKey].toLowerCase();
        const isPassing = val === 'pass';
        const isFailing = val === 'fail';
        const { results } = data;
        if (isStandard) {
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
        const { params, selectedRow, updateSelectedRow, pdfId } = this.props;
        const { page } = this.state;
        return (
            <AppQuery params={params} componentType={componentTypes.LIST_TABLE}>
                {({ loading, data }) => {
                    const isStandard = standardBaseTypes[params.entityType];
                    let tableData;
                    let contents = <Loader />;
                    let paginationComponent;
                    let headerText;
                    if (!loading || (data && data.results)) {
                        if (!data)
                            return (
                                <NoResultsMessage message="No compliance data available. Please run a scan." />
                            );
                        tableData = this.filterByComplianceState(data, params, isStandard);
                        if (tableData.length) {
                            createPDFTable(tableData, params, pdfId);
                        }
                        const totalRows = this.getTotalRows(tableData, isStandard);
                        const { groupBy } = params.query;
                        const groupedByText = groupBy
                            ? `across ${tableData.length} ${pluralize(groupBy, tableData.length)}`
                            : '';
                        const entityType = isStandard ? 'control' : params.entityType;
                        headerText = `${totalRows} ${pluralize(
                            entityType,
                            totalRows
                        )} ${groupedByText}`;
                        contents = isStandard ? (
                            <TableGroup
                                groups={tableData}
                                totalRows={totalRows}
                                tableColumns={entityToColumns[params.entityType]}
                                onRowClick={updateSelectedRow}
                                entityType={entityType}
                                idAttribute="id"
                                selectedRowId={selectedRow ? selectedRow.id : null}
                            />
                        ) : (
                            <Table
                                rows={tableData}
                                columns={entityToColumns[params.entityType]}
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
            </AppQuery>
        );
    }
}

export default ListTable;
