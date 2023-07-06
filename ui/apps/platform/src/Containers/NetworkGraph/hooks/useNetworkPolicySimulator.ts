import { useEffect, useState } from 'react';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import * as networkService from 'services/NetworkService';
import { ensureExhaustive } from 'utils/type.utils';
import { NetworkPolicy, NetworkPolicyModification } from 'types/networkPolicy.proto';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { Simulation } from '../utils/getSimulation';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { useScopeHierarchy } from './useScopeHierarchy';

export type NetworkPolicySimulator =
    | {
          state: 'ACTIVE';
          networkPolicies: NetworkPolicy[];
          isLoading: boolean;
          error: string;
      }
    | {
          state: 'GENERATED' | 'UNDO' | 'UPLOAD';
          modification: NetworkPolicyModification | null;
          isLoading: boolean;
          error: string;
      };

export type SetNetworkPolicyModification = (action: SetNetworkPolicyModificationAction) => void;

type SetNetworkPolicyModificationAction =
    | {
          state: 'ACTIVE';
          options: {
              clusterId: string;
          };
      }
    | {
          state: 'GENERATED';
          options: {
              scopeHierarchy: NetworkScopeHierarchy;
              networkDataSince: string;
              excludePortsAndProtocols: boolean;
          };
      }
    | {
          state: 'UNDO';
          options: {
              clusterId: string;
          };
      }
    | {
          state: 'UPLOAD';
          options: {
              modification: NetworkPolicyModification | null;
              error: string;
          };
      };

type UseNetworkPolicySimulatorParams = {
    simulation: Simulation;
};

const defaultResultState = {
    state: 'ACTIVE',
    networkPolicies: [],
    error: '',
    isLoading: true,
} as NetworkPolicySimulator;

function useNetworkPolicySimulator({ simulation }: UseNetworkPolicySimulatorParams): {
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
} {
    const scopeHierarchy = useScopeHierarchy();
    const [simulator, setSimulator] = useState<NetworkPolicySimulator>(defaultResultState);

    useEffect(() => {
        setNetworkPolicyModification({
            state: 'ACTIVE',
            options: { clusterId: scopeHierarchy.cluster.id },
        });
    }, [scopeHierarchy.cluster.id, simulation.isOn]);

    function setNetworkPolicyModification(action: SetNetworkPolicyModificationAction): void {
        const { state, options } = action;
        if (state === 'ACTIVE') {
            setSimulator({
                state: 'ACTIVE',
                networkPolicies: [],
                error: '',
                isLoading: true,
            });
        } else {
            setSimulator({
                state,
                modification: null,
                error: '',
                isLoading: true,
            });
        }
        switch (state) {
            case 'ACTIVE':
                // @TODO: Add the network search query as a second argument
                networkService
                    .fetchNetworkPoliciesByClusterId(options.clusterId)
                    .then((data: NetworkPolicy[]) => {
                        setSimulator({
                            state,
                            networkPolicies: data,
                            error: '',
                            isLoading: false,
                        });
                    })
                    .catch((error) => {
                        const message = getAxiosErrorMessage(error);
                        const errorMessage =
                            message ||
                            'An unknown error occurred while getting the list of clusters';

                        setSimulator({
                            state,
                            networkPolicies: [],
                            error: errorMessage,
                            isLoading: false,
                        });
                    });
                break;
            case 'GENERATED':
                networkService
                    .generateNetworkModification(
                        options.scopeHierarchy.cluster.id,
                        getRequestQueryStringForSearchFilter({
                            Namespace: options.scopeHierarchy.namespaces,
                            Deployment: options.scopeHierarchy.deployments,
                            ...options.scopeHierarchy.remainingQuery,
                        }),
                        options.networkDataSince,
                        options.excludePortsAndProtocols
                    )
                    .then((data: NetworkPolicyModification) => {
                        setSimulator({
                            state,
                            modification: data,
                            error: '',
                            isLoading: false,
                        });
                    })
                    .catch((error) => {
                        const message = getAxiosErrorMessage(error);
                        const errorMessage =
                            message ||
                            'An unknown error occurred while getting the list of clusters';

                        setSimulator({
                            state,
                            modification: null,
                            error: errorMessage,
                            isLoading: false,
                        });
                    });
                break;
            case 'UNDO':
                networkService
                    .getUndoNetworkModification(options.clusterId)
                    .then((data: NetworkPolicyModification) => {
                        setSimulator({
                            state,
                            modification: data,
                            error: '',
                            isLoading: false,
                        });
                    })
                    .catch((error) => {
                        const message = getAxiosErrorMessage(error);
                        const errorMessage =
                            message ||
                            'An unknown error occurred while getting the list of clusters';

                        setSimulator({
                            state,
                            modification: null,
                            error: errorMessage,
                            isLoading: false,
                        });
                    });
                break;
            case 'UPLOAD':
                setSimulator({
                    state,
                    modification: options.modification,
                    error: options.error,
                    isLoading: false,
                });
                break;
            default:
                ensureExhaustive(state);
        }
    }

    return { simulator, setNetworkPolicyModification };
}

export default useNetworkPolicySimulator;
