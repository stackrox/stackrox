import React, { Component } from 'react';
import PropTypes from 'prop-types';
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
        const columns = [{ key: 'host', label: 'Host' }, { key: 'result', label: 'Result' }];
        const rows = this.props.hostResults;
        return <Table columns={columns} rows={rows} onRowClick={this.props.onRowClick} />;
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
