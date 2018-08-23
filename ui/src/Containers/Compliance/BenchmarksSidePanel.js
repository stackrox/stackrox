import React, { Component } from 'react';
import PropTypes from 'prop-types';
import NoResultsMessage from 'Components/NoResultsMessage';
import ReactRowSelectTable from 'Components/ReactRowSelectTable';
import Panel from 'Components/Panel';

class BenchmarksSidePanel extends Component {
    static propTypes = {
        header: PropTypes.string.isRequired,
        hostResults: PropTypes.arrayOf(
            PropTypes.shape({
                host: PropTypes.string,
                result: PropTypes.string
            })
        ).isRequired,
        onClose: PropTypes.func.isRequired,
        onRowClick: PropTypes.func.isRequired
    };

    renderTable = () => {
        const columns = [
            { accessor: 'host', Header: 'Host' },
            { accessor: 'result', Header: 'Result' }
        ];
        const rows = this.props.hostResults;
        if (!rows.length) return <NoResultsMessage message="No Host Results" />;
        return (
            <ReactRowSelectTable
                columns={columns}
                rows={rows}
                onRowClick={this.props.onRowClick}
                minRows="20"
            />
        );
    };

    render() {
        const header = `Host results for "${this.props.header}"`;
        return (
            <Panel header={header} onClose={this.props.onClose} className="w-2/3">
                {this.renderTable()}
            </Panel>
        );
    }
}

export default BenchmarksSidePanel;
