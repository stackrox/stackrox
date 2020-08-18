import React from 'react';
import PropTypes from 'prop-types';
import { ArrowRight, ArrowLeft } from 'react-feather';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipCardSection from 'Components/TooltipCardSection';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

const NodePortsAndProtocols = ({ portsAndProtocols }) => {
    if (portsAndProtocols.length !== 0) {
        return <PortsAndProtocolsFields portsAndProtocols={portsAndProtocols} />;
    }
    return <div>Ports & Protocols Are Unavailable</div>;
};

// @TODO: Remove "showPortsAndProtocols" when the feature flag "ROX_NETWORK_GRAPH_PORTS" is defaulted to true
const NodeTooltipOverlay = ({
    deploymentName,
    numIngressFlows,
    numEgressFlows,
    ingressPortsAndProtocols,
    egressPortsAndProtocols,
    showPortsAndProtocols,
}) => {
    return (
        <DetailedTooltipOverlay
            title={deploymentName}
            body={
                <>
                    <div className="mb-2">
                        <TooltipCardSection
                            header={
                                <div className="flex items-center">
                                    <ArrowRight className="h-4 w-4 text-base-600" />
                                    <span className="ml-1">{numIngressFlows} ingress flows</span>
                                </div>
                            }
                        >
                            {showPortsAndProtocols && (
                                <NodePortsAndProtocols
                                    portsAndProtocols={ingressPortsAndProtocols}
                                />
                            )}
                        </TooltipCardSection>
                    </div>
                    <div>
                        <TooltipCardSection
                            header={
                                <div className="flex items-center">
                                    <ArrowLeft className="h-4 w-4 text-base-600" />
                                    <span className="ml-1">{numEgressFlows} egress flows</span>
                                </div>
                            }
                        >
                            {showPortsAndProtocols && (
                                <NodePortsAndProtocols
                                    portsAndProtocols={egressPortsAndProtocols}
                                />
                            )}
                        </TooltipCardSection>
                    </div>
                </>
            }
        />
    );
};

NodeTooltipOverlay.propTypes = {
    deploymentName: PropTypes.string.isRequired,
    numIngressFlows: PropTypes.number,
    numEgressFlows: PropTypes.number,
    ingressPortsAndProtocols: PropTypes.arrayOf(PropTypes.shape()),
    egressPortsAndProtocols: PropTypes.arrayOf(PropTypes.shape()),
    showPortsAndProtocols: PropTypes.bool,
};

NodeTooltipOverlay.defaultProps = {
    numIngressFlows: 0,
    numEgressFlows: 0,
    ingressPortsAndProtocols: [],
    egressPortsAndProtocols: [],
    showPortsAndProtocols: false,
};

export default NodeTooltipOverlay;
