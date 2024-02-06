import { SearchFilter } from 'types/search';

type IdleState = {
    type: 'IDLE';
};

type LoadingState = {
    type: 'LOADING';
};

type PollingState<DataType> = {
    type: 'POLLING';
    data: DataType[];
};

type CompleteState<DataType> = {
    type: 'COMPLETE';
    data: DataType[];
};

type EmptyState = {
    type: 'EMPTY' | 'FILTERED_EMPTY';
};

type ErrorState = {
    type: 'ERROR';
    error: Error;
};

export type TableUIState<DataType> =
    | IdleState
    | LoadingState
    | PollingState<DataType>
    | CompleteState<DataType>
    | EmptyState
    | ErrorState;

type GetTableUIStateProps<DataType> = {
    isLoading: boolean;
    isPolling: boolean;
    data: undefined | DataType[];
    error: Error | undefined;
    searchFilter: SearchFilter;
};

export function getTableUIState<DataType>({
    isLoading,
    isPolling,
    data,
    error,
    searchFilter,
}: GetTableUIStateProps<DataType>): TableUIState<DataType> {
    const hasSearchFilters = Object.keys(searchFilter).length > 0;

    if (error) {
        return { type: 'ERROR', error };
    }

    if (isLoading && isPolling && data) {
        return { type: 'POLLING', data };
    }

    if (isLoading) {
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
