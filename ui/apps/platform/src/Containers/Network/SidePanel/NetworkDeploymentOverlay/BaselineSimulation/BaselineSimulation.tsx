import React, { useState, useEffect, ReactElement } from 'react';

import { PanelBody, PanelHead, PanelHeadEnd, PanelNew, PanelTitle } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import useSearchFilteredData from 'hooks/useSearchFilteredData';
import useNetworkBaselineSimulation from 'Containers/Network/useNetworkBaselineSimulation';
import useFetchBaselineComparison from 'Containers/Network/useFetchBaselineComparisons';
import { getUndoModificationForDeployment } from 'services/NetworkService';
import { getDateTime } from 'utils/dateUtils';
import NetworkPolicyYAMLOptions from './NetworkPolicyYAMLOptions';
import SimulatedNetworkBaselines from './SimulatedNetworkBaselines';
import ApplyBaselineNetworkPolicy from './ApplyBaselineNetworkPolicy';
import BaselineSimulationSearch, {
    getSimulatedBaselineValueByCategory,
} from './BaselineSimulationSearch';
import useFetchBaselineGeneratedNetworkPolicy from './useFetchBaselineGeneratedNetworkPolicy';

type UndoModfication = {
    undoRecord: {
        user: string;
        applyTimestamp: string; // ISO 8601 timestamp
        originalModification: {
            applyYaml: string;
        };
        undoModification: {
            applyYaml: string;
        };
    };
};

export type BaselineSimulationProps = {
    deploymentId: string;
};

function BaselineSimulation({ deploymentId }: BaselineSimulationProps): ReactElement {
    const {
        baselineSimulationOptions: { excludePortsAndProtocols },
        stopBaselineSimulation,
        isUndoOn,
    } = useNetworkBaselineSimulation();
    const { simulatedBaselines, isLoading } = useFetchBaselineComparison();
    const { data: networkPolicy, isGeneratingNetworkPolicy } =
        useFetchBaselineGeneratedNetworkPolicy({
            deploymentId,
            includePorts: !excludePortsAndProtocols,
        });
    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);
    const filteredBaselines = useSearchFilteredData(
        simulatedBaselines,
        searchOptions,
        getSimulatedBaselineValueByCategory
    );
    const [undoModification, setUndoModification] = useState<UndoModfication | null>(null);

    useEffect(() => {
        getUndoModificationForDeployment(deploymentId)
            .then((response) => {
                setUndoModification(response);
            })
            .catch((err) => {
                // if there is application-applied undo record, that is returned as an error
                // so we filter that type of error out, before throwing
                if (!err?.response?.data?.message?.includes('no undo record stored')) {
                    throw Error('Error retrieving possible undo network policy');
                }
            });
    }, [deploymentId]);

    const undoAvailable = !!undoModification && !isUndoOn;

    return (
        <div className="bg-primary-100 rounded-b rounded-tr-lg shadow flex flex-1 flex-col">
            <PanelNew testid="baseline-simulation">
                <PanelHead>
                    <PanelTitle text="Baseline Simulation" />
                    <PanelHeadEnd>
                        <NetworkPolicyYAMLOptions
                            networkPolicy={networkPolicy}
                            undoAvailable={undoAvailable}
                            isUndoOn={isUndoOn}
                        />
                        <TablePagination
                            page={page}
                            dataLength={filteredBaselines.length}
                            setPage={setPage}
                        />
                    </PanelHeadEnd>
                </PanelHead>
                {!!undoModification && (
                    <p className="px-3 py-2">
                        The last network policy was applied by{' '}
                        <strong>{undoModification?.undoRecord?.user}</strong> on{' '}
                        <strong>
                            {getDateTime(undoModification?.undoRecord?.applyTimestamp || '')}
                        </strong>
                        .
                    </p>
                )}
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
                        isLoading={isLoading || isGeneratingNetworkPolicy}
                    />
                </PanelBody>
            </PanelNew>
            {networkPolicy && (
                <div className="flex justify-center items-center py-4 border-t border-primary-300 bg-primary-100">
                    <ApplyBaselineNetworkPolicy
                        deploymentId={deploymentId}
                        networkPolicy={networkPolicy}
                        stopBaselineSimulation={stopBaselineSimulation}
                    />
                </div>
            )}
        </div>
    );
}

export default BaselineSimulation;
