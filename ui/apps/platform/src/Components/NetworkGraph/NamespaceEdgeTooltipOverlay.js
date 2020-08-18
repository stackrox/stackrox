import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipCardSection from 'Components/TooltipCardSection';
import { filterModes } from 'constants/networkFilterModes';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

const DirectionalTooltipCardSection = ({ numBidirectional, numUnidirectional, type }) => {
    const numConnections = numBidirectional + numUnidirectional;
    if (!numConnections) {
        return null;
    }
    return (
        <div className="mb-2">
            <TooltipCardSection
                header={`${numConnections} ${type} ${pluralize('connection', numConnections)}`}
            >
                {!!numBidirectional && (
                    <div className="mb-1">
                        {numBidirectional} Bidirectional {pluralize('connection', numBidirectional)}
                    </div>
                )}
                {!!numUnidirectional && (
                    <div>
                        {numUnidirectional} Unidirectional{' '}
                        {pluralize('connection', numUnidirectional)}
                    </div>
                )}
            </TooltipCardSection>
        </div>
    );
};

const NamespaceEdgeTooltipOverlay = ({
    numBidirectionalLinks,
    numUnidirectionalLinks,
    numActiveBidirectionalLinks,
    numActiveUnidirectionalLinks,
    numAllowedBidirectionalLinks,
    numAllowedUnidirectionalLinks,
    portsAndProtocols,
    filterState,
}) => {
    const numConnections = numBidirectionalLinks + numUnidirectionalLinks;
    const title = `${numConnections} Network ${pluralize('Flow', numConnections)}`;
    const TooltipBody = (
        <>
            {filterState !== filterModes.allowed && (
                <DirectionalTooltipCardSection
                    numBidirectional={numActiveBidirectionalLinks}
                    numUnidirectional={numActiveUnidirectionalLinks}
                    type="active"
                />
            )}
            {filterState !== filterModes.active && (
                <DirectionalTooltipCardSection
                    numBidirectional={numAllowedBidirectionalLinks}
                    numUnidirectional={numAllowedUnidirectionalLinks}
                    type="allowed"
                />
            )}
            <div>
                <TooltipCardSection header="Ports & Protocols">
                    {portsAndProtocols.length !== 0 ? (
                        <PortsAndProtocolsFields portsAndProtocols={portsAndProtocols} />
                    ) : (
                        <div>Unavailable</div>
                    )}
                </TooltipCardSection>
            </div>
        </>
    );
    return <DetailedTooltipOverlay title={title} body={TooltipBody} />;
};

NamespaceEdgeTooltipOverlay.propTypes = {
    numBidirectionalLinks: PropTypes.number,
    numUnidirectionalLinks: PropTypes.number,
    numActiveBidirectionalLinks: PropTypes.number,
    numActiveUnidirectionalLinks: PropTypes.number,
    numAllowedBidirectionalLinks: PropTypes.number,
    numAllowedUnidirectionalLinks: PropTypes.number,
    portsAndProtocols: PropTypes.arrayOf(PropTypes.shape),
    filterState: PropTypes.oneOf(Object.values(filterModes)).isRequired,
};

NamespaceEdgeTooltipOverlay.defaultProps = {
    numBidirectionalLinks: 0,
    numUnidirectionalLinks: 0,
    numActiveBidirectionalLinks: 0,
    numActiveUnidirectionalLinks: 0,
    numAllowedBidirectionalLinks: 0,
    numAllowedUnidirectionalLinks: 0,
    portsAndProtocols: [],
};

export default NamespaceEdgeTooltipOverlay;
