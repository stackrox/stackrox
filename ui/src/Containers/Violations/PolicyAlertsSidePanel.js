import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Modal from 'Components/Modal';

import * as Icon from 'react-feather';

import axios from 'axios';
import dateFns from 'date-fns';

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_ALERTS':
            return { alerts: nextState.alerts };
        case 'OPEN_MODAL':
            return { isModalOpen: true, alert: nextState.alert };
        case 'CLOSE_MODAL':
            return { isModalOpen: false, alert: null };
        default:
            return prevState;
    }
};

class PolicyAlertsSidePanel extends Component {
    static propTypes = {
        policy: PropTypes.shape({
            name: PropTypes.string
        }).isRequired,
        onClose: PropTypes.func.isRequired
    }

    constructor(props) {
        super(props);

        this.state = {
            isModalOpen: false,
            alerts: [],
            alert: {},
        };
    }

    componentDidMount() {
        this.getAlerts(this.props.policy);
    }

    componentWillReceiveProps(nextProps) {
        this.getAlerts(nextProps.policy);
    }

    getAlerts = (data) => {
        axios.get('/v1/alerts', {
            params: {
                policy_name: data.name,
                stale: false
            }
        }).then((response) => {
            if (!response.data.alerts.length) return;
            console.log(response.data);
            this.update('UPDATE_ALERTS', { alerts: response.data.alerts });
        }).catch((error) => {
            console.error(error);
        });
    }

    handleOpenModal = (alert) => {
        this.update('OPEN_MODAL', { alert });
    }

    handleCloseModal = () => {
        this.update('CLOSE_MODAL');
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderModal = () => {
        if (!this.state.isModalOpen) return '';
        return (
            <Modal isOpen onRequestClose={this.handleCloseModal} className="w-1/2">
                <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                    <span className="flex flex-1">{this.state.alert.deployment.name} ({this.state.alert.deployment.id})</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.handleCloseModal} />
                </header>
                <div className="flex flex-1 overflow-y-scroll">
                    <div className="flex flex-col w-1/2 border-r border-primary-200">
                        <div className="bg-white m-3 flex-grow pb-2">
                            <header className="w-full p-3 font-bold border-b border-primary-200 mb-2">Violations</header>
                            <div>
                                {this.state.alert.violations.map(violation => <div key={`${violation.message}`} className="py-2 px-3 break-words">{violation.message}</div>)}
                            </div>
                        </div>
                        <div className="bg-white m-3 pb-2">
                            <header className="w-full p-3 border-b border-base-300 font-bold mb-2">Alert Summary</header>
                            <div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Description:</span> {this.state.alert.policy.description}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Deployment ID:</span> {this.state.alert.deployment.id}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Severity:</span> {this.state.alert.policy.severity}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Time:</span> {this.state.alert.time}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Type:</span> {this.state.alert.deployment.type}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Last Updated:</span> {this.state.alert.deployment.updatedAt || 'Never'}</div>
                            </div>
                        </div>

                    </div>
                    <div className="flex flex-col w-1/2">
                        <div className="bg-white m-3 pb-2">
                            <header className="w-full p-3 border-b border-primary-200 font-bold mb-2">Deployment Summary</header>
                            {this.state.alert.deployment.containers.map(container => (
                                <div key={container.image.sha}>
                                    <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Registry:</span> {container.image.registry}</div>
                                    <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Remote:</span> {container.image.remote}</div>
                                    <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">SHA:</span> {container.image.sha}</div>
                                    <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Tag:</span> {container.image.tag}</div>
                                </div>
                            ))}
                        </div>
                        <div className="bg-white m-3 pb-2">
                            <header className="w-full p-3 font-bold border-b border-primary-200 mb-2">Policy Details</header>
                            <div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Name:</span> {this.state.alert.policy.name}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Description:</span> {this.state.alert.policy.description}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Severity:</span> {this.state.alert.policy.severity}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Categories:</span> {this.state.alert.policy.categories.join(', ')}</div>
                                <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Disabled:</span> {String(this.state.alert.policy.disabled)}</div>
                                {
                                    (this.state.alert.policy.imagePolicy) ? <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Scan Age Day:</span> {this.state.alert.policy.imagePolicy.scanAgeDays || 0}</div> : ''
                                }
                            </div>
                        </div>
                    </div>
                </div>
            </Modal>
        );
    }

    renderTable = () => {
        const columns = [
            { key: 'deployment.name', label: 'Deployment' },
            { key: 'time', label: 'Time' }
        ];
        const rows = this.state.alerts.map((alert) => {
            const result = alert;
            result.time = dateFns.format(alert.time, 'MM/DD/YYYY hh:MM:ss A');
            return result;
        });
        return <Table columns={columns} rows={rows} onRowClick={this.handleOpenModal} />;
    }

    render() {
        if (!this.state.alerts) return '';
        const header = `${this.props.policy.name}`;
        const buttons = [
            {
                renderIcon: () => <Icon.X className="h-4 w-4" />,
                className: 'flex py-1 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: this.props.onClose
            }
        ];
        return (
            <Panel header={header} buttons={buttons} width="w-2/3">
                {this.renderTable()}
                {this.renderModal()}
            </Panel>
        );
    }
}

export default PolicyAlertsSidePanel;
