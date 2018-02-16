import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import Modal from 'Components/Modal';
import Table from 'Components/Table';

import { clusterTypeLabels } from 'messages/common';

class ClustersModal extends Component {
    static propTypes = {
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        clusterType: PropTypes.string.isRequired,
        onRequestClose: PropTypes.func.isRequired
    };

    renderClusters() {
        const { clusters } = this.props;
        if (clusters.length === 0) {
            return <div className="p3 w-full text-center">No Clusters</div>;
        }

        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'preventImage', label: 'StackRox Image' }
        ];
        return <Table columns={columns} rows={clusters} />;
    }

    render() {
        const { clusterType, onRequestClose } = this.props;
        return (
            <Modal isOpen onRequestClose={onRequestClose}>
                <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                    <span className="flex flex-1">{clusterTypeLabels[clusterType]}</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={onRequestClose} />
                </header>
                <div className="p-4">{this.renderClusters()}</div>
            </Modal>
        );
    }
}

export default ClustersModal;
