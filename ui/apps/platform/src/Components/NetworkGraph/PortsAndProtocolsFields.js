import React from 'react';
import PropTypes from 'prop-types';
import uniq from 'lodash/uniq';

import networkProtocolLabels from 'messages/networkGraph';
import TooltipFieldValue from 'Components/TooltipFieldValue';

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
 * Goes through a list of ports and returns a comma separated list of ports.
 * As a special case, a list containing 0 will be translated to "any".
 *
 * Example: [1, 2, 3, 4] -> "1, 2, 3, 4"
 *          [1, 2, 3, 4, 5, 6, 7, 8] -> "1, 2, 3, 4, 5, +3 more"
 *
 * @param {!Number[]} ports list of ports
 * @returns {!String}
 */
export function getPortsText(ports) {
    if (ports.some((p) => p === 0)) {
        return 'any port';
    }
    const numVisiblePorts = 5;
    if (ports.length <= numVisiblePorts) {
        return ports.join(', ');
    }
    const subsetOfPorts = ports.slice(0, numVisiblePorts);
    const numNonVisiblePorts = ports.slice(numVisiblePorts).length;
    return `${subsetOfPorts.join(', ')}, +${numNonVisiblePorts} more`;
}

const PortsAndProtocolsFields = ({ portsAndProtocols }) => {
    const portsToProtocolsMap = getPortsAndProtocolsMap(portsAndProtocols);
    const portsAndProtocolsFields = Object.keys(portsToProtocolsMap).map((protocol) => {
        const ports = uniq(portsToProtocolsMap[protocol]);

        const portsText = getPortsText(ports);
        return <TooltipFieldValue key={protocol} field={protocol} value={portsText} />;
    });
    return portsAndProtocolsFields;
};

PortsAndProtocolsFields.propTypes = {
    ingressPortsAndProtocols: PropTypes.arrayOf(PropTypes.shape()),
};

PortsAndProtocolsFields.defaultProps = {
    portsAndProtocols: [],
};

export default PortsAndProtocolsFields;
