import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { types } from 'reducers/policies/backend';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

import DetailsButtons from 'Containers/Policies/Wizard/Details/Buttons';
import EnforcementButtons from 'Containers/Policies/Wizard/Enforcement/Buttons';
import FormButtons from 'Containers/Policies/Wizard/Form/Buttons';
import PreviewButtons from 'Containers/Policies/Wizard/Preview/Buttons';

// Buttons are the buttons along the top of the wizard (like 'Next' or 'Edit').
function Buttons(props) {
    if (props.isFetchingPolicy || props.wizardPolicy == null) return null;

    switch (props.wizardStage) {
        case wizardStages.edit:
        case wizardStages.prepreview:
            return <FormButtons />;
        case wizardStages.preview:
            return <PreviewButtons />;
        case wizardStages.enforcement:
            return <EnforcementButtons />;
        case wizardStages.details:
        default:
            return <DetailsButtons />;
    }
}

Buttons.propTypes = {
    isFetchingPolicy: PropTypes.bool.isRequired,
    wizardPolicy: PropTypes.shape({}),
    wizardStage: PropTypes.string.isRequired
};

const mapStateToProps = createStructuredSelector({
    isFetchingPolicy: state => selectors.getLoadingStatus(state, types.FETCH_POLICY),
    wizardPolicy: selectors.getWizardPolicy,
    wizardStage: selectors.getWizardStage
});

export default connect(mapStateToProps)(Buttons);
