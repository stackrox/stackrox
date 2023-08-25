import { useRef, useEffect, useState, useCallback } from 'react';
import throttle from 'lodash/throttle';

type UseResizeObserverOptions = { throttleInterval: number };

/**
 * Hook to listen for resize events on a target element.
 */
function useResizeObserver(
    element: Element | null,
    { throttleInterval }: UseResizeObserverOptions = { throttleInterval: 200 }
): ResizeObserverEntry | null {
    const resizeObserverRef = useRef<ResizeObserver | null>(null);
    const [entry, setEntry] = useState<ResizeObserverEntry | null>(null);

    const disconnect = useCallback(() => resizeObserverRef.current?.disconnect(), []);

    useEffect(() => {
        if (element) {
            const callback = throttle(([firstEntry]) => setEntry(firstEntry), throttleInterval);
            resizeObserverRef.current = new ResizeObserver(callback);
            resizeObserverRef.current.observe(element);
        }

        return disconnect;
    }, [element, throttleInterval, disconnect]);

    return entry;
}

export default useResizeObserver;
