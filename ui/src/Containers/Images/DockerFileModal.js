import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import dateFns from 'date-fns';

import Modal from 'Components/Modal';
import Table from 'Components/Table';

class DockerFileModal extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.shape()).isRequired,
        onClose: PropTypes.func.isRequired
    };

    renderHeader = () => (
        <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
            <span className="flex flex-1 uppercase">Docker File</span>
            <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.props.onClose} />
        </header>
    );

    renderTable = () => {
        const columns = [
            {
                key: 'instruction',
                label: 'Instruction'
            },
            {
                key: 'value',
                label: 'Value'
            },
            {
                key: 'created',
                label: 'Created',
                align: 'right',
                className: 'w-1/5',
                keyValueFunc: timestamp => dateFns.format(timestamp, 'MM/DD/YYYY -- h:mm:ss A')
            }
        ];
        const rows = this.props.data;
        return (
            <div className="flex flex-1 p-3 overflow-y-scroll">
                <div className="flex flex-col w-full">
                    <Table columns={columns} rows={rows} />
                </div>
            </div>
        );
    };

    render() {
        return (
            <Modal isOpen onRequestClose={this.props.onClose} className="w-2/3">
                {this.renderHeader()}
                {this.renderTable()}
            </Modal>
        );
    }
}

export default DockerFileModal;
