import React, { Component } from 'react';
import Table from 'Components/Table';
import ReactModal from 'react-modal';
import * as Icon from 'react-feather';
import emitter from 'emitter';
import axios from 'axios';
import dateFns from 'date-fns';

class PolicyAlertsSidePanel extends Component {
    constructor(props) {
        super(props);

        this.state = {
            showPanel: false,
            table: {
                columns: [
                    { key: 'deployment.name', label: 'Deployment' },
                    { key: 'time', label: 'Time' },
                    { key: 'deployment.image.registry', label: 'Registry' }
                ],
                rows: []
            },
            modal: {
                showModal: false,
                data: {}
            }
        };
    }

    componentDidMount() {
        // set up event listeners for this componenet
        this.tableRowSelectedListener = emitter.addListener('PolicyAlertsTable:row-selected', (data) => {
            this.getAlerts(data);
        });
    }

    componentWillUnmount() {
        // remove event listeners
        this.tableRowSelectedListener.remove();
    }

    getAlerts(data) {
        if (!data) {
            this.setState({ showPanel: false, data: {} });
            return;
        }
        this.clearData();
        axios.get('/v1/alerts', {
            params: {
                policy_name: data.name,
                stale: false
            }
        }).then((response) => {
            if (!response.data || !response.data.alerts.length) return;
            const { table } = this.state;
            table.rows = response.data.alerts.map((alert) => {
                const result = Object.assign({}, alert);
                result.policy.category = alert.policy.category.replace('_', ' ').capitalizeFirstLetterOfWord();
                result.policy.imagePolicy.severity = alert.severity.split('_')[0].capitalizeFirstLetterOfWord();
                result.severity = alert.severity.split('_')[0].capitalizeFirstLetterOfWord();
                result.time = dateFns.format(alert.time, 'MM/DD/YYYY HH:MM:ss A');
                return result;
            });
            this.setState({ data, table });
        }).catch(() => {
            this.setState({ data: {} });
        });
    }

    displayHeader() {
        if (!this.state.data) return '';
        return (
            <div className="flex">
                <span className="flex flex-1 self-center text-primary-600 uppercase tracking-wide">Alerts for &quot;{this.state.data.name}&quot;</span>
                <Icon.X className="cursor-pointer h-6 w-6 text-primary-600 hover:text-primary-500" onClick={this.hidePanel} />
            </div>
        );
    }

    displayModalHeader() {
        if (this.state.modal.data === {} || !this.state.modal.data || !this.state.modal.data.deployment) return '';
        return (
            <header className="flex w-full p-3 font-bold border-b border-primary-200 flex-none bg-primary-500">
                <span className="flex flex-1 uppercase self-center text-white">{this.state.modal.data.deployment.name} ({this.state.modal.data.deployment.id})</span>
                <Icon.X className="cursor-pointer h-6 w-6 text-white" onClick={this.handleCloseModal} />
            </header>
        );
    }

    displayModalBody() {
        if (this.state.modal.data === {} || !this.state.modal.data || !this.state.modal.data.deployment) return '';
        return (
            <div className="flex flex-1 overflow-y-scroll">
                <div className="flex flex-col w-1/2 border-r border-primary-200">
                    <div className="bg-white m-3 flex-grow pb-2">
                        <header className="w-full p-3 font-bold border-b border-primary-200 mb-2">Violations</header>
                        <div>
                            {this.state.modal.data.policy.violations.map(violation => <div key={`${violation.message}`} className="py-2 px-3 break-words">{violation.message}</div>)}
                        </div>
                    </div>
                    <div className="bg-white m-3 pb-2">
                        <header className="w-full p-3 border-b border-base-300 font-bold mb-2">Alert Summary</header>
                        <div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Description:</span> {this.state.modal.data.policy.imagePolicy.description}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Deployment ID:</span> {this.state.modal.data.deployment.id}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Severity:</span> {this.state.modal.data.severity}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Time:</span> {this.state.modal.data.time}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Type:</span> {this.state.modal.data.deployment.type}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Last Updated:</span> {this.state.modal.data.deployment.updatedAt}</div>
                        </div>
                    </div>

                </div>
                <div className="flex flex-col w-1/2">
                    <div className="bg-white m-3 pb-2">
                        <header className="w-full p-3 border-b border-primary-200 font-bold mb-2">Image Summary</header>
                        <div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Registry:</span> {this.state.modal.data.deployment.image.registry}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Remote:</span> {this.state.modal.data.deployment.image.remote}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">SHA:</span> {this.state.modal.data.deployment.image.sha}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Tag:</span> {this.state.modal.data.deployment.image.tag}</div>
                        </div>
                    </div>
                    <div className="bg-white m-3 pb-2">
                        <header className="w-full p-3 font-bold border-b border-primary-200 mb-2">Policy Details</header>
                        <div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Name:</span> {this.state.modal.data.policy.name}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Description:</span> {this.state.modal.data.policy.imagePolicy.description}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Severity:</span> {this.state.modal.data.policy.imagePolicy.severity}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Category:</span> {this.state.modal.data.policy.category}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Disabled:</span> {String(this.state.modal.data.policy.imagePolicy.disabled)}</div>
                            <div className="py-2 px-3 truncate"><span className="font-bold text-primary-500">Scan Age Day:</span> {this.state.modal.data.policy.imagePolicy.scanAgeDays}</div>
                        </div>
                    </div>


                </div>
            </div>
        );
    }

    clearData() {
        this.setState({ showPanel: true });
    }

    hidePanel = () => {
        this.setState({ showPanel: false });
    }

    handleOpenModal = (row) => {
        const { modal } = this.state;
        modal.showModal = true;
        modal.data = row;
        this.setState({ modal });
    }

    handleCloseModal = () => {
        const { modal } = this.state;
        modal.showModal = false;
        this.setState({ modal });
    }

    render() {
        return (
            <aside className={`flex-col h-full bg-primary-100 md:w-2/3 border-l border-primary-300 ${(this.state.showPanel) ? 'flex' : ' hidden'}`}>
                <div className="p-3 border-b border-primary-300 w-full">{this.displayHeader()}</div>
                <div className="flex-1 p-3 overflow-y-scroll bg-white rounded-sm shadow">
                    <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.handleOpenModal} />
                </div>
                <ReactModal
                    isOpen={this.state.modal.showModal}
                    onRequestClose={this.handleCloseModal}
                    contentLabel="Modal"
                    ariaHideApp={false}
                    overlayClassName="ReactModal__Overlay react-modal-overlay p-4 flex"
                    // eslint-disable-next-line max-len
                    className="ReactModal__Content w-2/3 mx-auto my-0 flex flex-col self-center bg-primary-100 overflow-hidden max-h-full"
                >
                    {this.displayModalHeader()}
                    {this.displayModalBody()}
                </ReactModal>
            </aside>
        );
    }
}

export default PolicyAlertsSidePanel;
