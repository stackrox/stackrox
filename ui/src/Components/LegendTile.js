import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import deploymentActiveConnection from 'images/legend-icons/deployment-active-connection.svg';
import deploymentAllowedConnection from 'images/legend-icons/deployment-allowed-connection.svg';
import nonIsolatedDeploymentAllowed from 'images/legend-icons/non-isolated-deployment-allowed.svg';
import namespace from 'images/legend-icons/namespace.svg';
import namespaceAllowed from 'images/legend-icons/namespace-allowed.svg';
import namespaceConnection from 'images/legend-icons/namespace-connection.svg';
import * as constants from 'constants/networkGraph';

const svgMapping = {
    'active-connection': deploymentActiveConnection,
    'allowed-connection': deploymentAllowedConnection,
    namespace,
    'namespace-allowed-connection': namespaceAllowed,
    'namespace-connection': namespaceConnection,
    'non-isolated-deployment-allowed': nonIsolatedDeploymentAllowed
};

const fontIconMapping = {
    deployment: <i className="icon-node text-3xl" style={{ color: constants.COLORS.inactive }} />,
    'non-isolated-deployment-allowed': (
        <i className="icon-node text-3xl" style={{ color: constants.COLORS.nonIsolated }} />
    ),
    'deployment-allowed-connections': (
        <span className="text-center text-3xl relative">
            <i
                className="icon-potential absolute pin-t pin-r"
                style={{ color: constants.INTERNET_ACCESS_NODE_BORDER_COLOR }}
            />
            <i className="icon-node" style={{ color: constants.INTERNET_ACCESS_NODE_COLOR }} />
        </span>
    ),
    'namespace-egress-ingress': (
        <i
            className="icon-ingress-egress text-3xl"
            style={{ color: constants.INGRESS_EGRESS_ICON_COLOR }}
        />
    )
};

const LegendTile = ({ name, tooltip, type }) => (
    <Tooltip
        placement="top"
        overlay={<div>{tooltip}</div>}
        mouseLeaveDelay={0}
        className="flex items-center justify-center bg-base-100"
    >
        <div className="h-8 w-8 border-r border-dotted border-base-400">
            {type === 'font' && fontIconMapping[name]}
            {type === 'svg' && <img src={svgMapping[name]} alt={name} />}
        </div>
    </Tooltip>
);

LegendTile.propTypes = {
    name: PropTypes.string.isRequired,
    tooltip: PropTypes.string.isRequired,
    type: PropTypes.oneOf(['svg', 'font']).isRequired
};

export default LegendTile;
