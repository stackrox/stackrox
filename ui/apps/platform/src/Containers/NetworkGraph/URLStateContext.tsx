import React, { createContext, ReactNode, useContext, useMemo } from 'react';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLParameter, { HistoryAction, QueryValue } from 'hooks/useURLParameter';
import { DEFAULT_NETWORK_GRAPH_PAGE_SIZE } from './NetworkGraph.constants';

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

    const value = useMemo<URLStateValue>(
        () => ({
            pagination,
            paginationAnomalous,
            paginationBaseline,
            searchFilter,
            searchFilterSidePanel,
            searchParameterSidePanelTab: { selectedTabSidePanel, setSelectedTabSidePanel },
            searchParameterSidePanelToggle: { selectedToggleSidePanel, setSelectedToggleSidePanel },
        }),
        [
            pagination,
            paginationAnomalous,
            paginationBaseline,
            searchFilter,
            searchFilterSidePanel,
            selectedTabSidePanel,
            setSelectedTabSidePanel,
            selectedToggleSidePanel,
            setSelectedToggleSidePanel,
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
