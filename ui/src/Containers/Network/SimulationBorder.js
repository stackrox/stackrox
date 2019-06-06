import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import wizardStages from './Wizard/wizardStages';

function SimulationBorder({ wizardOpen, wizardStage, modificationState }) {
    if (
        !wizardOpen ||
        (wizardStage !== wizardStages.simulator && wizardStage !== wizardStages.creator)
    ) {
        return null;
    }
    const colorType = modificationState === 'ERROR' ? 'alert' : 'success';
    return (
        <div
            className={`absolute pin-t pin-l bg-${colorType}-600 text-base-100 font-600 uppercase p-2 z-1`}
        >
            Simulation Mode
        </div>
    );
}

SimulationBorder.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    wizardStage: PropTypes.string.isRequired,
    modificationState: PropTypes.string.isRequired
};

const getModificationState = createSelector(
    [selectors.getNetworkPolicyModification, selectors.getNetworkPolicyModificationState],
    (modification, modificationState) => {
        if (!modification) {
            return 'INITIAL';
        }
        return modificationState;
    }
);

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    modificationState: getModificationState
});

export default connect(mapStateToProps)(SimulationBorder);
