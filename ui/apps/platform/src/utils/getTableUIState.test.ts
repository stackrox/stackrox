import { getTableUIState } from './getTableUIState';
import type { GetTableUIStateProps } from './getTableUIState';

type DataType = {
    text: string;
};

describe('getTableUIState', () => {
    it('should show the IDLE state', () => {
        // If nothing is set, it should be IDLE
        const args: GetTableUIStateProps<DataType> = {
            isLoading: false,
            isPolling: false,
            data: undefined,
            error: undefined,
            searchFilter: {},
        };

        const tableUIState = getTableUIState<DataType>(args);

        expect(tableUIState).toEqual({
            type: 'IDLE',
        });
    });

    it('should show the ERROR state', () => {
        const error = new Error('There is an error');

        // Regardless of other values, if error is defined, then it should show an error
        expect(
            getTableUIState<DataType>({
                isLoading: true,
                isPolling: true,
                data: [{ text: 'Test 1' }],
                error,
                searchFilter: {
                    SEARCH_TERM: 'SEARCH_VALUE',
                },
            })
        ).toEqual({
            type: 'ERROR',
            error,
        });
    });

    it('should show the LOADING state', () => {
        const args: GetTableUIStateProps<DataType> = {
            isLoading: true,
            isPolling: false,
            data: undefined,
            error: undefined,
            searchFilter: {},
        };

        const tableUIState = getTableUIState<DataType>(args);

        expect(tableUIState).toEqual({
            type: 'LOADING',
        });
    });

    it('should show the EMPTY state', () => {
        const args: GetTableUIStateProps<DataType> = {
            isLoading: false,
            isPolling: false,
            data: [],
            error: undefined,
            searchFilter: {},
        };

        const tableUIState = getTableUIState<DataType>(args);

        expect(tableUIState).toEqual({
            type: 'EMPTY',
        });
    });

    it('should show the FILTERED_EMPTY state', () => {
        const args: GetTableUIStateProps<DataType> = {
            isLoading: false,
            isPolling: false,
            data: [],
            error: undefined,
            searchFilter: {
                SEARCH_TERM: 'SEARCH_VALUE',
            },
        };

        const tableUIState = getTableUIState<DataType>(args);

        expect(tableUIState).toEqual({
            type: 'FILTERED_EMPTY',
        });
    });

    it('should show the COMPLETE state', () => {
        const data = [{ text: 'Test 1' }];

        // When data is present without search filters
        expect(
            getTableUIState<DataType>({
                isLoading: false,
                isPolling: false,
                data,
                error: undefined,
                searchFilter: {},
            })
        ).toEqual({
            type: 'COMPLETE',
            data,
        });

        // When data is present with search filters
        expect(
            getTableUIState<DataType>({
                isLoading: false,
                isPolling: false,
                data,
                error: undefined,
                searchFilter: {
                    SEARCH_TERM: 'SEARCH_VALUE',
                },
            })
        ).toEqual({
            type: 'COMPLETE',
            data,
        });
    });
});
