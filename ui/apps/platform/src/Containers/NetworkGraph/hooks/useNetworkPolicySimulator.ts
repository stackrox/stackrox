import { useEffect, useState } from 'react';

import { NetworkPolicyModification } from 'Containers/Network/networkTypes';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import {
    fetchNetworkPoliciesByClusterId,
    generateNetworkModification,
    getUndoNetworkModification,
} from 'services/NetworkService';
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
              modification: NetworkPolicyModification;
          };
      };

function useNetworkPolicySimulator({ clusterId }): {
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: (action: SetNetworkPolicyModificationAction) => void;
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
                fetchNetworkPoliciesByClusterId(options.clusterId)
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
                generateNetworkModification(
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
                getUndoNetworkModification(options.clusterId)
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
                    error: '',
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
