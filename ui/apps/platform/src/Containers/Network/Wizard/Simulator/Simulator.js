import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

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
        <div
            data-testid="network-simulator-panel"
            className="w-full h-full absolute right-0 bottom-0 pt-1 pb-1 pr-1 shadow-md bg-base-200"
        >
            <Panel
                className="border-t-0 border-r-0 border-b-0"
                header="Network Policy Simulator"
                onClose={onCloseHandler}
                closeButtonClassName={`bg-${colorType}-600 hover:bg-${colorType}-700`}
                closeButtonIconColor="text-base-100"
            >
                <ProcessingView />
                <ErrorView />
                <SuccessView />
            </Panel>
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
