import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Panel from 'Components/Panel';

import * as Icon from 'react-feather';

import axios from 'axios';
import dateFns from 'date-fns';

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_ALERTS':
            return { alerts: nextState.alerts };
        default:
            return prevState;
    }
};

class PolicyAlertsSidePanel extends Component {
    static propTypes = {
        policy: PropTypes.shape({
            name: PropTypes.string,
            id: PropTypes.string
        }).isRequired,
        onClose: PropTypes.func.isRequired,
        onRowClick: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            alerts: []
        };
    }

    componentDidMount() {
        this.getAlerts(this.props.policy);
    }

    componentWillReceiveProps(nextProps) {
        this.getAlerts(nextProps.policy);
    }

    onRowClick = alert => {
        this.props.onRowClick(alert);
    };

    getAlerts = data => {
        axios
            .get('/v1/alerts', {
                params: {
                    policy_name: data.name,
                    stale: false
                }
            })
            .then(response => {
                if (!response.data.alerts.length) return;
                this.update('UPDATE_ALERTS', { alerts: response.data.alerts });
            })
            .catch(error => {
                console.error(error);
            });
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    whitelistDeployment = () => {
        const { id } = this.props.policy;
        axios
            .get(`/v1/policies/${id}`)
            .then(resp => {
                const newPolicy = resp.data;
                newPolicy.whitelists.push({
                    deployment: { name: this.state.alert.deployment.name }
                });
                axios
                    .put(`/v1/policies/${newPolicy.id}`, newPolicy)
                    .then(() => {
                        this.update('CLOSE_MODAL');
                    })
                    .catch(error => {
                        console.error(error);
                        return error;
                    });
            })
            .catch(error => {
                console.error(error);
                return error;
            });
    };

    renderTable = () => {
        const columns = [
            { key: 'deployment.name', label: 'Deployment' },
            { key: 'time', label: 'Time' }
        ];
        const rows = this.state.alerts.map(alert => {
            const result = Object.assign({}, alert);
            result.date = dateFns.format(alert.time, 'MM/DD/YYYY');
            result.time = dateFns.format(alert.time, 'h:mm:ss A');
            return result;
        });
        return <Table columns={columns} rows={rows} onRowClick={this.onRowClick} />;
    };

    render() {
        if (!this.state.alerts) return '';
        const header = `${this.props.policy.name}`;
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

export default PolicyAlertsSidePanel;
