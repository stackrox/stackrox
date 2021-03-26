import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import PropTypes from 'prop-types';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';

import ProcessingView from './ProcessingView';
import SuccessView from './SuccessView';
import ErrorView from './ErrorView';

const Simulator = ({ onClose, setModification, modificationState }) => {
    function onCloseHandler() {
        onClose();
        setModification(null);
    }

    const colorType = modificationState === 'ERROR' ? 'alert' : 'success';

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
                    <ProcessingView />
                    <ErrorView />
                    <SuccessView />
                </PanelBody>
            </PanelNew>
        </div>
    );
};

Simulator.propTypes = {
    onClose: PropTypes.func.isRequired,
    setModification: PropTypes.func.isRequired,
    modificationState: PropTypes.string.isRequired,
};

const mapStateToProps = createStructuredSelector({
    errorMessage: selectors.getNetworkErrorMessage,
    modificationState: selectors.getNetworkPolicyModificationState,
});

const mapDispatchToProps = {
    setModification: wizardActions.setNetworkPolicyModification,
};

export default connect(mapStateToProps, mapDispatchToProps)(Simulator);
