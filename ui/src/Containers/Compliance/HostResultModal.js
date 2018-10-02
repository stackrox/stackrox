import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import Modal from 'Components/Modal';

class HostResultModal extends Component {
    static propTypes = {
        benchmarkHostResult: PropTypes.shape({
            host: PropTypes.string,
            notes: PropTypes.arrayOf(PropTypes.string)
        }).isRequired,
        onClose: PropTypes.func.isRequired
    };

    renderHeader = () => (
        <header className="bg-primary-600 flex font-700 items-center text-base-100 text-xl uppercase w-full leading-normal">
            <span className="flex flex-1 mx-3">{this.props.benchmarkHostResult.host}</span>
            <button
                className="flex border-l border-primary-700 cursor-pointer h-full hover:bg-primary-700 p-3 text-base-100"
                onClick={this.props.onClose}
            >
                <Icon.X className="cursor-pointer h-6 w-6" />
            </button>
        </header>
    );

    renderBody = () => (
        <div className="flex flex-1 overflow-y-scroll">
            <div className="flex flex-1 flex-col bg-base-100 m-4">
                <header className="border-b border-base-400 font-700 pb-3 mb-3 text-lg w-full">
                    Notes
                </header>
                <ul className="leading-loose px-4">
                    {this.props.benchmarkHostResult.notes.map((note, i) => (
                        <li key={i} className="break-words">
                            {note}
                        </li>
                    ))}
                </ul>
            </div>
        </div>
    );

    render() {
        return (
            <Modal isOpen onRequestClose={this.props.onClose} className="w-1/2">
                {this.renderHeader()}
                {this.renderBody()}
            </Modal>
        );
    }
}

export default HostResultModal;
