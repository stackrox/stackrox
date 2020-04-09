import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { reduxForm } from 'redux-form';

import { selectors } from 'reducers';
import { knownBackendFlags } from 'utils/featureFlags';
import Panel from 'Components/Panel';
import FormButtons from 'Containers/Policies/Wizard/Form/FormButtons';
import FieldGroupCards from 'Containers/Policies/Wizard/Form/FieldGroupCards';
import FeatureEnabled from 'Containers/FeatureEnabled';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';

function FormPanel({ header, initialValues, fieldGroups, wizardPolicy, onClose }) {
    if (!wizardPolicy) return null;

    return (
        <Panel
            header={header}
            headerComponents={<FormButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <div className="w-full h-full">
                <form className="flex flex-col w-full overflow-auto pb-5">
                    <FieldGroupCards initialValues={initialValues} fieldGroups={fieldGroups} />
                    <FeatureEnabled featureFlag={knownBackendFlags.ROX_BOOLEAN_POLICY_LOGIC}>
                        <BooleanPolicySection />
                    </FeatureEnabled>
                </form>
            </div>
        </Panel>
    );
}

FormPanel.propTypes = {
    header: PropTypes.string,
    initialValues: PropTypes.shape({}).isRequired,
    fieldGroups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string
    }).isRequired,
    onClose: PropTypes.func.isRequired
};

FormPanel.defaultProps = {
    header: ''
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

export default reduxForm({ form: 'policyCreationForm' })(connect(mapStateToProps)(FormPanel));
