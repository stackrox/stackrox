import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as wizardActions } from 'reducers/network/wizard';

import wizardStages from '../../wizardStages';

const GenerateButton = ({ setWizardStage, loadActivePolicies }) => {
    function onClick() {
        loadActivePolicies();
        setWizardStage(wizardStages.simulator);
    }

    return (
        <div className="flex items-center ml-2 -mr-2">
            <button
                type="button"
                className="px-3 py-2 text-xs border-2 border-base-400 bg-base-100 hover:border-primary-400 hover:text-primary-700 font-700 rounded-sm text-center text-base-500 uppercase"
                onClick={onClick}
            >
                View Active YAML
            </button>
        </div>
    );
};

GenerateButton.propTypes = {
    loadActivePolicies: PropTypes.func.isRequired,
    setWizardStage: PropTypes.func.isRequired
};

const mapDispatchToProps = {
    loadActivePolicies: wizardActions.loadActiveNetworkPolicyModification,
    setWizardStage: wizardActions.setNetworkWizardStage
};

export default connect(
    null,
    mapDispatchToProps
)(GenerateButton);
