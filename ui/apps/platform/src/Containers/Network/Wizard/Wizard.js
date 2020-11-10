import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import wizardStages from './wizardStages';
import Details from './Details/Details';
import Creator from './Creator/Creator';
import Simulator from './Simulator/Simulator';
import CIDRPanel from './CIDRForm/CIDRPanel';
import NamespaceDetails from './NamespaceDetails/NamespaceDetails';
import ExternalDetails from './ExternalDetails/ExternalDetails';
import NodesUpdateSection from '../Graph/Overlays/NodesUpdateSection';
import ZoomButtons from '../Graph/Overlays/ZoomButtons';

function Wizard({ wizardOpen, wizardStage, onClose }) {
    const width = wizardOpen ? 'md:w-2/3 lg:w-2/5 min-w-144' : 'w-0';
    let panelContent = null;

    if (wizardOpen) {
        switch (wizardStage) {
            case wizardStages.details:
                panelContent = <Details onClose={onClose} />;
                break;
            case wizardStages.simulator:
                panelContent = <Simulator onClose={onClose} />;
                break;
            case wizardStages.creator:
                panelContent = <Creator onClose={onClose} />;
                break;
            case wizardStages.namespaceDetails:
                panelContent = <NamespaceDetails onClose={onClose} />;
                break;
            case wizardStages.externalDetails:
                panelContent = <ExternalDetails onClose={onClose} />;
                break;
            case wizardStages.cidrForm:
                panelContent = <CIDRPanel onClose={onClose} />;
                break;
            default:
                return null;
        }
    }

    return (
        <div
            className={`${width} h-full absolute right-0 bg-primary-200 shadow-lg theme-light network-panel`}
        >
            <NodesUpdateSection />
            <ZoomButtons />

            {panelContent}
        </div>
    );
}

Wizard.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
    wizardStage: PropTypes.string.isRequired,
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
});

const mapDispatchToProps = {
    onClose: pageActions.closeNetworkWizard,
};

export default connect(mapStateToProps, mapDispatchToProps)(Wizard);
