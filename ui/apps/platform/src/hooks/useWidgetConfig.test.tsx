/* eslint-disable @typescript-eslint/no-floating-promises */
import React from 'react';
import { render, renderHook, act } from '@testing-library/react';
import useWidgetConfig, { defaultStorageKey } from 'hooks/useWidgetConfig';

beforeEach(() => {
    localStorage.clear();
});

describe('useWidgetConfig hook', () => {
    it('should persist a widget configuration in localStorage', () => {
        const widgetId = 'single-component';
        const routeId = '/';
        const defaultConfig = {
            sort: 'asc',
            filter: '',
        };
        const { result: firstResult } = renderHook(() =>
            useWidgetConfig(widgetId, routeId, defaultConfig)
        );

        // Expect the initial state to be clean and return the provided defaults
        expect(firstResult.current[0]).toEqual(defaultConfig);

        // By default changes should merge the new and old configs
        act(() => {
            const [, updateConfig] = firstResult.current;
            updateConfig({ filter: 'Namespace:stackrox' });
        });
        expect(firstResult.current[0]).toEqual({ sort: 'asc', filter: 'Namespace:stackrox' });

        act(() => {
            const [, updateConfig] = firstResult.current;
            updateConfig({ sort: 'desc' });
        });
        expect(firstResult.current[0]).toEqual({ sort: 'desc', filter: 'Namespace:stackrox' });

        // Loading the same widget/route config in a new component, i.e. a page reload
        const { result: secondResult } = renderHook(() =>
            useWidgetConfig(widgetId, routeId, defaultConfig)
        );

        // Results should be the persisted config instead of the default
        expect(secondResult.current[0]).toEqual({ sort: 'desc', filter: 'Namespace:stackrox' });
    });

    it('should allow a custom update method to be used when updating the configuration', () => {
        const widgetId = 'single-component';
        const routeId = '/';
        const defaultConfig = {
            sort: 'asc',
            filter: '',
        };
        type Config = typeof defaultConfig;

        // Reducer that does a full config replace instead of merge
        const reducer = (_oldConfig, payload: Partial<Config>) => payload;

        const { result: firstResult } = renderHook(() =>
            useWidgetConfig(widgetId, routeId, defaultConfig, reducer)
        );

        expect(firstResult.current[0]).toEqual(defaultConfig);

        act(() => {
            const [, updateConfig] = firstResult.current;
            updateConfig({ filter: 'Namespace:stackrox' });
        });

        // Even though the `sort` property was overwritten, the hook ensures type safety by
        // replacing it with the default value
        expect(firstResult.current[0]).toEqual({ sort: 'asc', filter: 'Namespace:stackrox' });

        act(() => {
            const [, updateConfig] = firstResult.current;
            updateConfig({ sort: 'desc' });
        });

        // The `filter` property is overwritten and replaced with the default value
        expect(firstResult.current[0]).toEqual({ sort: 'desc', filter: '' });

        act(() => {
            const [, updateConfig] = firstResult.current;
            updateConfig({});
        });
        expect(firstResult.current[0]).toEqual(defaultConfig);
    });

    it('should store separate configurations for the same widget at different routes', () => {
        const widgetId = 'single-component';
        const defaultConfig = {
            sort: 'asc',
            filter: '',
        };
        const { result: firstResult } = renderHook(() => [
            useWidgetConfig(widgetId, '/pathA', defaultConfig),
            useWidgetConfig(widgetId, '/pathB', defaultConfig),
        ]);

        expect(firstResult.current[0][0]).toEqual(defaultConfig);
        expect(firstResult.current[1][0]).toEqual(defaultConfig);

        // Update the state of each widget config
        act(() => {
            firstResult.current[0][1]({ sort: 'desc' });
            firstResult.current[1][1]({ filter: 'Namespace:stackrox' });
        });
        expect(firstResult.current[0][0]).toEqual({ sort: 'desc', filter: '' });
        expect(firstResult.current[1][0]).toEqual({ sort: 'asc', filter: 'Namespace:stackrox' });

        // Reload the widget configurations
        const { result: secondResult } = renderHook(() => [
            useWidgetConfig(widgetId, '/pathA', defaultConfig),
            useWidgetConfig(widgetId, '/pathB', defaultConfig),
        ]);

        // Results should be persisted separately
        expect(secondResult.current[0][0]).toEqual({ sort: 'desc', filter: '' });
        expect(secondResult.current[1][0]).toEqual({ sort: 'asc', filter: 'Namespace:stackrox' });
    });

    it('should handle multiple unrelated widget configs simultaneously', () => {
        const idA = 'widgetA';
        const idB = 'widgetB';
        const renderSpies = { [idA]: jest.fn(), [idB]: jest.fn() };
        const hookReturns = { [idA]: {}, [idB]: {} };

        const defaultConfig = { sort: 'asc', filter: '' };

        function Widget({ id }) {
            renderSpies[id]();
            hookReturns[id] = useWidgetConfig(id, '/', defaultConfig);
            return <></>;
        }
        render(
            <>
                <Widget id={idA} />
                <Widget id={idB} />
            </>
        );

        expect(renderSpies[idA]).toHaveBeenCalledTimes(1);
        expect(renderSpies[idB]).toHaveBeenCalledTimes(1);

        act(() => {
            hookReturns[idA][1]({ sort: 'desc' });
        });

        // Updating the config for one component should not cause the other component to rerender
        // and should only update the config state of the changed component.
        expect(renderSpies[idA]).toHaveBeenCalledTimes(2);
        expect(renderSpies[idB]).toHaveBeenCalledTimes(1);
        expect(hookReturns[idA][0]).toEqual({ sort: 'desc', filter: '' });
        expect(hookReturns[idB][0]).toEqual({ sort: 'asc', filter: '' });

        act(() => {
            hookReturns[idB][1]({ filter: 'Namespace:stackrox' });
        });

        expect(renderSpies[idA]).toHaveBeenCalledTimes(2);
        expect(renderSpies[idB]).toHaveBeenCalledTimes(2);
        expect(hookReturns[idA][0]).toEqual({ sort: 'desc', filter: '' });
        expect(hookReturns[idB][0]).toEqual({ sort: 'asc', filter: 'Namespace:stackrox' });

        // Simulate page reload for the same configurations
        const { result: resultA } = renderHook(() => useWidgetConfig(idA, '/', defaultConfig));
        const { result: resultB } = renderHook(() => useWidgetConfig(idB, '/', defaultConfig));

        // This check is important since all dashboard configurations are stored in a
        // single object in local storage. This ensures that a write to one config does not result
        // in data loss when a second config is saved.
        expect(resultA.current[0]).toEqual({ sort: 'desc', filter: '' });
        expect(resultB.current[0]).toEqual({ sort: 'asc', filter: 'Namespace:stackrox' });
    });

    it('should provide resiliency against invalid external writes to localStorage', () => {
        const widgetId = 'single-component';
        const routeId = '/';
        const defaultConfig = { sort: 'asc', filter: '' };
        let result;

        // Write a value to the root widget config
        function storeRootConfig(config) {
            localStorage.setItem(defaultStorageKey, config);
        }

        // "Control" assertion to ensure pre-existing, type-compatible properties are saved and loaded correctly
        storeRootConfig(
            JSON.stringify({
                [widgetId]: {
                    [routeId]: {
                        filter: 'Namespace:stackrox',
                        sort: 'desc',
                    },
                },
            })
        );
        ({ result } = renderHook(() => useWidgetConfig(widgetId, routeId, defaultConfig)));
        expect(result.current[0]).toEqual({ sort: 'desc', filter: 'Namespace:stackrox' });

        // Force overwrite top level config as a string instead of object
        storeRootConfig('bogus-string');
        ({ result } = renderHook(() => useWidgetConfig(widgetId, routeId, defaultConfig)));
        expect(result.current[0]).toEqual(defaultConfig);

        // Force overwrite individual widget config as a string instead of object
        storeRootConfig(JSON.stringify({ [widgetId]: 'bogus-string' }));
        ({ result } = renderHook(() => useWidgetConfig(widgetId, routeId, defaultConfig)));
        expect(result.current[0]).toEqual(defaultConfig);

        // Force overwrite individual widget route config as a string instead of object
        storeRootConfig(JSON.stringify({ [widgetId]: { [routeId]: 'bogus-string' } }));
        ({ result } = renderHook(() => useWidgetConfig(widgetId, routeId, defaultConfig)));
        expect(result.current[0]).toEqual(defaultConfig);

        // Force overwrite top level config as an invalid JSON object
        storeRootConfig('{ "test": ["}""a }');
        ({ result } = renderHook(() => useWidgetConfig(widgetId, routeId, defaultConfig)));
        expect(result.current[0]).toEqual(defaultConfig);

        // Force overwrite individual widget route config with incompatible types
        storeRootConfig(
            JSON.stringify({
                [widgetId]: {
                    [routeId]: {
                        invalidKey: 'test',
                        sort: [
                            'valid key',
                            'but',
                            'invalid value type',
                            '(should be a string, not array)',
                        ],
                        filter: true, // Should also be a string
                    },
                },
            })
        );
        ({ result } = renderHook(() => useWidgetConfig(widgetId, routeId, defaultConfig)));
        expect(result.current[0]).toEqual(defaultConfig);
    });
});
