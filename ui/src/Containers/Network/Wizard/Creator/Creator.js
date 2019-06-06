import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

import DragAndDrop from './Tiles/DragAndDrop';
import Generate from './Tiles/Generate';
import ViewActive from './Buttons/ViewActive';

import wizardStages from '../wizardStages';

const Creator = ({ onClose, wizardOpen, wizardStage }) => {
    if (!wizardOpen || wizardStage !== wizardStages.creator) {
        return null;
    }

    const header = 'SELECT AN OPTION';
    return (
        <div data-test-id="network-creator-panel" className="h-full w-full shadow-md bg-base-200">
            <Panel header={header} onClose={onClose} buttons={<ViewActive />}>
                <div className="flex h-full w-full flex-col p-4 pb-0">
                    <Generate />
                    <div className="w-full my-5 text-center flex items-center flex-no-shrink">
                        <div className="h-px bg-base-400 w-full" />
                        <span className="relative px-2 font-700">OR</span>
                        <div className="h-px bg-base-400 w-full" />
                    </div>
                    <DragAndDrop />
                </div>
            </Panel>
        </div>
    );
};

Creator.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    wizardStage: PropTypes.string.isRequired,
    onClose: PropTypes.func.isRequired
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage
});

const mapDispatchToProps = {
    onClose: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Creator);
