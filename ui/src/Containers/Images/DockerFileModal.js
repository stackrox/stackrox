import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import Table from 'Components/Table';
import Modal from 'Components/Modal';

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
                accessor: 'instruction',
                Header: 'Instruction',
                className: 'text-left pl-3'
            },
            {
                accessor: 'value',
                Header: 'Value',
                widthClassName: 'w-2/5 text-left',
                className: 'w-2/5 text-left pl-3',
                wrap: true
            },
            {
                accessor: 'created',
                Header: 'Created',
                align: 'right',
                widthClassName: 'w-1/5 text-right pr-3',
                className: 'w-1/5 text-right pr-3',
                Cell: ({ original }) => dateFns.format(original.created, dateTimeFormat)
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
            <Modal isOpen onRequestClose={this.props.onClose} className="w-full lg:w-2/3">
                {this.renderHeader()}
                {this.renderTable()}
            </Modal>
        );
    }
}

export default DockerFileModal;
