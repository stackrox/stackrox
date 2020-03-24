import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import Panel from 'Components/Panel';
import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';
import DetailsButtons from 'Containers/Policies/Wizard/Details/DetailsButtons';

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
            <div className="w-full h-full">
                <div className="flex flex-col w-full overflow-auto pb-5">
                    <Fields policy={wizardPolicy} />
                    <ConfigurationFields policy={wizardPolicy} />
                </div>
            </div>
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
