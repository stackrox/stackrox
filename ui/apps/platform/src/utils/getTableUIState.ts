import type { SearchFilter } from 'types/search';

export type IdleState = {
    type: 'IDLE';
};

export type LoadingState = {
    type: 'LOADING';
};

export type CompleteState<DataType> = {
    type: 'COMPLETE';
    data: DataType[];
};

export type EmptyState = {
    type: 'EMPTY' | 'FILTERED_EMPTY';
};

export type ErrorState = {
    type: 'ERROR';
    error: Error;
};

export type TableUIState<DataType> =
    | IdleState
    | LoadingState
    | CompleteState<DataType>
    | EmptyState
    | ErrorState;

export type GetTableUIStateProps<DataType> = {
    isLoading: boolean;
    isPolling?: boolean;
    data: undefined | DataType[];
    error: Error | undefined;
    searchFilter: SearchFilter;
};

export function getTableUIState<DataType>({
    isLoading,
    isPolling = false,
    data,
    error,
    searchFilter,
}: GetTableUIStateProps<DataType>): TableUIState<DataType> {
    const hasSearchFilters = Object.keys(searchFilter).length > 0;

    if (error) {
        return { type: 'ERROR', error };
    }

    if (isLoading && !isPolling) {
        return { type: 'LOADING' };
    }

    if (data && data.length > 0) {
        return { type: 'COMPLETE', data };
    }

    if (hasSearchFilters) {
        return { type: 'FILTERED_EMPTY' };
    }

    if (data && data.length === 0) {
        return { type: 'EMPTY' };
    }

    return { type: 'IDLE' };
}
