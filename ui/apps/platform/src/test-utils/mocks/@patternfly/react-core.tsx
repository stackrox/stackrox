import * as PFReactCore from '@patternfly/react-core';

const { debounce, ...rest } = jest.requireActual('@patternfly/react-core');

// Overrides the PF `debounce` function to do nothing other than return the original function. This
// can be used to avoid issues in tests that result from state updates in a debounced function.
export const mockDebounce = { ...rest, debounce: (fn: () => void) => fn } as jest.Mock<
    typeof PFReactCore
>;
