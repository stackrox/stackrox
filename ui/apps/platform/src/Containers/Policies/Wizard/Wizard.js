import React, { useCallback } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as formMessageActions } from 'reducers/formMessages';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { actions as wizardActions } from 'reducers/policies/wizard';
import SidePanelAdjacentArea from 'Components/SidePanelAdjacentArea';
import WizardPanel from 'Containers/Policies/Wizard/WizardPanel';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import { preFormatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';

// Wizard is the side panel that pops up when you click on a row in the table.
function Wizard({
    wizardPolicy,
    wizardOpen,
    clearFormMessages,
    closeWizard,
    history,
    setWizardPolicy,
    selectPolicyId,
    setWizardStage,
}) {
    const onClose = useCallback(() => {
        clearFormMessages();
        closeWizard();
        setWizardPolicy({ name: '' });
        selectPolicyId('');
        setWizardStage(wizardStages.details);
        history.push({
            pathname: `/main/policies`,
        });
    }, [clearFormMessages, closeWizard, history, setWizardPolicy, selectPolicyId, setWizardStage]);

    if (!wizardOpen) {
        return null;
    }

    const initialValues = wizardPolicy && preFormatPolicyFields(wizardPolicy);

    return (
        <SidePanelAdjacentArea width="1/2">
            <WizardPanel initialValues={initialValues} onClose={onClose} />
        </SidePanelAdjacentArea>
    );
}

Wizard.propTypes = {
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string,
    }),
    wizardOpen: PropTypes.bool.isRequired,
    clearFormMessages: PropTypes.func.isRequired,
    closeWizard: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    setWizardPolicy: PropTypes.func.isRequired,
    selectPolicyId: PropTypes.func.isRequired,
    setWizardStage: PropTypes.func.isRequired,
};

Wizard.defaultProps = {
    wizardPolicy: null,
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy,
    wizardOpen: selectors.getWizardOpen,
});

const mapDispatchToProps = {
    clearFormMessages: formMessageActions.clearFormMessages,
    closeWizard: pageActions.closeWizard,
    selectPolicyId: tableActions.selectPolicyId,
    setWizardPolicy: wizardActions.setWizardPolicy,
    setWizardStage: wizardActions.setWizardStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(Wizard));
