import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import Panel from 'Components/Panel';
import DetailsButtons from './DetailsButtons';
import PolicyDetails from './PolicyDetails';

function PolicyDetailsPanel({ header, wizardPolicy, onClose }) {
    if (!wizardPolicy) return null;

    return (
        <Panel
            header={header}
            headerComponents={<DetailsButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <PolicyDetails policy={wizardPolicy} />
        </Panel>
    );
}

PolicyDetailsPanel.propTypes = {
    header: PropTypes.string,
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string
    }).isRequired,
    onClose: PropTypes.func.isRequired
};

PolicyDetailsPanel.defaultProps = {
    header: ''
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

export default connect(mapStateToProps)(PolicyDetailsPanel);
