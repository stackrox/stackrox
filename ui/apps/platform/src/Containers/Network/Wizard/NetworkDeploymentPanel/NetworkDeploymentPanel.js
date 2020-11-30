import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { withRouter } from 'react-router-dom';

import { selectors } from 'reducers';

function NetworkDeploymentPanel({ selectedDeployment }) {
    const { name, type } = selectedDeployment;

    // TODO: This is just a placeholder. We'll use a floating panel component here
    return (
        <div className="relative right-0 top-0">
            <div className="bg-primary-800 flex items-center m-2 min-w-108 p-3 rounded-lg shadow text-primary-100">
                <div className="flex flex-1 flex-col">
                    <div>{name}</div>
                    <div className="italic text-primary-200 text-xs capitalize">
                        {type.toLowerCase()}
                    </div>
                </div>
                <ul className="flex ml-8 items-center text-sm uppercase font-700">
                    <li className="mr-2">
                        <div className="bg-primary-500 border-2 border-primary-400 leading-none p-1 px-2 rounded-full">
                            Flows
                        </div>
                    </li>
                    <li className="mr-2">Policies</li>
                    <li>Details</li>
                </ul>
            </div>
            <div className="bg-primary-100 border border-primary-300 m-2 shadow-md">
                <div className="p-2">Add stuff here...</div>
            </div>
        </div>
    );
}

NetworkDeploymentPanel.propTypes = {
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

export default withRouter(connect(mapStateToProps, null)(NetworkDeploymentPanel));
