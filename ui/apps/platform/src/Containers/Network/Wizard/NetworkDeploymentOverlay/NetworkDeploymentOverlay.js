import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { withRouter } from 'react-router-dom';

import { selectors } from 'reducers';

import Tab from 'Components/Tab';
import NetworkEntityTabbedOverlay from 'Components/NetworkEntityTabbedOverlay';

function NetworkDeploymentOverlay({ selectedDeployment }) {
    const { name, type } = selectedDeployment;

    return (
        <div className="flex flex-1 flex-col">
            <NetworkEntityTabbedOverlay entityName={name} entityType={type}>
                <Tab title="Flows">
                    <div className="p-4 bg-primary-100">Add Flows here...</div>
                </Tab>
                <Tab title="Policies">
                    <div className="p-4 bg-primary-100">Add Policies here...</div>
                </Tab>
                <Tab title="Details">
                    <div className="p-4 bg-primary-100">Add Details here...</div>
                </Tab>
            </NetworkEntityTabbedOverlay>
        </div>
    );
}

NetworkDeploymentOverlay.propTypes = {
    selectedDeployment: PropTypes.shape({
        id: PropTypes.string.isRequired,
        name: PropTypes.string.isRequired,
        type: PropTypes.string.isRequired,
        edges: PropTypes.arrayOf(PropTypes.shape({})),
    }).isRequired,
};

const mapStateToProps = createStructuredSelector({
    selectedDeployment: selectors.getSelectedNode,
});

export default withRouter(connect(mapStateToProps, null)(NetworkDeploymentOverlay));
