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
import NamespaceDetailsOverlay from './NamespaceDetails/NamespaceDetailsOverlay';
import ExternalDetailsOverlay from './ExternalDetails/ExternalDetailsOverlay';
import NodesUpdateSection from '../Graph/Overlays/NodesUpdateSection';
import ZoomButtons from '../Graph/Overlays/ZoomButtons';

function SidePanel({ lastUpdatedTimestamp, sidePanelOpen, sidePanelStage, onClose }) {
    if (
        sidePanelOpen &&
        (sidePanelStage === sidepanelStages.details ||
            sidePanelStage === sidepanelStages.externalDetails ||
            sidePanelStage === sidepanelStages.namespaceDetails)
    ) {
        return (
            <div className="network-panel">
                <div className="absolute flex flex-1 max-h-full right-0 w-1/3 min-w-168 max-w-184">
                    {lastUpdatedTimestamp && (
                        <NodesUpdateSection lastUpdatedTimestamp={lastUpdatedTimestamp} />
                    )}
                    {sidePanelStage === sidepanelStages.details && <NetworkDeploymentOverlay />}
                    {sidePanelStage === sidepanelStages.externalDetails && (
                        <ExternalDetailsOverlay />
                    )}
                    {sidePanelStage === sidepanelStages.namespaceDetails && (
                        <NamespaceDetailsOverlay />
                    )}
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
            case sidepanelStages.simulator:
                panelContent = <Simulator onClose={onClose} />;
                break;
            case sidepanelStages.creator:
                panelContent = <Creator onClose={onClose} />;
                break;
            case sidepanelStages.cidrForm:
                panelContent = <CIDRPanel onClose={onClose} />;
                break;
            default:
                return null;
        }
    }

    return (
        <div className={`${width} h-full absolute right-0 bg-base-100 shadow-lg network-panel`}>
            {lastUpdatedTimestamp && (
                <NodesUpdateSection lastUpdatedTimestamp={lastUpdatedTimestamp} />
            )}
            <ZoomButtons pinnedLeft />

            {panelContent}
        </div>
    );
}

SidePanel.propTypes = {
    sidePanelOpen: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
    sidePanelStage: PropTypes.string.isRequired,
    lastUpdatedTimestamp: PropTypes.instanceOf(Date),
};

SidePanel.defaultProps = {
    lastUpdatedTimestamp: null,
};

const mapStateToProps = createStructuredSelector({
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
    sidePanelOpen: selectors.getSidePanelOpen,
    sidePanelStage: selectors.getSidePanelStage,
});

const mapDispatchToProps = {
    onClose: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(SidePanel);
