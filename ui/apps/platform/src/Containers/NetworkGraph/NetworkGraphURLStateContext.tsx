import React, { createContext, useContext, useMemo } from 'react';
import type { ReactNode } from 'react';

import { timeWindows } from 'constants/timeWindows';
import type { TimeWindow } from 'constants/timeWindows';
import useURLPagination from 'hooks/useURLPagination';
import useURLParameter from 'hooks/useURLParameter';
import type { HistoryAction, QueryValue } from 'hooks/useURLParameter';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';

import { edgeStates } from './components/EdgeStateSelect';
import type { EdgeState } from './components/EdgeStateSelect';
import { DEFAULT_NETWORK_GRAPH_PAGE_SIZE, EDGE_STATE, TIME_WINDOW } from './NetworkGraph.constants';

export const SIDE_PANEL_SEARCH_PREFIX = 's2';

export type NetworkGraphURLStateValue = {
    pagination: ReturnType<typeof useURLPagination>;
    paginationSecondary: ReturnType<typeof useURLPagination>;

    searchFilter: ReturnType<typeof useURLSearch>;
    searchFilterSidePanel: ReturnType<typeof useURLSearch>;

    sidePanelTab: {
        selectedTabSidePanel: QueryValue;
        setSelectedTabSidePanel: (tab: QueryValue, historyAction?: HistoryAction) => void;
    };

    sidePanelToggle: {
        selectedToggleSidePanel: QueryValue;
        setSelectedToggleSidePanel: (toggle: QueryValue, historyAction?: HistoryAction) => void;
    };

    edgeState: {
        edgeState: EdgeState;
        setEdgeState: (edgeState: EdgeState, historyAction?: HistoryAction) => void;
    };

    timeWindow: {
        timeWindow: TimeWindow;
        setTimeWindow: (timeWindow: TimeWindow, historyAction?: HistoryAction) => void;
    };
};

const NetworkGraphURLStateContext = createContext<NetworkGraphURLStateValue | undefined>(undefined);

export function NetworkGraphURLStateProvider({ children }: { children: ReactNode }) {
    const pagination = useURLPagination(DEFAULT_NETWORK_GRAPH_PAGE_SIZE);
    const paginationSecondary = useURLPagination(DEFAULT_NETWORK_GRAPH_PAGE_SIZE, 'secondary');

    const searchFilter = useURLSearch();
    const searchFilterSidePanel = useURLSearch(SIDE_PANEL_SEARCH_PREFIX);

    const [selectedTabSidePanel, setSelectedTabSidePanel] = useURLParameter(
        'sidePanelTabState',
        undefined
    );

    const [selectedToggleSidePanel, setSelectedToggleSidePanel] = useURLParameter(
        'sidePanelToggleState',
        undefined
    );

    const [edgeState, setEdgeState] = useURLStringUnion(EDGE_STATE, edgeStates, 'active');
    const [timeWindow, setTimeWindow] = useURLStringUnion(TIME_WINDOW, timeWindows, 'Past hour');

    const value = useMemo<NetworkGraphURLStateValue>(
        () => ({
            pagination,
            paginationSecondary,
            searchFilter,
            searchFilterSidePanel,
            sidePanelTab: { selectedTabSidePanel, setSelectedTabSidePanel },
            sidePanelToggle: { selectedToggleSidePanel, setSelectedToggleSidePanel },
            edgeState: { edgeState, setEdgeState },
            timeWindow: { timeWindow, setTimeWindow },
        }),
        [
            edgeState,
            pagination,
            paginationSecondary,
            searchFilter,
            searchFilterSidePanel,
            selectedTabSidePanel,
            selectedToggleSidePanel,
            setEdgeState,
            setSelectedTabSidePanel,
            setSelectedToggleSidePanel,
            setTimeWindow,
            timeWindow,
        ]
    );

    return (
        <NetworkGraphURLStateContext.Provider value={value}>
            {children}
        </NetworkGraphURLStateContext.Provider>
    );
}

function useNetworkGraphURLState() {
    const context = useContext(NetworkGraphURLStateContext);
    if (!context) {
        throw new Error(
            'useNetworkGraphURLState must be used within <NetworkGraphURLStateProvider>'
        );
    }
    return context;
}

export const usePagination = () => useNetworkGraphURLState().pagination;
export const usePaginationSecondary = () => useNetworkGraphURLState().paginationSecondary;
export const useSearchFilter = () => useNetworkGraphURLState().searchFilter;
export const useSearchFilterSidePanel = () => useNetworkGraphURLState().searchFilterSidePanel;
export const useSidePanelTab = () => useNetworkGraphURLState().sidePanelTab;
export const useSidePanelToggle = () => useNetworkGraphURLState().sidePanelToggle;
export const useEdgeState = () => useNetworkGraphURLState().edgeState;
export const useTimeWindow = () => useNetworkGraphURLState().timeWindow;
