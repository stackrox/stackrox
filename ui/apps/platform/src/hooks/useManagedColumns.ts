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
    isShown: boolean;
    isShownByDefault: boolean;
    isUntoggleAble: boolean;
};

// The incoming type for the default column configuration
type InitialColumnConfig = Pick<ColumnConfig, 'isShownByDefault' | 'title'>;

// Basic validation of the shape of the object in local storage
function tablePreferencesValidator(value: unknown): value is TablePreferencesStorage {
    return yup.object({ columnManagement: yup.object().required() }).isValidSync(value);
}

// Using the existing stored table config as a base, merge in the provided config options
// and use the existing visibility state if it exists, otherwise use the default visibility
function getCurrentColumnConfig<ColumnName extends string>(
    tablePreferences: TablePreferencesStorage,
    tableId: string,
    columnConfig: Record<ColumnName, InitialColumnConfig>
): Record<ColumnName, ColumnConfig> {
    const tableConfig = {};
    Object.entries<InitialColumnConfig>(columnConfig).forEach(
        ([key, { title, isShownByDefault }]) => {
            const isShown =
                tablePreferences.columnManagement[tableId]?.[key]?.isShown ?? isShownByDefault;
            tableConfig[key] = {
                key,
                title,
                isShownByDefault,
                isShown,
                isUntoggleAble: false,
            };
        }
    );
    // Type assertion :( - we know that the keys are valid as the return object is created
    // from the same keys as the provided columnConfig. Fudging type safety here for this internal
    // function is worthwhile in order to gain additional safety in the useManagedColumns hook.
    return tableConfig as Record<ColumnName, ColumnConfig>;
}

export type ManagedColumns<ColumnKey extends string> = {
    /* The current configuration state of the columns */
    columns: Readonly<Record<ColumnKey, ColumnConfig>>;
    /* Toggle the visibility of a single column */
    toggleVisibility: (name: string) => void;
    /* Sets the visibility of multiple columns at once */
    setColumns: (columns: Record<string, boolean>) => void;
};

export function useManagedColumns<ColumnName extends string>(
    tableId: string,
    initialConfig: Readonly<Record<ColumnName, InitialColumnConfig>>
): ManagedColumns<ColumnName> {
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

    function toggleVisibility(name: string): void {
        setTablePreferencesStorage((prev) => {
            const isShown = !columns[name].isShown;
            return updateVisibility(prev, tableId, [[name, isShown]]);
        });
    }

    function setColumns(newColumns: Record<string, boolean>): void {
        setTablePreferencesStorage((prev) => {
            return updateVisibility(prev, tableId, Object.entries(newColumns));
        });
    }

    return {
        columns,
        setColumns,
        toggleVisibility,
    };
}
