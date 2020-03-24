import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import cloneDeep from 'lodash/cloneDeep';
import { Copy, Edit } from 'react-feather';

import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';

function DetailsButtons({ wizardPolicy, setWizardStage, setWizardPolicy }) {
    function goToEdit() {
        setWizardStage(wizardStages.edit);
    }

    function onPolicyClone() {
        const newPolicy = cloneDeep(wizardPolicy);
        newPolicy.id = '';
        newPolicy.name += ' (COPY)';
        setWizardPolicy(newPolicy);
        setWizardStage(wizardStages.edit);
    }

    return (
        <React.Fragment>
            <PanelButton
                icon={<Copy className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={onPolicyClone}
                tooltip="Clone policy"
            >
                Clone
            </PanelButton>
            <PanelButton
                icon={<Edit className="h-4 w-4" />}
                className="btn btn-base"
                onClick={goToEdit}
                tooltip="Edit policy"
            >
                Edit
            </PanelButton>
        </React.Fragment>
    );
}

DetailsButtons.propTypes = {
    wizardPolicy: PropTypes.shape({}).isRequired,

    setWizardStage: PropTypes.func.isRequired,
    setWizardPolicy: PropTypes.func.isRequired
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicy: wizardActions.setWizardPolicy
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(DetailsButtons);
