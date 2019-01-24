import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { horizontalBarData, sunburstData, sunburstLegendData } from 'mockData/graphDataMock';
import entityTypes from 'constants/entityTypes';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import TableGroup from 'Components/TableGroup';
import {
    groupedData,
    subTableColumns,
    tableData as data,
    tableColumns as columns
} from 'mockData/tableDataMock';
import Widget from 'Components/Widget';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import Sunburst from 'Components/visuals/Sunburst';
// import ComplianceEntityPage from 'Containers/Compliance2/Entity/Page';
import Header from './Header';

const entity = 'control';
class ComplianceListPage extends Component {
    static propTypes = {
        // data: PropTypes.arrayOf(PropTypes.shape({})),
        grouped: PropTypes.bool
        // entityType: PropTypes.string.isRequired
    };

    static defaultProps = {
        // data: [],
        grouped: true
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0,
            selectedRow: null
        };
    }

    updateSelectedRow = selectedRow => this.setState({ selectedRow });

    clearSelectedRow = () => {
        this.setState({ selectedRow: null });
    };

    renderSidePanel = () => {
        const { selectedRow } = this.state;
        if (!selectedRow) return '';
        return (
            <Panel header={selectedRow.node || selectedRow.control} onClose={this.clearSelectedRow}>
                {/* <ComplianceEntityPage /> */}
            </Panel>
        );
    };

    setTablePage = page => this.setState({ page });

    renderTable = () => {
        const { grouped } = this.props;
        const { selectedRow, page } = this.state;
        return grouped ? (
            <TableGroup
                groups={groupedData}
                tableColumns={subTableColumns}
                onRowClick={this.updateSelectedRow}
                idAttribute={entity}
                selectedRowId={selectedRow ? selectedRow[entity] : null}
            />
        ) : (
            <Table
                rows={data}
                columns={columns}
                onRowClick={this.updateSelectedRow}
                idAttribute="node"
                selectedRowId={selectedRow ? selectedRow.node : null}
                noDataText="No results found. Please refine your search."
                page={page}
            />
        );
    };

    render() {
        // const { data } = this.props;
        const { page } = this.state;
        const paginationComponent = (
            <TablePagination page={page} dataLength={data.length} setPage={this.setTablePage} />
        );
        return (
            <section className="flex flex-col h-full">
                <Header />
                <CollapsibleBanner>
                    <StandardsAcrossEntity type={entityTypes.CLUSTERS} data={horizontalBarData} />
                    <StandardsByEntity type={entityTypes.CLUSTERS} />
                    <Widget header="Compliance Across Controls" className="bg-base-100">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                            containerProps={{
                                style: {
                                    borderRight: '1px solid var(--base-300)'
                                }
                            }}
                        />
                    </Widget>
                    <StandardsAcrossEntity type={entityTypes.NAMESPACES} data={horizontalBarData} />
                    <StandardsAcrossEntity type={entityTypes.NODES} data={horizontalBarData} />
                </CollapsibleBanner>
                <div className="flex">
                    <Panel header={entityTypes.NODES} headerComponents={paginationComponent}>
                        {this.renderTable()}
                    </Panel>
                    {this.renderSidePanel()}
                </div>
            </section>
        );
    }
}

export default ComplianceListPage;
