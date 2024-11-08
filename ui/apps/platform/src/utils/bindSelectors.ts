/* eslint-disable @typescript-eslint/no-unsafe-function-type */
/**
 * Given the map of selectors which work on a particular slice of a global Redux state,
 * and a slicer that extracts this slice form a global state, returns map of selectors
 * with the same keys but with selectors that work on a global state.
 *
 * @param slicer function that takes global state and returns sub-state (slice)
 * @param selectors map of Redux selectors that take / work on sub-state of the global state
 * @returns map of selectors that can work on a the global state
 */
export default function bindSelectors<T extends Record<string, Function>>(
    slicer: Function,
    selectors: T
): T {
    return Object.keys(selectors).reduce(
        (boundSelectors, selector) => ({
            ...boundSelectors,
            [selector]: (state, ...args): Record<string, unknown> =>
                // eslint-disable-next-line @typescript-eslint/no-unsafe-return
                selectors[selector](slicer(state), ...args),
        }),
        {} as T
    );
}
