import * as PFReactCore from '@patternfly/react-core';

// disable because unused debounce might be specified for rest spread idiom.
/* eslint-disable @typescript-eslint/no-unused-vars */
const { debounce, ...rest } = jest.requireActual('@patternfly/react-core');
/* eslint-enable @typescript-eslint/no-unused-vars */

// Overrides the PF `debounce` function to do nothing other than return the original function. This
// can be used to avoid issues in tests that result from state updates in a debounced function.
export const mockDebounce = { ...rest, debounce: (fn: () => void) => fn } as jest.Mock<
    typeof PFReactCore
>;
