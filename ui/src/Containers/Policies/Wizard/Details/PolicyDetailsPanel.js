import React from 'react';
import PropTypes from 'prop-types';

import Panel from 'Components/Panel';
import DetailsButtons from './DetailsButtons';
import PolicyDetails from './PolicyDetails';

function PolicyDetailsPanel({ header, onClose, policy }) {
    return (
        <Panel
            header={header}
            headerComponents={<DetailsButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <PolicyDetails policy={policy} />
        </Panel>
    );
}

PolicyDetailsPanel.propTypes = {
    header: PropTypes.string,
    onClose: PropTypes.func.isRequired,
    policy: PropTypes.shape({}).isRequired,
};

PolicyDetailsPanel.defaultProps = {
    header: '',
};

export default PolicyDetailsPanel;
