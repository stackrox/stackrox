/* eslint-disable no-console */
import { useEffect, useState } from 'react';
import { JsonValue } from 'utils/type.utils';

declare global {
    interface WindowEventMap {
        'use-local-storage': StorageEvent;
    }
}

type StateUpdater<State> = State | ((previousValue: State) => State);

export type UseLocalStorageReturn<Storage> = [Storage, (t: StateUpdater<Storage>) => void];

/**
 * A hook that allows you to store a value in local storage and have it automatically
 * synced across all instances of the hook on the page. If a previous stored value exists and
 * is valid, it will be used instead of the initial value.
 *
 * @param key
 *      The key to use for the local storage item
 * @param initialValue
 *      The initial value to use if no value is stored
 * @param isValidPredicate
 *      A type predicate that returns true if the stored value is valid, ensuring that the returned value
 *      is of the correct type at runtime.
 * @returns
 *      A tuple containing the stored value and a function to update it
 */
function useLocalStorage<Storage extends JsonValue>(
    key: string,
    initialValue: Storage,
    isValidPredicate: (rawValue: JsonValue) => rawValue is Storage
): UseLocalStorageReturn<Storage> {
    const [storedValue, setInternalStoredValue] = useState<Storage>(() => {
        try {
            // Load any previously stored value, if it exists and is valid
            const item = window.localStorage.getItem(key);
            const parsedItem = JSON.parse(item ?? 'null');
            return isValidPredicate(parsedItem) ? parsedItem : initialValue;
        } catch (error) {
            // On error, return the initial value
            return initialValue;
        }
    });

    function setStoredValue(newValue: StateUpdater<Storage>): unknown {
        try {
            const valueToStore = newValue instanceof Function ? newValue(storedValue) : newValue;
            const stringifiedValue = JSON.stringify(valueToStore);
            // Save to local storage and dispatch custom event to notify other hook instances
            window.localStorage.setItem(key, stringifiedValue);
            window.dispatchEvent(
                new StorageEvent('use-local-storage', { key, newValue: stringifiedValue })
            );
            return undefined;
        } catch (error: unknown) {
            return error;
        }
    }

    // Subscribe to storage events from other instances of this hook
    function storageChangeListener(event: StorageEvent) {
        if (event.key !== key) {
            return;
        }

        try {
            const parsedValue = JSON.parse(event.newValue ?? 'null');
            if (isValidPredicate(parsedValue)) {
                setInternalStoredValue(parsedValue);
            } else {
                console.warn(
                    'An invalid value was set in local storage, ignoring it.',
                    parsedValue
                );
            }
        } catch (error: unknown) {
            console.warn('Failed to parse incoming JSON value', error);
        }
    }

    useEffect(() => {
        // 'storage' to handle events from other tabs, 'local-storage' to handle events from other hooks
        window.addEventListener('storage', storageChangeListener);
        window.addEventListener('use-local-storage', storageChangeListener);
        return () => {
            window.removeEventListener('storage', storageChangeListener);
            window.removeEventListener('use-local-storage', storageChangeListener);
        };
    });

    return [storedValue, setStoredValue];
}

export default useLocalStorage;
