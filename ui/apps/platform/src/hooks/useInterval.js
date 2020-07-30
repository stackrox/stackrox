import { useEffect, useRef } from 'react';

/**
 * This hook is an implementation of Dan Abramov's blog post "Making setInterval Declarative with React Hooks".
 * https://overreacted.io/making-setinterval-declarative-with-react-hooks/
 *
 * @param   {function}  callback  the code you want called at every interval
 * @param   {number}  delay     the number of milliseconds between function executions
 */
export default function useInterval(callback, delay) {
    const savedCallback = useRef();

    // Remember the latest callback.
    useEffect(() => {
        savedCallback.current = callback;
    }, [callback]);

    // Set up the interval.
    useEffect(
        // eslint-disable-next-line consistent-return
        () => {
            function tick() {
                savedCallback.current();
            }
            if (delay !== null) {
                const id = setInterval(tick, delay);
                return () => clearInterval(id);
            }
        },
        [delay]
    );
}
