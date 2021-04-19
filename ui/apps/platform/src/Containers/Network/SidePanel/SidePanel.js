import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import sidepanelStages from './sidepanelStages';
import NetworkDeploymentOverlay from './NetworkDeploymentOverlay';
import Creator from './Creator/Creator';
import Simulator from './Simulator/Simulator';
import CIDRPanel from './CIDRForm/CIDRPanel';
import NamespaceDetails from './NamespaceDetails/NamespaceDetails';
import ExternalDetailsOverlay from './ExternalDetails/ExternalDetailsOverlay';
import NodesUpdateSection from '../Graph/Overlays/NodesUpdateSection';
import ZoomButtons from '../Graph/Overlays/ZoomButtons';

function SidePanel({ sidePanelOpen, sidePanelStage, onClose }) {
    if (
        sidePanelOpen &&
        (sidePanelStage === sidepanelStages.details ||
            sidePanelStage === sidepanelStages.externalDetails)
    ) {
        const paletteComponent =
            sidePanelStage === sidepanelStages.details ? (
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

    const width = sidePanelOpen ? 'md:w-2/3 lg:w-2/5 min-w-208' : 'w-0';
    let panelContent = null;

    if (sidePanelOpen) {
        switch (sidePanelStage) {
            case sidepanelStages.details:
                return null; // supserseded by NetworkDeploymentOverlay
            case sidepanelStages.simulator:
                panelContent = <Simulator onClose={onClose} />;
                break;
            case sidepanelStages.creator:
                panelContent = <Creator onClose={onClose} />;
                break;
            case sidepanelStages.namespaceDetails:
                panelContent = <NamespaceDetails onClose={onClose} />;
                break;
            case sidepanelStages.externalDetails:
                return null; // superseded by ExternalDetailsOverlay
            case sidepanelStages.cidrForm:
                panelContent = <CIDRPanel onClose={onClose} />;
                break;
            default:
                return null;
        }
    }

    return (
        <div className={`${width} h-full absolute right-0 bg-base-100 shadow-lg network-panel`}>
            <NodesUpdateSection />
            <ZoomButtons pinnedLeft />

            {panelContent}
        </div>
    );
}

SidePanel.propTypes = {
    sidePanelOpen: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
    sidePanelStage: PropTypes.string.isRequired,
};

const mapStateToProps = createStructuredSelector({
    sidePanelOpen: selectors.getSidePanelOpen,
    sidePanelStage: selectors.getSidePanelStage,
});

const mapDispatchToProps = {
    onClose: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(SidePanel);
