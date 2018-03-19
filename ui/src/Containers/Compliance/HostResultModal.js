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
        <header className="flex w-full p-4 font-bold flex-none">
            <span className="flex flex-1">{this.props.benchmarkHostResult.host}</span>
            <Icon.X className="cursor-pointer h-6 w-6" onClick={this.props.onClose} />
        </header>
    );

    renderBody = () => (
        <div className="flex flex-1 overflow-y-scroll">
            <div className="flex flex-1 flex-col bg-white m-4">
                <header className="w-full p-4 font-bold">Notes</header>
                <div>
                    {this.props.benchmarkHostResult.notes.map((note, i) => (
                        <div key={i} className="py-2 px-4 break-words">
                            {note}
                        </div>
                    ))}
                </div>
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
