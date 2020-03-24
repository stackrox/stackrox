import React, { useCallback } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { actions as wizardActions } from 'reducers/policies/wizard';
import WizardPanel from 'Containers/Policies/Wizard/WizardPanel';
import {
    policyStatus,
    policyDetails,
    policyConfiguration
} from 'Containers/Policies/Wizard/Form/descriptors';
import { clientOnlyWhitelistFieldNames } from 'Containers/Policies/Wizard/Form/whitelistFieldNames';
import { preFormatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';

// Wizard is the side panel that pops up when you click on a row in the table.
function Wizard({
    wizardPolicy,
    wizardOpen,
    closeWizard,
    setWizardPolicy,
    selectPolicyId,
    fieldGroups
}) {
    const onClose = useCallback(
        () => {
            closeWizard();
            setWizardPolicy({ name: '' });
            selectPolicyId('');
        },
        [closeWizard, setWizardPolicy, selectPolicyId]
    );

    if (!wizardOpen) return null;

    const initialValues = wizardPolicy && preFormatPolicyFields(wizardPolicy);

    return (
        <div className="w-full">
            <WizardPanel
                initialValues={initialValues}
                fieldGroups={fieldGroups}
                onClose={onClose}
            />
        </div>
    );
}

Wizard.propTypes = {
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string
    }),
    wizardOpen: PropTypes.bool.isRequired,
    closeWizard: PropTypes.func.isRequired,
    setWizardPolicy: PropTypes.func.isRequired,
    selectPolicyId: PropTypes.func.isRequired,
    fieldGroups: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

Wizard.defaultProps = {
    wizardPolicy: null
};

const getFieldGroups = createSelector(
    [selectors.getNotifiers, selectors.getImages, selectors.getPolicyCategories],
    (notifiers, images, policyCategories) => {
        const { descriptor } = policyDetails;
        const policyDetailsFormFields = descriptor.map(field => {
            const newField = Object.assign({}, field);
            let { options } = newField;
            switch (field.jsonpath) {
                case 'categories':
                    options = policyCategories.map(category => ({
                        label: category,
                        value: category
                    }));
                    break;
                case clientOnlyWhitelistFieldNames.WHITELISTED_IMAGE_NAMES:
                    options = images.map(image => ({
                        label: image.name,
                        value: image.name
                    }));
                    break;
                case 'notifiers':
                    options = notifiers.map(notifier => ({
                        label: notifier.name,
                        value: notifier.id
                    }));
                    break;
                default:
                    break;
            }
            newField.options = options;
            return newField;
        });
        policyDetails.descriptor = policyDetailsFormFields;
        return [policyStatus, policyDetails, policyConfiguration];
    }
);

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy,
    wizardOpen: selectors.getWizardOpen,
    fieldGroups: getFieldGroups
});

const mapDispatchToProps = {
    closeWizard: pageActions.closeWizard,
    selectPolicyId: tableActions.selectPolicyId,
    setWizardPolicy: wizardActions.setWizardPolicy
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Wizard);
