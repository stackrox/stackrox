import { useCallback, useState } from 'react';
import isPlainObject from 'lodash/isPlainObject';

import { WidgetConfigStorage, RouteId, WidgetId, WidgetConfig } from 'types/widgetConfig';

export type UseWidgetConfigReturn<ConfigT extends WidgetConfig, UpdateAction> = [
    ConfigT,
    (action: UpdateAction) => Promise<ConfigT>,
];

export const defaultStorageKey = 'widgetConfigurations';

function loadConfigs(): WidgetConfigStorage {
    try {
        const configs = localStorage.getItem(defaultStorageKey) ?? '{}';
        return JSON.parse(configs) as WidgetConfigStorage;
    } catch (err) {
        return {};
    }
}

// This attempts to do basic runtime checks of the stored value to avoid app errors
// in the case of corrupted localStorage, usage from plain JS files, etc.
//
// This will ensure that the returned config:
//  - is a regular object
//  - contains all properties specified on the `defaultConfig` value provided to the hook
//  - has property values that match the types on the provided `defaultConfig`
//
// This is not an exhaustive type check (won't check array length, union types, ...) but
// should cover most possible error states.
function loadSafeConfig<T extends WidgetConfig>(
    widgetId: string,
    routeId: string,
    defaultConfig: T
): T {
    const rootConfigs = loadConfigs();
    const parsedConfig = rootConfigs[widgetId]?.[routeId] ?? {};
    const configObject = isPlainObject(parsedConfig) ? parsedConfig : {};
    const cleanConfig = {};
    Object.entries(defaultConfig).forEach(([defaultKey, defaultValue]) => {
        if (configObject[defaultKey] && typeof configObject[defaultKey] === typeof defaultValue) {
            cleanConfig[defaultKey] = configObject[defaultKey];
        }
    });
    return { ...defaultConfig, ...cleanConfig };
}

function saveConfigs(newConfigs: WidgetConfigStorage): Promise<void> {
    return new Promise((resolve, reject) => {
        try {
            localStorage.setItem(defaultStorageKey, JSON.stringify(newConfigs));
            resolve();
        } catch (error) {
            reject(error);
        }
    });
}

/**
 * Hook used to persist configuration for UI widgets. The current implementation uses
 * localStorage, but this may be updated to load from the server in the future. Separate
 * configurations are stored for the unique combination of a `widgetId` and a `routeId`, which
 * can be used to store different configs for the same widget in multiple locations throughout
 * the app.
 *
 * @param widgetId A unique identifier for a widget. All instances
 *      of a widget sharing this identifier should use the same data type for
 *      configuration storage.
 * @param routeId
 *      A route that identifies a specific instance of a configuration for
 *      a widget. This is typically the URL pathname for a widget, but can be any `string`.
 * @param defaultConfig
 *      The default configuration to be used for this widget when initializing
 *      storage, or as a fallback in case of errors.
 * @param reducer
 *      An optional update function used to make custom state updates to a widget config.
 *
 * @template ConfigT
 *      The object type used for config storage.
 * @template ActionT
 *      The object type provided to the update function for updating the config. Defaults
 *      to an object that is a partial version of the config type as the default update function
 *      is a shallow object merge.
 *
 * @returns
 *      A 2-tuple of the current state of the configuration and a function to update the
 *      configuration. The update function defaults to merging the provided object with the current
 *      state if no `reducer` parameter is provided.
 */
function useWidgetConfig<ConfigT extends WidgetConfig, ActionT = Partial<ConfigT>>(
    widgetId: WidgetId,
    routeId: RouteId,
    defaultConfig: ConfigT,
    reducer?: (oldConfig: ConfigT, action: ActionT) => ConfigT
): UseWidgetConfigReturn<ConfigT, ActionT> {
    const [widgetRouteConfig, setWidgetRouteConfig] = useState<ConfigT>(() => {
        return loadSafeConfig<ConfigT>(widgetId, routeId, defaultConfig);
    });

    const configUpdateFn = useCallback(
        (config: any) => {
            const nextValue = reducer
                ? reducer(widgetRouteConfig, config)
                : { ...widgetRouteConfig, ...config };
            // This helps ensure type-safety by applying the updated config on top of
            // the defaults, e.g. in cases where a custom reducer that deleted these properties.
            // This is not possible in TypeScript but will prevent plain JS implementations from
            // breaking TS consumers at runtime.
            const mergedDefault: ConfigT = { ...defaultConfig, ...nextValue };

            // Note, when persisting changes we need to load in the saved configs for all
            // widgets instead of keeping the value in state because it is possible
            // that other widgets will save changes after this instance of the hook has been initialized.
            // If we kept the top level config value in state, unrelated changes would be overwritten.
            const rootConfigs = loadConfigs();
            const widgetConfigs = rootConfigs[widgetId] ?? {};
            const newConfigs = {
                ...rootConfigs,
                [widgetId]: {
                    ...widgetConfigs,
                    [routeId]: {
                        ...widgetRouteConfig,
                        ...mergedDefault,
                    },
                },
            };
            setWidgetRouteConfig(mergedDefault);
            return saveConfigs(newConfigs).then(() => mergedDefault);
        },
        [widgetId, routeId, defaultConfig, reducer, widgetRouteConfig]
    );

    return [widgetRouteConfig, configUpdateFn];
}

export default useWidgetConfig;
