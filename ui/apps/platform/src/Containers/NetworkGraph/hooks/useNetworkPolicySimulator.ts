import { useEffect, useState } from 'react';

import { NetworkPolicyModification } from 'Containers/Network/networkTypes';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import * as networkService from 'services/NetworkService';
import { ensureExhaustive } from 'utils/type.utils';
import { NetworkPolicy } from 'types/networkPolicy.proto';

type NetworkPolicySimulator =
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

type SetNetworkPolicyModificationAction =
    | {
          state: 'ACTIVE';
          options: {
              clusterId: string;
              searchQuery: string;
          };
      }
    | {
          state: 'GENERATED';
          options: {
              clusterId: string;
              searchQuery: string;
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

function useNetworkPolicySimulator({ clusterId }): {
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: (action: SetNetworkPolicyModificationAction) => void;
    applyNetworkPolicyModification: () => void;
} {
    const defaultResultState = {
        state: 'ACTIVE',
        networkPolicies: [],
        error: '',
        isLoading: true,
    } as NetworkPolicySimulator;

    const [simulator, setSimulator] = useState<NetworkPolicySimulator>(defaultResultState);

    useEffect(() => {
        setNetworkPolicyModification({
            state: 'ACTIVE',
            options: {
                clusterId,
                searchQuery: '',
            },
        });
    }, []);

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
                        options.clusterId,
                        options.searchQuery,
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

    function applyNetworkPolicyModification() {
        if (simulator.state === 'ACTIVE') {
            return;
        }
        setSimulator({
            state: simulator.state,
            modification: simulator.modification,
            error: '',
            isLoading: true,
        });
        networkService
            .applyNetworkPolicyModification(clusterId, simulator.modification)
            .then(() => {
                setNetworkPolicyModification({
                    state: 'ACTIVE',
                    options: {
                        clusterId,
                        searchQuery: '',
                    },
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while applying the network policies';

                setSimulator({
                    state: simulator.state,
                    modification: simulator.modification,
                    error: errorMessage,
                    isLoading: false,
                });
            });
    }

    return { simulator, setNetworkPolicyModification, applyNetworkPolicyModification };
}

export default useNetworkPolicySimulator;
