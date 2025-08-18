import * as yup from 'yup';
import merge from 'lodash/merge';

import useLocalStorage from './useLocalStorage';

// The type of the object stored in local storage
type TablePreferencesStorage = {
    columnManagement: Record<string, Record<string, ColumnConfig>>;
};

// The configuration for a single column in a table
export type ColumnConfig = {
    key: string;
    title: string;
    // Whether or not the column is shown in the table
    isShown: boolean;
    // Whether or not the column is shown by default when a configuration is first created
    isShownByDefault: boolean;
    // Whether or not the column is untoggleable in the Column Management modal.
    // If true, the column will not be shown in the Column Management modal.
    isUntoggleAble: boolean;
};

// The incoming type for the default column configuration
// Note that `title` is displayed in the Column Management modal and should match the column header value
type InitialColumnConfig = {
    title: string;
    isShownByDefault: boolean;
    isUntoggleAble?: boolean;
};

// Basic validation of the shape of the object in local storage
function tablePreferencesValidator(value: unknown): value is TablePreferencesStorage {
    return yup.object({ columnManagement: yup.object().required() }).isValidSync(value);
}

// Using the existing stored table config as a base, merge in the provided config options
// and use the existing visibility state if it exists, otherwise use the default visibility
function getCurrentColumnConfig<ColumnKey extends string>(
    tablePreferences: TablePreferencesStorage,
    tableId: string,
    columnConfig: Record<ColumnKey, InitialColumnConfig>
): Record<ColumnKey, ColumnConfig> {
    const tableConfig = {};
    Object.entries<InitialColumnConfig>(columnConfig).forEach(
        ([key, { title, isShownByDefault, isUntoggleAble }]) => {
            const isShown =
                tablePreferences.columnManagement[tableId]?.[key]?.isShown ?? isShownByDefault;
            tableConfig[key] = {
                key,
                title,
                isShownByDefault,
                isShown,
                isUntoggleAble: isUntoggleAble ?? false,
            };
        }
    );
    // Type assertion :( - we know that the keys are valid as the return object is created
    // from the same keys as the provided columnConfig. Fudging type safety here for this internal
    // function is worthwhile in order to gain additional safety in the useManagedColumns hook.
    return tableConfig as Record<ColumnKey, ColumnConfig>;
}

// Helper function to filter columns based on a predicate like feature flag dependency
// For example, (key) => key !== 'whatever' || isWhateverEnabled
export function filterManagedColumns<T extends Record<string, InitialColumnConfig>>(
    defaultColumns: T,
    predicate: (key: keyof T) => boolean
) {
    // Break potential chain into steps so easier to see types (and maybe remove casts in the future).
    const entries = Object.entries(defaultColumns) as [[keyof T, InitialColumnConfig]];
    const entriesFiltered = entries.filter(([key]) => predicate(key));
    const fromEntries = Object.fromEntries(entriesFiltered) as T;
    return fromEntries;
}

export function overrideManagedColumns<ColumnKey extends string>(
    managedColumns: Record<ColumnKey, ColumnConfig>,
    overrides: Partial<Record<ColumnKey, Partial<ColumnConfig>>>
): Record<ColumnKey, ColumnConfig> {
    return merge({}, managedColumns, overrides);
}

// Helper function to generate a visibility class based on the current column state
export function generateVisibilityForColumns<T extends Record<string, ColumnConfig>>(
    columnVisibilityState: T
) {
    return function getVisibilityClass(key: keyof T) {
        const state = columnVisibilityState[key];
        if (!state || state.isShown) {
            return '';
        }
        return 'pf-v5-u-display-none';
    };
}

export function getHiddenColumnCount(columnState: Record<string, ColumnConfig>): number {
    return Object.values(columnState).filter(({ isShown }) => !isShown).length;
}

export type ManagedColumns<ColumnKey extends string> = {
    /* The current configuration state of the columns */
    columns: Readonly<Record<ColumnKey, ColumnConfig>>;
    /* Toggle the visibility of a single column */
    toggleVisibility: (key: string) => void;
    /* Sets the visibility of multiple columns at once */
    setVisibility: (columns: Record<string, boolean>) => void;
};

export function useManagedColumns<ColumnKey extends string>(
    // `tableId` is a globally unique identifier for the table that indexes the column configuration
    // in local storage. It is typically formed by combining the Container folder name and the table file name.
    tableId: string,
    initialConfig: Readonly<Record<ColumnKey, InitialColumnConfig>>
): ManagedColumns<ColumnKey> {
    const [tablePreferencesStorage, setTablePreferencesStorage] = useLocalStorage(
        'tablePreferences',
        { columnManagement: {} },
        tablePreferencesValidator
    );

    const columns = getCurrentColumnConfig(tablePreferencesStorage, tableId, initialConfig);

    function updateVisibility(
        tablePreferences: TablePreferencesStorage,
        tableId: string,
        updates: [string, boolean][]
    ): TablePreferencesStorage {
        const validUpdates = updates.filter(([key]) => columns[key] !== undefined);
        validUpdates.forEach(([key, isShown]) => {
            columns[key].isShown = isShown;
        });
        return {
            columnManagement: merge({}, tablePreferences.columnManagement, { [tableId]: columns }),
        };
    }

    function toggleVisibility(key: string): void {
        setTablePreferencesStorage((prev) => {
            const isShown = !columns[key].isShown;
            return updateVisibility(prev, tableId, [[key, isShown]]);
        });
    }

    function setVisibility(newColumns: Record<string, boolean>): void {
        setTablePreferencesStorage((prev) => {
            return updateVisibility(prev, tableId, Object.entries(newColumns));
        });
    }

    return {
        columns,
        setVisibility,
        toggleVisibility,
    };
}
