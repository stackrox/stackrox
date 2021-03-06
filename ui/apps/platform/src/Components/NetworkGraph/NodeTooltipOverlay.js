import React from 'react';
import PropTypes from 'prop-types';
import { Rss } from 'react-feather';

import { DetailedTooltipOverlay } from '@stackrox/ui-components';
import TooltipCardSection from 'Components/TooltipCardSection';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

const NodeTooltipOverlay = ({
    deploymentName,
    numIngressFlows,
    numEgressFlows,
    ingressPortsAndProtocols,
    egressPortsAndProtocols,
    listenPorts,
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
                                    {numIngressFlows} ingress flows
                                </div>
                            }
                        >
                            {ingressPortsAndProtocols.length !== 0 && (
                                <PortsAndProtocolsFields
                                    portsAndProtocols={ingressPortsAndProtocols}
                                />
                            )}
                        </TooltipCardSection>
                    </div>
                    <div className="mb-2">
                        <TooltipCardSection
                            header={
                                <div className="flex items-center">
                                    {numEgressFlows} egress flows
                                </div>
                            }
                        >
                            {egressPortsAndProtocols.length !== 0 && (
                                <PortsAndProtocolsFields
                                    portsAndProtocols={egressPortsAndProtocols}
                                />
                            )}
                        </TooltipCardSection>
                    </div>
                    <div>
                        <TooltipCardSection
                            header={
                                <div className="flex items-center">
                                    <Rss className="h-4 w-4 text-base-600" />
                                    <span className="ml-1">
                                        Listening Ports: {listenPorts.length}
                                    </span>
                                </div>
                            }
                        >
                            {listenPorts && listenPorts.length !== 0 && (
                                <PortsAndProtocolsFields portsAndProtocols={listenPorts} />
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
    listenPorts: PropTypes.arrayOf(PropTypes.shape),
};

NodeTooltipOverlay.defaultProps = {
    numIngressFlows: 0,
    numEgressFlows: 0,
    ingressPortsAndProtocols: [],
    egressPortsAndProtocols: [],
    listenPorts: [],
};

export default NodeTooltipOverlay;
