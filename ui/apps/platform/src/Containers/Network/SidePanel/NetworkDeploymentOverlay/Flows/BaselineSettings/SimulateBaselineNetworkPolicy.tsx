import React, { ReactElement, useState } from 'react';

import useNetworkBaselineSimulation from 'Containers/Network/useNetworkBaselineSimulation';

import { PrimaryButton, CheckboxWithLabel } from '@stackrox/ui-components';

function SimulateBaselineNetworkPolicy(): ReactElement {
    const [excludePortsAndProtocols, setExcludePortsAndProtocols] = useState(false);
    const { startBaselineSimulation } = useNetworkBaselineSimulation();

    function onClick() {
        startBaselineSimulation({ excludePortsAndProtocols });
    }

    function onCheckboxChange(event: React.ChangeEvent<HTMLInputElement>) {
        setExcludePortsAndProtocols(event.target.checked);
    }

    return (
        <div className="flex flex-col justify-center">
            <div className="pb-3">
                <CheckboxWithLabel
                    id="exclude-ports-and-protocols"
                    ariaLabel="Exclude Ports and Protocols"
                    checked={excludePortsAndProtocols}
                    onChange={onCheckboxChange}
                >
                    Exclude Ports and Protocols
                </CheckboxWithLabel>
            </div>
            <PrimaryButton onClick={onClick}>Simulate Baseline as Network Policy</PrimaryButton>
        </div>
    );
}

export default SimulateBaselineNetworkPolicy;
