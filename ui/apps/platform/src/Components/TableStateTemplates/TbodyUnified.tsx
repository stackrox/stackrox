import React from 'react';
import type { ReactElement, ReactNode } from 'react';

import type { TableUIState } from 'utils/getTableUIState';
import { ensureExhaustive } from 'utils/type.utils';

import { TbodyEmpty } from './TbodyEmpty';
import type { TbodyEmptyProps } from './TbodyEmpty';
import { TbodyError } from './TbodyError';
import type { TbodyErrorProps } from './TbodyError';
import { TbodyFilteredEmpty } from './TbodyFilteredEmpty';
import type { TbodyFilteredEmptyProps } from './TbodyFilteredEmpty';
import { TbodyLoading } from './TbodyLoading';
import type { TbodyLoadingProps } from './TbodyLoading';

export type TbodyUnifiedProps<T> = {
    /** The lifecycle state of a table data request */
    tableState: TableUIState<T>;
    colSpan: number;
    /**
     *  A function that renders the table body with the data. Can be a render prop or a component.
     */
    renderer: (props: { data: T[] }) => ReactNode;
    /** Props passed to the table loading state */
    loadingProps?: Omit<TbodyLoadingProps, 'colSpan'>;
    /** Props passed to the table error state */
    errorProps?: Omit<TbodyErrorProps, 'colSpan' | 'error'>;
    /** Props passed to the table empty state */
    emptyProps?: Omit<TbodyEmptyProps, 'colSpan'>;
    /** Props passed to the table filtered-empty state */
    filteredEmptyProps?: Omit<TbodyFilteredEmptyProps, 'colSpan'>;
};

/**
 * A component that encapsulates the rendering logic for table bodies based on the
 * lifecycle state of a table data request.
 */
function TbodyUnified<T>({
    tableState,
    colSpan,
    renderer,
    loadingProps,
    errorProps,
    emptyProps,
    filteredEmptyProps,
}: TbodyUnifiedProps<T>): ReactElement {
    const { type } = tableState;
    switch (type) {
        /*
            TODO (dv - 2024-04-22) We don't have a design for the IDLE state, and it will likely be component
            specific. We should add a prop for the IDLE state that allows the user to pass in a custom
            component or render prop once this is necessary.
        */
        case 'IDLE':
            return <></>;
        case 'LOADING':
            return <TbodyLoading colSpan={colSpan} {...loadingProps} />;
        case 'ERROR':
            return <TbodyError colSpan={colSpan} error={tableState.error} {...errorProps} />;
        case 'EMPTY':
            return <TbodyEmpty colSpan={colSpan} {...emptyProps} />;
        case 'FILTERED_EMPTY':
            return <TbodyFilteredEmpty colSpan={colSpan} {...filteredEmptyProps} />;
        /*
            We don't have a specific design for the POLLING state at this time, but discussions so far
            are that the UI changes when POLLING will be _outside_ of the table itself. In this case we
            would want to continue to show the table as it was before the POLLING state, as the indicator
            that the data is being refreshed would be elsewhere in the UI.
        */
        case 'COMPLETE':
            return <>{renderer({ data: tableState.data })}</>;
        default:
            return ensureExhaustive(type);
    }
}

export default TbodyUnified;
