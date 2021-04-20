import React, { useState, ReactElement } from 'react';

import { PanelBody, PanelHead, PanelHeadEnd, PanelNew, PanelTitle } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import { FilterState } from 'Containers/Network/networkTypes';
import useSearchFilteredData from 'hooks/useSearchFilteredData';
import NetworkPolicyYAMLOptions from './NetworkPolicyYAMLOptions';
import SimulatedNetworkBaselines from './SimulatedNetworkBaselines';
import BaselineSimulationSearch, {
    getSimulatedBaselineValueByCategory,
} from './BaselineSimulationSearch';
import useFetchBaselineComparison from './useFetchBaselineComparisons';

export type BaselineSimulationProps = {
    deploymentId: string;
    filterState: FilterState;
};

function BaselineSimulation({ deploymentId, filterState }: BaselineSimulationProps): ReactElement {
    const { simulatedBaselines, isLoading } = useFetchBaselineComparison({
        deploymentId,
        filterState,
    });
    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);
    const filteredBaselines = useSearchFilteredData(
        simulatedBaselines,
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
                            <BaselineSimulationSearch
                                networkBaselines={simulatedBaselines}
                                searchOptions={searchOptions}
                                setSearchOptions={setSearchOptions}
                            />
                        </div>
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <SimulatedNetworkBaselines
                        simulatedBaselines={filteredBaselines}
                        isLoading={isLoading}
                    />
                </PanelBody>
            </PanelNew>
        </div>
    );
}

export default BaselineSimulation;
