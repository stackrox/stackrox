import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Panel from 'Components/Panel';

import * as Icon from 'react-feather';

import dateFns from 'date-fns';
import { sortTime } from 'sorters/sorters';

class PolicyAlertsSidePanel extends Component {
    static propTypes = {
        header: PropTypes.string.isRequired,
        alerts: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        onClose: PropTypes.func.isRequired,
        onRowClick: PropTypes.func.isRequired
    };

    renderTable = () => {
        const columns = [
            { key: 'deployment.name', label: 'Deployment' },
            { key: 'deployment.clusterName', label: 'Cluster' },
            { key: 'time', label: 'Time', sortMethod: sortTime }
        ];
        const rows = this.props.alerts.map(alert => {
            const result = Object.assign({}, alert);
            result.date = dateFns.format(alert.time, 'MM/DD/YYYY');
            result.time = dateFns.format(alert.time, 'h:mm:ss A');
            return result;
        });
        return <Table columns={columns} rows={rows} onRowClick={this.props.onRowClick} />;
    };

    render() {
        const buttons = [
            {
                renderIcon: () => <Icon.X className="h-4 w-4" />,
                className:
                    'flex py-1 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: this.props.onClose
            }
        ];
        return (
            <Panel header={this.props.header} buttons={buttons} width="w-2/3">
                {this.renderTable()}
            </Panel>
        );
    }
}

export default PolicyAlertsSidePanel;
