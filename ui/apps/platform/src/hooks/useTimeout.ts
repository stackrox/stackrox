import { useCallback, useEffect, useRef, useMemo } from 'react';

export type UseTimeoutReturn<Args extends unknown[]> = [
    (delay: number, ...args: Args) => void,
    () => void,
];

/**
 * A hook that calls a callback after a delay via setTimeout. Automatically
 * cleans up the timeout on unmount. The execute callback function expects the timeout
 * delay as the first argument and the callback arguments as the rest of the arguments.
 *
 * @param callback The callback to call after the delay
 * @returns A tuple containing a function to execute the callback and a function to cancel the pending timeout
 */
export default function useTimeout<Return, Args extends unknown[]>(
    callback: (...args: Args) => Return
): UseTimeoutReturn<Args> {
    const timeoutRef = useRef<ReturnType<typeof setTimeout>>();
    const callbackRef = useRef(callback);

    function cleanup() {
        if (typeof timeoutRef.current !== 'undefined') {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = undefined;
        }
    }

    useEffect(() => {
        callbackRef.current = callback;
    }, [callback]);

    useEffect(() => cleanup, []);

    const execCallback = useCallback((delay: number, ...args: Args) => {
        cleanup();
        timeoutRef.current = setTimeout(() => callbackRef.current(...args), delay);
    }, []);

    return useMemo(() => [execCallback, cleanup], [execCallback]);
}
