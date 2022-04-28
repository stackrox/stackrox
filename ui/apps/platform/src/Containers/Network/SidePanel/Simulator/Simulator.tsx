import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import ProcessingView from './ProcessingView';
import SuccessView from './SuccessView';
import ErrorView from './ErrorView';

type SimulatorProps = {
    onClose: () => void;
    setModification: (modification) => void;
    modificationState: string;
    policyGraphState: string;
};

function Simulator({
    onClose,
    setModification,
    modificationState,
    policyGraphState,
}: SimulatorProps) {
    function onCloseHandler() {
        onClose();
        setModification(null);
    }

    const colorType = modificationState === 'ERROR' ? 'alert' : 'success';
    const isProcessing = modificationState === 'REQUEST' && policyGraphState === 'REQUEST';
    const isError = modificationState === 'ERROR' && policyGraphState === 'ERROR';
    const isSuccess = modificationState === 'SUCCESS' && policyGraphState === 'SUCCESS';

    return (
        <div className="w-full h-full absolute right-0 bottom-0 pt-1 pb-1 pr-1 shadow-md bg-base-200">
            <PanelNew testid="network-simulator-panel">
                <PanelHead>
                    <PanelTitle
                        isUpperCase
                        testid="network-simulator-panel-header"
                        text="Network Policy Simulator"
                    />
                    <PanelHeadEnd>
                        <CloseButton
                            onClose={onCloseHandler}
                            className={`bg-${colorType}-600 hover:bg-${colorType}-700`}
                            iconColor="text-base-100"
                        />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    {isProcessing && <ProcessingView />}
                    {isError && <ErrorView />}
                    {isSuccess && <SuccessView />}
                </PanelBody>
            </PanelNew>
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    errorMessage: selectors.getNetworkPolicyErrorMessage,
    modificationState: selectors.getNetworkPolicyModificationState,
    policyGraphState: selectors.getNetworkPolicyGraphState,
});

const mapDispatchToProps = {
    setModification: sidepanelActions.setNetworkPolicyModification,
};

export default connect(mapStateToProps, mapDispatchToProps)(Simulator);
