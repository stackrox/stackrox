import React from 'react';
import PropTypes from 'prop-types';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipCardSection from 'Components/TooltipCardSection';
import TooltipFieldValue from 'Components/TooltipFieldValue';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

// @TODO: Remove "showPortsAndProtocols" when the feature flag "ROX_NETWORK_GRAPH_PORTS" is defaulted to true
const EdgeTooltipOverlay = ({
    source,
    target,
    isBidirectional,
    portsAndProtocols,
    showPortsAndProtocols,
}) => {
    const title = 'Network Flow';
    const tooltipContents =
        portsAndProtocols.length !== 0 ? (
            <PortsAndProtocolsFields portsAndProtocols={portsAndProtocols} />
        ) : (
            <div>Unavailable</div>
        );
    return (
        <DetailedTooltipOverlay
            title={title}
            body={
                <>
                    <div className="mb-2">
                        <TooltipCardSection
                            header={`1 ${
                                isBidirectional ? 'Bidirectional' : 'Unidirectional'
                            } Connection`}
                        >
                            {!isBidirectional && (
                                <>
                                    <TooltipFieldValue key={source} field="Source" value={source} />
                                    <TooltipFieldValue key={target} field="Target" value={target} />
                                </>
                            )}
                        </TooltipCardSection>
                    </div>
                    {showPortsAndProtocols && (
                        <TooltipCardSection header="Ports & Protocols">
                            {tooltipContents}
                        </TooltipCardSection>
                    )}
                </>
            }
        />
    );
};

EdgeTooltipOverlay.propTypes = {
    source: PropTypes.string.isRequired,
    target: PropTypes.string.isRequired,
    isBidirectional: PropTypes.bool,
    portsAndProtocols: PropTypes.arrayOf(PropTypes.shape),
    showPortsAndProtocols: PropTypes.bool,
};

EdgeTooltipOverlay.defaultProps = {
    isBidirectional: false,
    portsAndProtocols: [],
    showPortsAndProtocols: false,
};

export default EdgeTooltipOverlay;
