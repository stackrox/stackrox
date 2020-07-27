import React from 'react';
import PropTypes from 'prop-types';
import { ArrowRight, ArrowLeft } from 'react-feather';

import networkProtocolLabels from 'messages/networkGraph';
import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipFieldValue from 'Components/TooltipFieldValue';
import TooltipCardSection from 'Components/TooltipCardSection';

/**
 * Goes through a list of ports and protocols and groups them based on
 * the protocol (protocol -> ports)
 *
 * @param {!Object[]} portsAndProtocols list of ports and protocols
 * @returns {!Object{}}
 */
export function getPortsAndProtocolsMap(portsAndProtocols) {
    const portsAndProtocolsMap = {};
    portsAndProtocols.forEach(({ port, protocol }) => {
        const protocolLabel = networkProtocolLabels[protocol];
        if (portsAndProtocolsMap[protocolLabel]) {
            portsAndProtocolsMap[protocolLabel].push(port);
        } else {
            portsAndProtocolsMap[protocolLabel] = [port];
        }
    });
    return portsAndProtocolsMap;
}

/**
 * Goes through a list of ports and returns a comma separated list of ports
 *
 * Example: [1, 2, 3, 4] -> "1, 2, 3, 4"
 *          [1, 2, 3, 4, 5, 6, 7, 8] -> "1, 2, 3, 4, 5, +3 more"
 *
 * @param {!Number[]} ports list of ports
 * @returns {!String}
 */
export function getPortsText(ports) {
    const numVisiblePorts = 5;
    if (ports.length <= numVisiblePorts) return ports.join(', ');
    const subsetOfPorts = ports.slice(0, numVisiblePorts);
    const numNonVisiblePorts = ports.slice(numVisiblePorts).length;
    return `${subsetOfPorts.join(', ')}, +${numNonVisiblePorts} more`;
}

const NetworkTooltipOverlay = ({ node, ingressPortsAndProtocols, egressPortsAndProtocols }) => {
    const { name } = node;
    const ingressPortsToProtocolsMap = getPortsAndProtocolsMap(ingressPortsAndProtocols);
    const egressPortsToProtocolsMap = getPortsAndProtocolsMap(egressPortsAndProtocols);

    const egressPortsAndProtocolsFields = Object.keys(egressPortsToProtocolsMap).map((protocol) => {
        const ports = egressPortsToProtocolsMap[protocol];
        const portsText = getPortsText(ports);
        return <TooltipFieldValue key={protocol} field={protocol} value={portsText} />;
    });

    const ingressPortsAndProtocolsFields = Object.keys(ingressPortsToProtocolsMap).map(
        (protocol) => {
            const ports = ingressPortsToProtocolsMap[protocol];
            const portsText = getPortsText(ports);
            return <TooltipFieldValue key={protocol} field={protocol} value={portsText} />;
        }
    );

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
                            {ingressPortsAndProtocolsFields}
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
                            {egressPortsAndProtocolsFields}
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
