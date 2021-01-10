import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { knownBackendFlags } from 'utils/featureFlags';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import wizardStages from './wizardStages';
import NetworkDeploymentOverlay from './NetworkDeploymentOverlay';
import Details from './Details/Details';
import Creator from './Creator/Creator';
import Simulator from './Simulator/Simulator';
import CIDRPanel from './CIDRForm/CIDRPanel';
import NamespaceDetails from './NamespaceDetails/NamespaceDetails';
import ExternalDetails from './ExternalDetails/ExternalDetails';
import ExternalDetailsOverlay from './ExternalDetails/ExternalDetailsOverlay';
import NodesUpdateSection from '../Graph/Overlays/NodesUpdateSection';
import ZoomButtons from '../Graph/Overlays/ZoomButtons';

function Wizard({ wizardOpen, wizardStage, onClose }) {
    const networkDetectionEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_NETWORK_DETECTION);

    if (
        networkDetectionEnabled &&
        wizardOpen &&
        (wizardStage === wizardStages.details || wizardStage === wizardStages.externalDetails)
    ) {
        const paletteComponent =
            wizardStage === wizardStages.details ? (
                <NetworkDeploymentOverlay onClose={onClose} />
            ) : (
                <ExternalDetailsOverlay onClose={onClose} />
            );
        return (
            <div className="network-panel">
                <div className="absolute flex flex-1 max-h-full right-0 w-1/3 min-w-168 max-w-184">
                    <NodesUpdateSection />
                    {paletteComponent}
                </div>
                <div className="absolute h-full right-0">
                    <ZoomButtons pinnedLeft />
                </div>
            </div>
        );
    }

    const width = wizardOpen ? 'md:w-2/3 lg:w-2/5 min-w-208' : 'w-0';
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
            <ZoomButtons pinnedLeft />

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
