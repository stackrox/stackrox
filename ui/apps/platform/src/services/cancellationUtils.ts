export type CancellableRequest<T> = {
    /** A Promise that has been created so that it can be cancelled via AbortSignal */
    request: Promise<T>;
    /**
     * A function that cancels the execution of the returned Promise.
     */
    cancel: () => void;
};

export function isCancellableRequest<T>(
    maybeRequest: unknown
): maybeRequest is CancellableRequest<T> {
    return (
        typeof maybeRequest === 'object' &&
        maybeRequest !== null &&
        'cancel' in maybeRequest &&
        'request' in maybeRequest
    );
}

/**
 * A subclass of Error used to signal that a Promise rejected due to cancellation.
 * This can be used to more easily separate the behavior of a cancelled
 * operation versus an actual error.
 */
export class CancelledPromiseError extends Error {
    constructor(message = 'The network request has been canceled.') {
        super(message);
        this.name = 'CanceledPromise';
    }
}

/**
 * Used to attach a AbortController to an axios request and return
 * a function that can be used to cancel the request.  Note that this only
 * cancels the original Promise instance, and will not cancel new Promises created
 * in the Promise chain.
 *
 * @param promiseProvider A function that receives an injected AbortSignal
 * and returns a Promise. The body of the function should use the AbortSignal
 * to wire up a cancellation mechanism in the returned Promise.
 *
 * @returns An object containing the returned Promise and a function used to
 * cancel that Promise.
 */
export function makeCancellableAxiosRequest<T>(
    promiseProvider: (signal: AbortSignal) => Promise<T>
): CancellableRequest<T> {
    const controller = new AbortController();
    const request = promiseProvider(controller.signal).catch((err) => {
        return err.message === 'canceled'
            ? Promise.reject(new CancelledPromiseError())
            : Promise.reject(err);
    });
    return { request, cancel: () => controller.abort() };
}
