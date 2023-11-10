import { renderHook, act } from '@testing-library/react';

import waitForNextUpdate from 'test-utils/waitForNextUpdate';
import useRestMutation from './useRestMutation';

// Utility function to track the order of callbacks as the hook transitions
// through its lifecycle.
function trackCallbacksInArray(array: string[], scope: 'global' | 'local') {
    return {
        onSuccess: (data: string) => {
            array.push(`${scope} success: ${data}`);
        },
        onError: (error: unknown) => {
            array.push(`${scope} error: ${error as string}`);
        },
        onSettled: (data: string | undefined, error: unknown) => {
            array.push(`${scope} settled: [${String(data)}, ${error as string}]`);
        },
    };
}

describe('useRestMutation hook', () => {
    it('should correctly handle success lifecycle statuses and data', async () => {
        jest.useFakeTimers();

        const requestFn = (arg: string) =>
            new Promise<string>((resolve) =>
                setTimeout(() => resolve(`called with "${arg}"`), 1000)
            );

        const callbackResults: string[] = [];

        const { result } = renderHook(() =>
            useRestMutation(requestFn, trackCallbacksInArray(callbackResults, 'global'))
        );

        // Check initial state
        expect(result.current.isIdle).toBe(true);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe(undefined);
        expect(result.current.error).toBe(undefined);

        act(() => {
            result.current.mutate('test', trackCallbacksInArray(callbackResults, 'local'));
        });

        // Check loading state
        expect(result.current.isIdle).toBe(false);
        expect(result.current.isLoading).toBe(true);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe(undefined);
        expect(result.current.error).toBe(undefined);

        // Expire timeout timer and wait for state to change
        jest.runAllTimers();
        await waitForNextUpdate(result);

        // Check success state
        expect(result.current.isIdle).toBe(false);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe('called with "test"');
        expect(result.current.error).toBe(undefined);

        expect(callbackResults).toEqual([
            'global success: called with "test"',
            'local success: called with "test"',
            'global settled: [called with "test", undefined]',
            'local settled: [called with "test", undefined]',
        ]);

        act(() => {
            result.current.reset();
        });

        // Check reset state
        expect(result.current.isIdle).toBe(true);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe(undefined);
        expect(result.current.error).toBe(undefined);
    });

    it('should correctly handle failure lifecycle statuses and data', async () => {
        jest.useFakeTimers();

        const requestFn = (arg: string) =>
            new Promise<string>((resolve, reject) =>
                // Using a 'string' instead of `Error` for simplicity
                // eslint-disable-next-line prefer-promise-reject-errors
                setTimeout(() => reject(`error with "${arg}"`), 1000)
            );

        const callbackResults: string[] = [];

        const { result } = renderHook(() =>
            useRestMutation(requestFn, trackCallbacksInArray(callbackResults, 'global'))
        );

        // Check initial state
        expect(result.current.isIdle).toBe(true);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe(undefined);
        expect(result.current.error).toBe(undefined);

        act(() => {
            result.current.mutate('test', trackCallbacksInArray(callbackResults, 'local'));
        });

        // Check loading state
        expect(result.current.isIdle).toBe(false);
        expect(result.current.isLoading).toBe(true);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe(undefined);
        expect(result.current.error).toBe(undefined);

        // Expire timeout timer and wait for state to change
        jest.runAllTimers();
        await waitForNextUpdate(result);

        // Check failure state
        expect(result.current.isIdle).toBe(false);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(true);
        expect(result.current.data).toBe(undefined);
        expect(result.current.error as string).toBe('error with "test"');

        expect(callbackResults).toEqual([
            'global error: error with "test"',
            'local error: error with "test"',
            'global settled: [undefined, error with "test"]',
            'local settled: [undefined, error with "test"]',
        ]);

        act(() => {
            result.current.reset();
        });

        // Check reset state
        expect(result.current.isIdle).toBe(true);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isSuccess).toBe(false);
        expect(result.current.isError).toBe(false);
        expect(result.current.data).toBe(undefined);
    });
});
