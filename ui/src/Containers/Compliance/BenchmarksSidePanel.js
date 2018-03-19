import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Panel from 'Components/Panel';

import * as Icon from 'react-feather';

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
        const buttons = [
            {
                renderIcon: () => <Icon.X className="h-4 w-4" />,
                className:
                    'flex py-1 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: this.props.onClose
            }
        ];
        return (
            <Panel header={header} buttons={buttons} width="w-2/3">
                {this.renderTable()}
            </Panel>
        );
    }
}

export default BenchmarksSidePanel;
