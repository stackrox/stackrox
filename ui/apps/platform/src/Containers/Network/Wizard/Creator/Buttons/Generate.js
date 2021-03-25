import React from 'react';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import wizardStages from 'Containers/Network/Wizard/wizardStages';

import { CheckboxWithLabel } from '@stackrox/ui-components';

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
            <CheckboxWithLabel
                id="checkbox-exclude-ports-protocols"
                ariaLabel="Exclude ports and protocols"
                checked={!!excludePortsProtocols}
                onChange={onChangeHandler}
            >
                Exclude Ports & Protocols
            </CheckboxWithLabel>
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
