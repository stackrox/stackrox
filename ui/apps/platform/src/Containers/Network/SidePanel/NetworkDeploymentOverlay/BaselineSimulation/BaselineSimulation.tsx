import React, { ReactElement } from 'react';

import { PanelBody, PanelHead, PanelHeadEnd, PanelNew, PanelTitle } from 'Components/Panel';
import NetworkPolicyYAMLOptions from './NetworkPolicyYAMLOptions';
import SimulatedNetworkBaselines from './SimulatedNetworkBaselines';

function BaselineSimulation(): ReactElement {
    return (
        <div className="bg-primary-100 rounded-b rounded-tr-lg shadow flex flex-1">
            <PanelNew testid="baseline-simulation">
                <PanelHead>
                    <PanelTitle text="Baseline Simulation" />
                    <PanelHeadEnd>
                        <NetworkPolicyYAMLOptions />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <SimulatedNetworkBaselines />
                </PanelBody>
            </PanelNew>
        </div>
    );
}

export default BaselineSimulation;
