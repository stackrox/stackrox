import React, { useState, ReactElement } from 'react';
import * as Icon from 'react-feather';

import LegendTile from 'Components/LegendTile';

function LegendContent({ toggleLegend }: { toggleLegend: (value) => void }): ReactElement {
    return (
        <>
            <div className="flex justify-between border-b border-base-400 p-2 uppercase items-center">
                Legend
                <button type="button" className="flex" onClick={toggleLegend}>
                    <Icon.X className="h-3 w-3" />
                </button>
            </div>
            <div className="bg-primary-100">
                <div className="flex border-b border-base-400" data-testid="deployment-legend">
                    <LegendTile name="deployment" tooltip="Deployment" type="font" />
                    <LegendTile
                        name="deployment-external-connections"
                        tooltip="Deployment with active external connections"
                        type="svg"
                    />
                    <LegendTile
                        name="deployment-allowed-connections"
                        tooltip="Deployment with allowed external connections"
                        type="font"
                    />
                    <LegendTile
                        name="non-isolated-deployment-allowed"
                        tooltip="Non-isolated deployment (all connections allowed)"
                        type="font"
                    />
                </div>
                <div className="flex border-b border-base-400" data-testid="namespace-legend">
                    <LegendTile name="namespace" tooltip="Namespace" type="svg" />
                    <LegendTile
                        name="namespace-allowed-connection"
                        tooltip="Namespace with allowed external connections"
                        type="svg"
                    />
                    <LegendTile
                        name="namespace-connection"
                        tooltip="Namespace connection"
                        type="svg"
                    />
                </div>
                <div className="flex border-b border-base-400" data-testid="connection-legend">
                    <LegendTile name="active-connection" tooltip="Active connection" type="svg" />
                    <LegendTile name="allowed-connection" tooltip="Allowed connection" type="svg" />
                    <LegendTile
                        name="namespace-egress-ingress"
                        tooltip="Namespace external egress/ingress traffic"
                        type="font"
                    />
                </div>
            </div>
        </>
    );
}

function Legend(): ReactElement {
    const [isOpen, toggleOpen] = useState(true);

    function toggleLegend() {
        toggleOpen(!isOpen);
    }

    function handleKeyUp(e) {
        return e.key === 'Enter' ? toggleLegend() : null;
    }

    return (
        <div
            data-testid="legend"
            className="env-graph-legend absolute bottom-0 left-0 mb-2 ml-2 bg-base-100 text-base-500 text-sm font-700 border-base-400 border-2 rounded-sm z-10"
        >
            {!isOpen && (
                <div
                    role="button"
                    className="uppercase p-2 hover:bg-base-200 hover:text-primary-700 cursor-pointer"
                    onClick={toggleLegend}
                    onKeyUp={handleKeyUp}
                    tabIndex={0}
                >
                    Legend
                </div>
            )}
            {isOpen && <LegendContent toggleLegend={toggleLegend} />}
        </div>
    );
}

export default Legend;
