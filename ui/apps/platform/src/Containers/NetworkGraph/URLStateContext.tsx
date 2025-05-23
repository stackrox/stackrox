import React, { createContext, ReactNode, useContext, useMemo } from 'react';

import { timeWindows, TimeWindow } from 'constants/timeWindows';
import useURLPagination from 'hooks/useURLPagination';
import useURLParameter, { HistoryAction, QueryValue } from 'hooks/useURLParameter';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';

import { DEFAULT_NETWORK_GRAPH_PAGE_SIZE, TIME_WINDOW } from './NetworkGraph.constants';

export interface URLStateValue {
    pagination: ReturnType<typeof useURLPagination>;
    paginationAnomalous: ReturnType<typeof useURLPagination>;
    paginationBaseline: ReturnType<typeof useURLPagination>;

    searchFilter: ReturnType<typeof useURLSearch>;
    searchFilterSidePanel: ReturnType<typeof useURLSearch>;

    searchParameterSidePanelTab: {
        selectedTabSidePanel: QueryValue;
        setSelectedTabSidePanel: (tab: QueryValue, historyAction?: HistoryAction) => void;
    };

    searchParameterSidePanelToggle: {
        selectedToggleSidePanel: QueryValue;
        setSelectedToggleSidePanel: (toggle: QueryValue, historyAction?: HistoryAction) => void;
    };

    timeWindowParameter: {
        timeWindow: TimeWindow;
        setTimeWindow: (timeWindow: TimeWindow, historyAction?: HistoryAction) => void;
    };
}

const URLStateContext = createContext<URLStateValue | undefined>(undefined);

export function URLStateProvider({ children }: { children: ReactNode }) {
    const pagination = useURLPagination(DEFAULT_NETWORK_GRAPH_PAGE_SIZE);
    const paginationAnomalous = useURLPagination(DEFAULT_NETWORK_GRAPH_PAGE_SIZE, 'anomalous');
    const paginationBaseline = useURLPagination(DEFAULT_NETWORK_GRAPH_PAGE_SIZE, 'baseline');

    const searchFilter = useURLSearch();
    const searchFilterSidePanel = useURLSearch('sidePanel');

    const [selectedTabSidePanel, setSelectedTabSidePanel] = useURLParameter(
        'sidePanelTabState',
        undefined
    );

    const [selectedToggleSidePanel, setSelectedToggleSidePanel] = useURLParameter(
        'sidePanelToggleState',
        undefined
    );

    const [timeWindow, setTimeWindow] = useURLStringUnion(TIME_WINDOW, timeWindows, 'Past hour');

    const value = useMemo<URLStateValue>(
        () => ({
            pagination,
            paginationAnomalous,
            paginationBaseline,
            searchFilter,
            searchFilterSidePanel,
            searchParameterSidePanelTab: { selectedTabSidePanel, setSelectedTabSidePanel },
            searchParameterSidePanelToggle: { selectedToggleSidePanel, setSelectedToggleSidePanel },
            timeWindowParameter: { timeWindow, setTimeWindow },
        }),
        [
            pagination,
            paginationAnomalous,
            paginationBaseline,
            searchFilter,
            searchFilterSidePanel,
            selectedTabSidePanel,
            selectedToggleSidePanel,
            setSelectedTabSidePanel,
            setSelectedToggleSidePanel,
            setTimeWindow,
            timeWindow,
        ]
    );

    return <URLStateContext.Provider value={value}>{children}</URLStateContext.Provider>;
}

export function useURLState() {
    const context = useContext(URLStateContext);
    if (!context) {
        throw new Error('useURLState must be within <URLStateProvider>');
    }
    return context;
}

export const usePagination = () => useURLState().pagination;
export const usePaginationAnomalous = () => useURLState().paginationAnomalous;
export const usePaginationBaseline = () => useURLState().paginationBaseline;
export const useSearchFilter = () => useURLState().searchFilter;
export const useSearchFilterSidePanel = () => useURLState().searchFilterSidePanel;
export const useParameterSidePanelTab = () => useURLState().searchParameterSidePanelTab;
export const useParameterSidePanelToggle = () => useURLState().searchParameterSidePanelToggle;
export const useTimeWindowParameter = () => useURLState().timeWindowParameter;
