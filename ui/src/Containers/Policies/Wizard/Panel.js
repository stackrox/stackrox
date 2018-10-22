import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { types } from 'reducers/policies/backend';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import Loader from 'Components/Loader';

import DetailsPanel from 'Containers/Policies/Wizard/Details/Panel';
import EnforcementPanel from 'Containers/Policies/Wizard/Enforcement/Panel';
import FormPanel from 'Containers/Policies/Wizard/Form/Panel';
import PreviewPanel from 'Containers/Policies/Wizard/Preview/Panel';

// Panel is the contents of the wizard.
function Panel(props) {
    if (props.isFetchingPolicy || props.wizardPolicy == null) return <Loader />;

    switch (props.wizardStage) {
        case wizardStages.edit:
        case wizardStages.prepreview:
            return <FormPanel />;
        case wizardStages.preview:
            return <PreviewPanel />;
        case wizardStages.enforcement:
            return <EnforcementPanel />;
        case wizardStages.details:
        default:
            return <DetailsPanel />;
    }
}

Panel.propTypes = {
    isFetchingPolicy: PropTypes.bool.isRequired,
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string
    }),
    wizardStage: PropTypes.string.isRequired
};

const mapStateToProps = createStructuredSelector({
    isFetchingPolicy: state => selectors.getLoadingStatus(state, types.FETCH_POLICY),
    wizardPolicy: selectors.getWizardPolicy,
    wizardStage: selectors.getWizardStage
});

export default connect(mapStateToProps)(Panel);
