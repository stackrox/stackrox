import React from 'react';
import PropTypes from 'prop-types';
import { ArrowRight, ArrowLeft } from 'react-feather';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipCardSection from 'Components/TooltipCardSection';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

const NetworkTooltipOverlay = ({ node, ingressPortsAndProtocols, egressPortsAndProtocols }) => {
    const { name } = node;
    return (
        <DetailedTooltipOverlay
            title={name}
            body={
                <>
                    <div className="mb-2">
                        <TooltipCardSection
                            header={
                                <div className="flex items-center">
                                    <ArrowRight className="h-4 w-4 text-base-600" />
                                    <span className="ml-1">
                                        {ingressPortsAndProtocols.length} ingress flows
                                    </span>
                                </div>
                            }
                        >
                            {ingressPortsAndProtocols.length !== 0 ? (
                                <PortsAndProtocolsFields
                                    portsAndProtocols={ingressPortsAndProtocols}
                                />
                            ) : (
                                <div>No ports & protocols</div>
                            )}
                        </TooltipCardSection>
                    </div>
                    <div>
                        <TooltipCardSection
                            header={
                                <div className="flex items-center">
                                    <ArrowLeft className="h-4 w-4 text-base-600" />
                                    <span className="ml-1">
                                        {egressPortsAndProtocols.length} egress flows
                                    </span>
                                </div>
                            }
                        >
                            {egressPortsAndProtocols.length !== 0 ? (
                                <PortsAndProtocolsFields
                                    portsAndProtocols={egressPortsAndProtocols}
                                />
                            ) : (
                                <div>No ports & protocols</div>
                            )}
                        </TooltipCardSection>
                    </div>
                </>
            }
        />
    );
};

NetworkTooltipOverlay.propTypes = {
    node: PropTypes.shape({
        name: PropTypes.string.isRequired,
    }).isRequired,
    ingressPortsAndProtocols: PropTypes.arrayOf(PropTypes.shape()),
    egressPortsAndProtocols: PropTypes.arrayOf(PropTypes.shape()),
};

NetworkTooltipOverlay.defaultProps = {
    ingressPortsAndProtocols: [],
    egressPortsAndProtocols: [],
};

export default NetworkTooltipOverlay;
