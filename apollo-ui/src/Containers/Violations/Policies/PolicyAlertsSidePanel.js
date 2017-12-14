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
            policy: {},
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
        }

        this.hidePanel = this.hidePanel.bind(this);
        this.handleOpenModal = this.handleOpenModal.bind(this);
        this.handleCloseModal = this.handleCloseModal.bind(this);
    }

    getAlerts(data) {
        if (!data) {
            this.setState({ showPanel: false, data: {}, alerts:[] }); 
            return;
        }
        this.clearData();
        axios.get('/v1/alerts', {
            params: {
                policy_name: data.name
            }
        }).then((response) => {
            if(!response.data || !response.data.alerts.length) return;
            var table = this.state.table;
            table.rows = response.data.alerts.map((alert) => {
                alert.policy.category = alert.policy.category.replace('_', ' ').capitalizeFirstLetterOfWord();
                alert.policy.imagePolicy.severity = alert.severity.split('_')[0].capitalizeFirstLetterOfWord();
                alert.severity = alert.severity.split('_')[0].capitalizeFirstLetterOfWord();
                alert.time = dateFns.format(alert.time, 'MM/DD/YYYY HH:MM:ss A')
                return alert;
            });
            this.setState({ data: data, table: table });
        }).catch((error) => {
            this.setState({ data: {}, alerts: [] });
        });
    }

    displayHeader() {
        if(!this.state.data) return "";
        return (
            <div className="flex">
                <span className="flex flex-1 self-center text-primary-600 uppercase tracking-wide">Alerts for "{this.state.data.name}"</span>
                <Icon.X className="cursor-pointer h-6 w-6 text-primary-600 hover:text-primary-500" onClick={() => { this.hidePanel() }} />
            </div>
        );
    }

    displayModalHeader() {
        if (this.state.modal.data === {} || !this.state.modal.data || !this.state.modal.data.deployment) return "";
        return (
            <header className="flex w-full p-4 font-bold flex-none">
                <span className="flex flex-1">{this.state.modal.data.deployment.name} ({this.state.modal.data.deployment.id})</span>
                <Icon.X className="cursor-pointer h-6 w-6" onClick={() => { this.handleCloseModal() }} />
            </header>
        );
    }

    displayModalBody() {
        if (this.state.modal.data === {} || !this.state.modal.data || !this.state.modal.data.deployment) return "";
        return (
            <div className="flex flex-1 overflow-y-scroll">
                <div className="flex flex-col w-1/2 border-r border-base-300">
                    <div className="bg-white m-4 flex-1">
                        <header className="w-full p-4 border-b border-base-300 font-bold">Alert Summary</header>
                        <div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Description:</span> {this.state.modal.data.policy.imagePolicy.description}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Deployment ID:</span> {this.state.modal.data.deployment.id}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Severity:</span> {this.state.modal.data.severity}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Time:</span> {this.state.modal.data.time}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Type:</span> {this.state.modal.data.deployment.type}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Last Updated:</span> {this.state.modal.data.deployment.updatedAt}</div>
                        </div>
                    </div>
                    <div className="bg-white m-4 flex-1">
                        <header className="w-full p-4 border-b border-base-300 font-bold">Image Summary</header>
                        <div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Registry:</span> {this.state.modal.data.deployment.image.registry}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Remote:</span> {this.state.modal.data.deployment.image.remote}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">SHA:</span> {this.state.modal.data.deployment.image.sha}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Metadata:</span> {this.state.modal.data.deployment.image.metadata}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Scan:</span> {this.state.modal.data.deployment.image.scan}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Tag:</span> {this.state.modal.data.deployment.image.tag}</div>
                        </div>
                    </div>
                </div>
                <div className="flex flex-col w-1/2">
                    <div className="bg-white m-4 flex-1">
                        <header className="w-full p-4 font-bold">Policy Details</header>
                        <div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Category:</span> {this.state.modal.data.policy.category}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Name:</span> {this.state.modal.data.policy.name}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Description:</span> {this.state.modal.data.policy.imagePolicy.description}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Severity:</span> {this.state.modal.data.policy.imagePolicy.severity}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Disabled:</span> {String(this.state.modal.data.policy.imagePolicy.disabled)}</div>
                            <div className="py-2 px-4 truncate"><span className="font-bold">Scan Age Day:</span> {this.state.modal.data.policy.imagePolicy.scanAgeDays}</div>
                        </div>
                    </div>
                    <div className="bg-white m-4 flex-1">
                        <header className="w-full p-4 font-bold">Violations</header>
                        <div>
                            {this.state.modal.data.policy.violations.map((violation, i) => { return <div key={'policy-alerts-violation-' + i} className="py-2 px-4 break-words">{violation.message}</div>; }) }
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    clearData() {
        this.setState({ showPanel: true, alerts: [] });
    }

    hidePanel() {
        this.setState({ showPanel: false });
    }

    handleOpenModal(row) {
        var modal = this.state.modal;
        modal.showModal = true;
        modal.data = row;
        this.setState({ modal: modal });
    }

    handleCloseModal() {
        var modal = this.state.modal;
        modal.showModal = false;
        this.setState({ modal: modal });
    }

    render() {
        return (
            <aside className={"flex-col h-full bg-primary-100 md:w-2/3 border-l border-primary-300 " + ((this.state.showPanel) ? 'flex' : ' hidden')}>
                <div className="p-3 border-b border-primary-300 w-full">{this.displayHeader()}</div>
                <div className="flex-1 p-3 overflow-y-scroll bg-white rounded-sm shadow">
                    <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.handleOpenModal.bind(this)}></Table>
                </div>
                <ReactModal
                    isOpen={this.state.modal.showModal}
                    onRequestClose={this.handleCloseModal}
                    contentLabel="Modal"
                    ariaHideApp={false}
                    overlayClassName="ReactModal__Overlay react-modal-overlay p-4 flex"
                    className="ReactModal__Content w-2/3 mx-auto my-0 flex flex-col self-center bg-primary-100 overflow-hidden max-h-full">
                    {this.displayModalHeader()}
                    {this.displayModalBody()}
                </ReactModal>
            </aside>
        );
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

}

export default PolicyAlertsSidePanel;
