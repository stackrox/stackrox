import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';

import deployment from 'images/legend-icons/deployment.svg';
import deploymentAllowedConnections from 'images/legend-icons/deployment-allowed-connections.svg';
import deploymentActiveConnection from 'images/legend-icons/deployment-active-connection.svg';
import deploymentAllowedConnection from 'images/legend-icons/deployment-allowed-connection.svg';
import namespace from 'images/legend-icons/namespace.svg';
import namespaceAllowed from 'images/legend-icons/namespace-allowed.svg';
import namespaceConnection from 'images/legend-icons/namespace-connection.svg';
import namespaceEgress from 'images/legend-icons/namespace-egress.svg';
import namespaceIngress from 'images/legend-icons/namespace-ingress.svg';
import namespaceEgressIngress from 'images/legend-icons/namespace-egress-ingress.svg';

const svgMapping = {
    deployment,
    'deployment-allowed-connections': deploymentAllowedConnections,
    'active-connection': deploymentActiveConnection,
    'allowed-connection': deploymentAllowedConnection,
    namespace,
    'namespace-allowed-connection': namespaceAllowed,
    'namespace-connection': namespaceConnection,
    'namespace-egress': namespaceEgress,
    'namespace-ingress': namespaceIngress,
    'namespace-egress-ingress': namespaceEgressIngress
};

const LegendTile = ({ svgName, tooltip }) => (
    <Tooltip
        placement="top"
        overlay={<div>{tooltip}</div>}
        mouseLeaveDelay={0}
        className="flex items-center justify-center bg-base-100"
    >
        <div className="h-8 w-8 border-r border-dotted border-base-400">
            <img src={svgMapping[svgName]} alt={svgName} />
        </div>
    </Tooltip>
);

LegendTile.propTypes = {
    svgName: PropTypes.string.isRequired,
    tooltip: PropTypes.string.isRequired
};

export default LegendTile;
