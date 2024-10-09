import { renderHook, act } from '@testing-library/react';

import { generateVisibilityForColumns, useManagedColumns } from './useManagedColumns';

beforeEach(() => {
    window.localStorage.clear();
});

const defaultColumnConfig = {
    Name: {
        title: 'Name',
        isShownByDefault: true,
    },
    CVSS: {
        title: 'CVSS',
        isShownByDefault: true,
    },
    'NVD CVSS': {
        title: 'NVD CVSS',
        isShownByDefault: false,
    },
};

test('should return default column values', () => {
    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result.current.columns).toEqual({
        Name: {
            title: 'Name',
            key: 'Name',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        CVSS: {
            title: 'CVSS',
            key: 'CVSS',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        'NVD CVSS': {
            title: 'NVD CVSS',
            key: 'NVD CVSS',
            isShown: false,
            isShownByDefault: false,
            isUntoggleAble: false,
        },
    });
});

test('should toggle column visibility individually', () => {
    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    act(() => {
        result.current.toggleVisibility('Name');
    });

    expect(result.current.columns.Name.isShown).toEqual(false);
    expect(result.current.columns.CVSS.isShown).toEqual(true);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(false);

    act(() => {
        result.current.toggleVisibility('CVSS');
    });

    expect(result.current.columns.Name.isShown).toEqual(false);
    expect(result.current.columns.CVSS.isShown).toEqual(false);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(false);

    act(() => {
        result.current.toggleVisibility('Name');
    });

    expect(result.current.columns.Name.isShown).toEqual(true);
    expect(result.current.columns.CVSS.isShown).toEqual(false);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(false);

    // Check toggling a column that doesn't exist does not add an extra column to the state
    act(() => {
        result.current.toggleVisibility('Bogus');
    });

    expect(result.current.columns.Name.isShown).toEqual(true);
    expect(result.current.columns.CVSS.isShown).toEqual(false);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(false);
    // @ts-expect-error Should see a type error here when using an invalid key
    expect(result.current.columns.Bogus).toBeUndefined();
});

test('should set all columns to a specific visibility', () => {
    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    act(() => {
        result.current.setVisibility({
            Name: false,
            CVSS: false,
            'NVD CVSS': true,
        });
    });

    expect(result.current.columns.Name.isShown).toEqual(false);
    expect(result.current.columns.CVSS.isShown).toEqual(false);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(true);

    act(() => {
        result.current.setVisibility({
            Name: true,
            CVSS: true,
            'NVD CVSS': true,
        });
    });

    expect(result.current.columns.Name.isShown).toEqual(true);
    expect(result.current.columns.CVSS.isShown).toEqual(true);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(true);

    // Check setting a column that doesn't exist does not add an extra column to the state
    act(() => {
        result.current.setVisibility({
            Name: false,
            CVSS: true,
            Bogus: false,
        });
    });

    expect(result.current.columns.Name.isShown).toEqual(false);
    expect(result.current.columns.CVSS.isShown).toEqual(true);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(true);
    // @ts-expect-error Should see a type error here when using an invalid key
    expect(result.current.columns.Bogus).toBeUndefined();

    // Check that setting an empty object does not change the state
    act(() => {
        result.current.setVisibility({});
    });

    expect(result.current.columns.Name.isShown).toEqual(false);
    expect(result.current.columns.CVSS.isShown).toEqual(true);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(true);

    // Check setting a partial state does not change the other columns
    act(() => {
        result.current.setVisibility({
            Name: true,
        });
    });

    expect(result.current.columns.Name.isShown).toEqual(true);
    expect(result.current.columns.CVSS.isShown).toEqual(true);
    expect(result.current.columns['NVD CVSS'].isShown).toEqual(true);
});

test('should persist column visibility in local storage', () => {
    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result.current.columns.Name.isShown).toEqual(true);

    act(() => {
        result.current.toggleVisibility('Name');
    });

    expect(result.current.columns.Name.isShown).toEqual(false);

    // Initialize a new hook to simulate a page refresh and verify the state is persisted
    const { result: result2 } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result2.current.columns.Name.isShown).toEqual(false);
});

test('should maintain separate column visibility states for different tables', () => {
    const { result: result1 } = renderHook(() => useManagedColumns('test1', defaultColumnConfig));

    expect(result1.current.columns.Name.isShown).toEqual(true);

    act(() => {
        result1.current.toggleVisibility('Name');
    });

    expect(result1.current.columns.Name.isShown).toEqual(false);

    // Initialize a new hook with a different table id and verify the state is separate
    const { result: result2 } = renderHook(() => useManagedColumns('test2', defaultColumnConfig));

    expect(result2.current.columns.Name.isShown).toEqual(true);
});

test('should return default column values on invalid local storage data', () => {
    window.localStorage.setItem(
        'tablePreferences',
        JSON.stringify({
            columnManagement: {
                test: {
                    bogus: 1234567,
                },
            },
        })
    );

    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result.current.columns).toEqual({
        Name: {
            title: 'Name',
            key: 'Name',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        CVSS: {
            title: 'CVSS',
            key: 'CVSS',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        'NVD CVSS': {
            title: 'NVD CVSS',
            key: 'NVD CVSS',
            isShown: false,
            isShownByDefault: false,
            isUntoggleAble: false,
        },
    });
});

test('should return existing data merged with defaults if default config parameters are changed in the application', () => {
    // Initialize a previous saved state with only a single column
    window.localStorage.setItem(
        'tablePreferences',
        JSON.stringify({
            columnManagement: {
                test: {
                    Name: {
                        key: 'Name',
                        title: 'Name',
                        isShown: false,
                        isShownByDefault: true,
                        isUntoggleAble: false,
                    },
                },
            },
        })
    );

    // Load the hook with a modified default column config
    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result.current.columns).toEqual({
        Name: {
            title: 'Name',
            key: 'Name',
            isShown: false,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        CVSS: {
            title: 'CVSS',
            key: 'CVSS',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        'NVD CVSS': {
            title: 'NVD CVSS',
            key: 'NVD CVSS',
            isShown: false,
            isShownByDefault: false,
            isUntoggleAble: false,
        },
    });
});

test('should not lose state for other tables when a change is made to an existing table config', () => {
    window.localStorage.setItem(
        'tablePreferences',
        JSON.stringify({
            columnManagement: {
                // Initialize a previous saved state with only a single column
                test1: {
                    CVSS: {
                        title: 'CVSS',
                        key: 'CVSS',
                        isShown: true,
                        isShownByDefault: true,
                        isUntoggleAble: false,
                    },
                },
                test2: {
                    'First discovered': {
                        title: 'First discovered',
                        key: 'First discovered',
                        // Set to `false` to simulate a previous state where the column was hidden
                        isShown: false,
                        isShownByDefault: true,
                        isUntoggleAble: false,
                    },
                },
            },
        })
    );

    const secondTableConfig = {
        'First discovered': {
            title: 'First discovered',
            isShownByDefault: true,
        },
    };

    const { result: result1 } = renderHook(() => useManagedColumns('test1', defaultColumnConfig));

    expect(result1.current.columns.Name.isShown).toEqual(true);

    act(() => {
        result1.current.toggleVisibility('Name');
    });

    expect(result1.current.columns.Name.isShown).toEqual(false);

    // Initialize a new hook with a different table id and verify the state is separate
    const { result: result2 } = renderHook(() => useManagedColumns('test2', secondTableConfig));

    expect(result2.current.columns['First discovered'].isShown).toEqual(false);

    // Reload the state from the first table and ensure it is correct
    const { result: result3 } = renderHook(() => useManagedColumns('test1', defaultColumnConfig));

    expect(result3.current.columns.Name.isShown).toEqual(false);
});

test('should not corrupt state for other tables on invalid local storage data', () => {
    const secondTableConfig = {
        'First discovered': {
            title: 'First discovered',
            isShownByDefault: true,
        },
    };

    window.localStorage.setItem(
        'tablePreferences',
        JSON.stringify({
            columnManagement: {
                test1: {
                    bogus: 12345667,
                },
                test2: {
                    'First discovered': {
                        title: 'First discovered',
                        isShown: true,
                        isShownByDefault: true,
                        isUntoggleAble: false,
                    },
                },
            },
        })
    );

    const { result: result1 } = renderHook(() => useManagedColumns('test1', defaultColumnConfig));

    expect(result1.current.columns).toEqual({
        Name: {
            title: 'Name',
            key: 'Name',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        CVSS: {
            title: 'CVSS',
            key: 'CVSS',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        'NVD CVSS': {
            title: 'NVD CVSS',
            key: 'NVD CVSS',
            isShown: false,
            isShownByDefault: false,
            isUntoggleAble: false,
        },
    });

    // Trigger a write to local storage on the first table, overwriting the bogus data
    act(() => {
        result1.current.toggleVisibility('Name');
    });

    expect(result1.current.columns.Name.isShown).toEqual(false);

    // Check that the second table is not affected by the write
    const { result: result2 } = renderHook(() => useManagedColumns('test2', secondTableConfig));

    expect(result2.current.columns).toEqual({
        'First discovered': {
            title: 'First discovered',
            key: 'First discovered',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
    });

    // Toggle a write on the second table to ensure it is not affected by the first table
    act(() => {
        result2.current.toggleVisibility('First discovered');
    });

    // Reload the state from the first table and ensure both tables have the correct state
    const { result: result3 } = renderHook(() => useManagedColumns('test1', defaultColumnConfig));

    // Table one, reloaded
    expect(result3.current.columns.Name.isShown).toEqual(false);
    expect(result3.current.columns.CVSS.isShown).toEqual(true);
    expect(result3.current.columns['NVD CVSS'].isShown).toEqual(false);

    // Table two
    expect(result2.current.columns['First discovered'].isShown).toEqual(false);
});

test('should create local storage object if top level key is invalid value', () => {
    window.localStorage.setItem(
        'tablePreferences',
        JSON.stringify({
            columnManagement: 'bogus',
        })
    );

    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result.current.columns).toEqual({
        Name: {
            title: 'Name',
            key: 'Name',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        CVSS: {
            title: 'CVSS',
            key: 'CVSS',
            isShown: true,
            isShownByDefault: true,
            isUntoggleAble: false,
        },
        'NVD CVSS': {
            title: 'NVD CVSS',
            key: 'NVD CVSS',
            isShown: false,
            isShownByDefault: false,
            isUntoggleAble: false,
        },
    });

    act(() => {
        result.current.toggleVisibility('Name');
    });

    expect(result.current.columns.Name.isShown).toEqual(false);

    const { result: result2 } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    expect(result2.current.columns.Name.isShown).toEqual(false);
});

test('should correctly generate visibility classes', () => {
    const { result } = renderHook(() => useManagedColumns('test', defaultColumnConfig));

    act(() => {
        result.current.toggleVisibility('Name');
    });

    const getVisibilityClass = generateVisibilityForColumns(result.current.columns);

    expect(getVisibilityClass('Name')).toEqual('pf-v5-u-display-none');
    expect(getVisibilityClass('CVSS')).toEqual('');
    expect(getVisibilityClass('NVD CVSS')).toEqual('pf-v5-u-display-none');
    // Check that a column that doesn't exist returns an empty string and that attempting call the function
    // with an invalid key is not permitted when using TypeScript with a correctly inferred type
    // @ts-expect-error Should see a type error here when using an invalid key
    expect(getVisibilityClass('Bogus')).toEqual('');
});
