import React, { Component } from 'react';
import PropTypes from 'prop-types';
import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/Table';
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
            <Table columns={columns} rows={rows} onRowClick={this.props.onRowClick} minRows="20" />
        );
    };

    render() {
        const header = `Host results for "${this.props.header}"`;
        return (
            <Panel
                header={header}
                onClose={this.props.onClose}
                className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
            >
                <div className="bg-base-100 w-full">{this.renderTable()}</div>
            </Panel>
        );
    }
}

export default BenchmarksSidePanel;
