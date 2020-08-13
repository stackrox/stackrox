import React from 'react';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import FeatureEnabled from 'Containers/FeatureEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import wizardStages from '../../wizardStages';

const GenerateButton = ({
    setWizardStage,
    requestNetworkPolicyModification,
    excludePortsProtocols,
    setExcludePortsProtocolsState,
}) => {
    function onClick() {
        requestNetworkPolicyModification();
        setWizardStage(wizardStages.simulator);
    }

    function onChangeHandler() {
        setExcludePortsProtocolsState(!excludePortsProtocols);
    }

    return (
        <>
            <FeatureEnabled featureFlag={knownBackendFlags.ROX_NETWORK_GRAPH_PORTS}>
                {({ featureEnabled }) => {
                    return (
                        featureEnabled && (
                            <div className="flex justify-center">
                                <input
                                    type="checkbox"
                                    data-testid="checkbox-exclude-ports-protocols"
                                    id="exclude-ports-protocols"
                                    checked={!!excludePortsProtocols}
                                    onChange={onChangeHandler}
                                    aria-label="Exclude ports and protocols"
                                />
                                <label htmlFor="exclude-ports-protocols" className="pl-2">
                                    Exclude ports & protocols
                                </label>
                            </div>
                        )
                    );
                }}
            </FeatureEnabled>
            <div className="flex m-3 py-2 items-center justify-center">
                <button
                    type="button"
                    className="rounded-sm px-4 py-3 border-2 border-primary-300 hover:border-primary-400 text-center text-3xlg font-700 text-primary-700 bg-primary-100 hover:bg-primary-200"
                    onClick={onClick}
                >
                    Generate and simulate network policies
                </button>
            </div>
        </>
    );
};

GenerateButton.propTypes = {
    setWizardStage: PropTypes.func.isRequired,
    requestNetworkPolicyModification: PropTypes.func.isRequired,
    excludePortsProtocols: PropTypes.bool.isRequired,
    setExcludePortsProtocolsState: PropTypes.func.isRequired,
};

const mapStateToProps = createStructuredSelector({
    excludePortsProtocols: selectors.getNetworkPolicyExcludePortsProtocolsState,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setNetworkWizardStage,
    requestNetworkPolicyModification: wizardActions.generateNetworkPolicyModification,
    setExcludePortsProtocolsState: wizardActions.setNetworkPolicyExcludePortsProtocolsState,
};

export default connect(mapStateToProps, mapDispatchToProps)(GenerateButton);
