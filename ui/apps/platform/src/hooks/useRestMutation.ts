import { useState } from 'react';

export type RequestStatus = 'idle' | 'loading' | 'success' | 'error';

export type MutationOptions<Result> = {
    /** A function to call when the mutation is successful */
    onSuccess?: (data: Result) => void;
    /** A function to call when the mutation fails */
    onError?: (error: unknown) => void;
    /** A function to call after the mutation is successful or fails */
    onSettled?: (data: Result | undefined, error: unknown) => void;
};

export type UseRestMutationReturn<Payload, Result> = {
    /** The result of the mutation. */
    data: Result | undefined;
    /** Whether the mutation is currently idle, i.e. before the mutation is first triggered */
    isIdle: boolean;
    /** Whether the mutation is currently loading */
    isLoading: boolean;
    /** Whether the mutation was successful */
    isSuccess: boolean;
    /** Whether the mutation failed */
    isError: boolean;
    /** The current status of the mutation lifecycle */
    status: RequestStatus;
    /** The error that occurred during the mutation */
    error: unknown;
    /**
     * A function to trigger the mutation. Accepts an optional object to run callbacks for
     * this specific mutation.
     */
    mutate: (payload: Payload, localOptions?: MutationOptions<Result>) => void;
    /** A function to reset the mutation to its initial state and clear any data or error */
    reset: () => void;
};

const defaultState = { data: undefined, status: 'idle', error: undefined } as const;

/**
 * A hook used for general purpose REST mutations. The API is a subset of the one provided by
 * `react-query`'s `useMutation` hook.
 *
 * @param requestFn The function that will be called when the mutation is triggered.
 * @param options Options for the mutation to run callbacks globally each time the mutation is triggered. These will
 *                be called before the local callbacks passed to the `mutate` function.
 * @returns An object containing the current status of the mutation, the result of the mutation,
 *          and a function to trigger the mutation.
 */
export default function useRestMutation<Payload, Result>(
    requestFn: (payload: Payload) => Promise<Result>,
    options: MutationOptions<Result> = {}
): UseRestMutationReturn<Payload, Result> {
    const [state, setState] = useState<{
        data: Result | undefined;
        status: RequestStatus;
        error: unknown;
    }>(defaultState);
    const { status } = state;

    const mutate = (payload: Payload, localOptions: MutationOptions<Result> = {}) => {
        const request = requestFn(payload);

        setState((prevState) => ({ ...prevState, status: 'loading', error: undefined }));

        // Store the result of the request callbacks in local variables so that we can
        // call the `onSettled` callback with the correct data or error.
        let mutationData: Result | undefined;
        let mutationError: unknown;

        request
            .then((data: Result) => {
                mutationData = data;
                setState({ data, status: 'success', error: undefined });
                options.onSuccess?.(data);
                localOptions.onSuccess?.(data);
            })
            .catch((error: unknown) => {
                mutationError = error;
                setState({ data: undefined, status: 'error', error });
                options.onError?.(error);
                localOptions.onError?.(error);
            })
            .finally(() => {
                options.onSettled?.(mutationData, mutationError);
                localOptions.onSettled?.(mutationData, mutationError);
            });
    };

    function reset() {
        setState(defaultState);
    }

    return {
        ...state,
        isIdle: status === 'idle',
        isLoading: status === 'loading',
        isSuccess: status === 'success',
        isError: status === 'error',
        mutate,
        reset,
    };
}
