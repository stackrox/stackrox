import React, { ReactElement } from 'react';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import sidepanelStages from 'Containers/Network/SidePanel/sidepanelStages';

import { CheckboxWithLabel } from '@stackrox/ui-components';

type GenerateButtonProps = {
    setSidePanelStage: (stage) => void;
    requestNetworkPolicyModification: () => void;
    excludePortsProtocols: boolean;
    setExcludePortsProtocolsState: (state) => void;
};

function GenerateButton({
    setSidePanelStage,
    requestNetworkPolicyModification,
    excludePortsProtocols,
    setExcludePortsProtocolsState,
}: GenerateButtonProps): ReactElement {
    function onClick() {
        requestNetworkPolicyModification();
        setSidePanelStage(sidepanelStages.simulator);
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
}

const mapStateToProps = createStructuredSelector({
    excludePortsProtocols: selectors.getNetworkPolicyExcludePortsProtocolsState,
});

const mapDispatchToProps = {
    setSidePanelStage: sidepanelActions.setSidePanelStage,
    requestNetworkPolicyModification: sidepanelActions.generateNetworkPolicyModification,
    setExcludePortsProtocolsState: sidepanelActions.setNetworkPolicyExcludePortsProtocolsState,
};

export default connect(mapStateToProps, mapDispatchToProps)(GenerateButton);
