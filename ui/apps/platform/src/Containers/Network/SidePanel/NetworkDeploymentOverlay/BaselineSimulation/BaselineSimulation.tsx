import React, { ReactElement } from 'react';

import { PanelBody, PanelHead, PanelHeadEnd, PanelNew, PanelTitle } from 'Components/Panel';
import NetworkPolicyYAMLOptions from './NetworkPolicyYAMLOptions';
import SimulatedNetworkBaselines from './SimulatedNetworkBaselines';

const simulatedNetworkBaselines = [
    {
        peer: {
            entity: {
                id: '12345',
                type: 'DEPLOYMENT',
                name: 'kube-dns',
                namespace: 'stackrox',
            },
            added: {
                port: '8080',
                protocol: 'L4_PROTOCOL_TCP',
                ingress: false,
            },
            state: 'active',
        },
        simulatedStatus: 'ADDED',
    },
    {
        peer: {
            entity: {
                id: '12345',
                type: 'DEPLOYMENT',
                name: 'kube-dns',
                namespace: 'stackrox',
            },
            removed: {
                port: '80',
                protocol: 'L4_PROTOCOL_TCP',
                ingress: true,
            },
            state: 'active',
        },
        simulatedStatus: 'REMOVED',
    },
    {
        peer: {
            entity: {
                id: '45678',
                type: 'DEPLOYMENT',
                name: 'sensor',
                namespace: 'stackrox',
            },
            modified: {
                added: {
                    port: '80',
                    protocol: 'L4_PROTOCOL_TCP',
                    ingress: true,
                },
                removed: {
                    port: '3000',
                    protocol: 'L4_PROTOCOL_TCP',
                    ingress: false,
                },
            },
            state: 'active',
        },
        simulatedStatus: 'MODIFIED',
    },
    {
        peer: {
            entity: {
                id: '24564',
                type: 'DEPLOYMENT',
                name: 'sensor',
                namespace: 'stackrox',
            },
            unmodified: {
                port: '80',
                protocol: 'L4_PROTOCOL_UDP',
                ingress: true,
            },
            state: 'active',
        },
        simulatedStatus: 'UNMODIFIED',
    },
];

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
                    <SimulatedNetworkBaselines
                        simulatedNetworkBaselines={simulatedNetworkBaselines}
                    />
                </PanelBody>
            </PanelNew>
        </div>
    );
}

export default BaselineSimulation;
