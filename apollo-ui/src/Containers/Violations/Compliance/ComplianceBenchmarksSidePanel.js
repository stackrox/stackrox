import React, { Component } from 'react';
import Table from 'Components/Table';
import ReactModal from 'react-modal';
import * as Icon from 'react-feather';
import emitter from 'emitter';

class ComplianceBenchmarksSidePanel extends Component {
    constructor(props) {
        super(props);

        this.state = {
            showPanel: false,
            data: {},
            table: {
                columns: [
                    { key: 'host', label: 'Host' },
                    { key: 'result', label: 'Result' }
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
        this.tableRowSelectedListener = emitter.addListener('ComplianceTable:row-selected', (data) => {
            this.setData(data);
        });
    }

    componentWillUnmount() {
        // remove event listeners
        this.tableRowSelectedListener.remove();
    }

    setData(data) {
        this.clearData();
        const { state } = this;
        state.data.name = data.definition.name;
        state.table.rows = data.hostResults;
        this.setState({ data: state.data, table: state.table });
    }

    displayHeader() {
        if (!this.state.data) return '';
        return (
            <div className="flex flex-row">
                {/* <span className="font-semibold">Policy: </span>
                <span>{this.state.data.name}</span> */}
                <span className="flex flex-1 self-center text-primary-600 uppercase tracking-wide">Host Results for &quot;{this.state.data.name}&quot;</span>
                <Icon.X className="cursor-pointer h-6 w-6 text-primary-600 hover:text-primary-500" onClick={this.hidePanel} />
            </div>
        );
    }

    displayModalHeader() {
        if (this.state.modal.data === {} || !this.state.modal.data || !this.state.modal.data.host) return '';
        return (
            <header className="flex w-full p-4 font-bold flex-none">
                <span className="flex flex-1">{this.state.modal.data.host}</span>
                <Icon.X className="cursor-pointer h-6 w-6" onClick={this.handleCloseModal} />
            </header>
        );
    }

    displayModalBody() {
        if (this.state.modal.data === {} || !this.state.modal.data || !this.state.modal.data.notes) return '';
        return (
            <div className="flex flex-1 overflow-y-scroll">
                <div className="flex flex-1 flex-col bg-white m-4">
                    <header className="w-full p-4 font-bold">Notes</header>
                    <div>
                        {this.state.modal.data.notes.map((note, i) => <div key={i} className="py-2 px-4 break-words">{note}</div>)}
                    </div>
                </div>
            </div>
        );
    }

    clearData() {
        this.setState({ showPanel: true, hosts: [] });
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
            <aside className={`flex-col h-full bg-primary-100 md:w-2/3 border-l border-primary-300 ${(this.state.showPanel) ? 'flex' : 'hidden'}`}>
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

export default ComplianceBenchmarksSidePanel;
