import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import wizardStages from './Wizard/wizardStages';

function SimulationBorder(props) {
    if (!props.wizardOpen || props.wizardStage !== wizardStages.simulator) {
        return null;
    }

    const colorType = props.networkGraphState === 'ERROR' ? 'alert' : 'success';
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
    networkGraphState: PropTypes.string.isRequired
};

const getNetworkGraphState = createSelector(
    [selectors.getNetworkYamlFile, selectors.getNetworkGraphState],
    (yamlFile, networkGraphState) => {
        if (!yamlFile) {
            return 'INITIAL';
        }
        return networkGraphState;
    }
);

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    networkGraphState: getNetworkGraphState
});

export default connect(mapStateToProps)(SimulationBorder);
