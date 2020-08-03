import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipCardSection from 'Components/TooltipCardSection';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

const NamespaceEdgeTooltipOverlay = ({
    numBidirectionalLinks,
    numUnidirectionalLinks,
    portsAndProtocols,
}) => {
    const numConnections = numBidirectionalLinks + numUnidirectionalLinks;
    const title = `${numConnections} ${pluralize('Connection', numConnections)}`;
    return (
        <DetailedTooltipOverlay
            // TODO (ROX-5215): We will change this to say "Network Flows" and put this info in another
            // TooltipCardSection where we'll also show the number of bidirectional and
            // unidirectional connections
            title={title}
            body={
                <TooltipCardSection header="Ports & Protocols">
                    <PortsAndProtocolsFields portsAndProtocols={portsAndProtocols} />
                </TooltipCardSection>
            }
        />
    );
};

NamespaceEdgeTooltipOverlay.propTypes = {
    numBidirectionalLinks: PropTypes.number,
    numUnidirectionalLinks: PropTypes.number,
    portsAndProtocols: PropTypes.arrayOf(PropTypes.shape),
};

NamespaceEdgeTooltipOverlay.defaultProps = {
    numBidirectionalLinks: 0,
    numUnidirectionalLinks: 0,
    portsAndProtocols: [],
};

export default NamespaceEdgeTooltipOverlay;
