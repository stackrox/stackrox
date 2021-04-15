import React, { useState, ReactElement } from 'react';

import useSearchFilteredData from 'hooks/useSearchFilteredData';

import { PanelBody, PanelHead, PanelHeadEnd, PanelNew, PanelTitle } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NetworkPolicyYAMLOptions from './NetworkPolicyYAMLOptions';
import SimulatedNetworkBaselines from './SimulatedNetworkBaselines';
import SimulatedBaselinesSearch, {
    getSimulatedBaselineValueByCategory,
} from './SimulatedBaselinesSearch';

const simulatedNetworkBaselines = [
    {
        peer: {
            entity: {
                id: '12345',
                type: 'DEPLOYMENT',
                name: 'sensor',
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
                name: 'collector',
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
                name: 'monitoring',
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
    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);
    const filteredBaselines = useSearchFilteredData(
        simulatedNetworkBaselines,
        searchOptions,
        getSimulatedBaselineValueByCategory
    );

    return (
        <div className="bg-primary-100 rounded-b rounded-tr-lg shadow flex flex-1">
            <PanelNew testid="baseline-simulation">
                <PanelHead>
                    <PanelTitle text="Baseline Simulation" />
                    <PanelHeadEnd>
                        <NetworkPolicyYAMLOptions />
                        <TablePagination
                            page={page}
                            dataLength={filteredBaselines.length}
                            setPage={setPage}
                        />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelHead>
                    <PanelHeadEnd>
                        <div className="pr-3 w-full">
                            <SimulatedBaselinesSearch
                                networkBaselines={simulatedNetworkBaselines}
                                searchOptions={searchOptions}
                                setSearchOptions={setSearchOptions}
                            />
                        </div>
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <SimulatedNetworkBaselines simulatedNetworkBaselines={filteredBaselines} />
                </PanelBody>
            </PanelNew>
        </div>
    );
}

export default BaselineSimulation;
