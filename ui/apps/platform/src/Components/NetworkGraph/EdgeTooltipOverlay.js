import React from 'react';
import PropTypes from 'prop-types';

import { DetailedTooltipOverlay } from '@stackrox/ui-components';
import TooltipCardSection from 'Components/TooltipCardSection';
import TooltipFieldValue from 'Components/TooltipFieldValue';
import PortsAndProtocolsFields from './PortsAndProtocolsFields';

const EdgeTooltipOverlay = ({ source, target, isBidirectional, portsAndProtocols }) => {
    const title = 'Network Flow';
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
                    {portsAndProtocols.length !== 0 && (
                        <TooltipCardSection header="Ports & Protocols">
                            <PortsAndProtocolsFields portsAndProtocols={portsAndProtocols} />
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
};

EdgeTooltipOverlay.defaultProps = {
    isBidirectional: false,
    portsAndProtocols: [],
};

export default EdgeTooltipOverlay;
