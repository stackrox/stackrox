import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { types } from 'reducers/policies/backend';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import Loader from 'Components/Loader';
import PolicyDetailsPanel from 'Containers/Policies/Wizard/Details/PolicyDetailsPanel';
import EnforcementPanel from 'Containers/Policies/Wizard/Enforcement/EnforcementPanel';
import PreviewPanel from 'Containers/Policies/Wizard/Preview/PreviewPanel';
import FormPanel from 'Containers/Policies/Wizard/Form/FormPanel';

// Panel is the contents of the wizard.
function Panel({
    isFetchingPolicy,
    wizardPolicy,
    wizardStage,
    initialValues,
    fieldGroups,
    onClose,
}) {
    if (isFetchingPolicy || wizardPolicy == null) return <Loader />;
    const header = wizardPolicy === null ? '' : wizardPolicy.name;

    switch (wizardStage) {
        case wizardStages.edit:
        case wizardStages.prepreview:
            return (
                <FormPanel
                    header={header}
                    onClose={onClose}
                    initialValues={initialValues}
                    fieldGroups={fieldGroups}
                />
            );
        case wizardStages.preview:
            return <PreviewPanel header={header} onClose={onClose} />;
        case wizardStages.enforcement:
            return <EnforcementPanel header={header} onClose={onClose} />;
        case wizardStages.details:
        default:
            return (
                <PolicyDetailsPanel
                    header={header}
                    onClose={onClose}
                    initialValues={initialValues}
                />
            );
    }
}

Panel.propTypes = {
    isFetchingPolicy: PropTypes.bool.isRequired,
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string,
    }),
    wizardStage: PropTypes.string.isRequired,
    initialValues: PropTypes.shape({}),
    fieldGroups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    onClose: PropTypes.func.isRequired,
};

Panel.defaultProps = {
    wizardPolicy: null,
    initialValues: null,
};

const mapStateToProps = createStructuredSelector({
    isFetchingPolicy: (state) => selectors.getLoadingStatus(state, types.FETCH_POLICY),
    wizardPolicy: selectors.getWizardPolicy,
    wizardStage: selectors.getWizardStage,
});

export default connect(mapStateToProps)(Panel);
