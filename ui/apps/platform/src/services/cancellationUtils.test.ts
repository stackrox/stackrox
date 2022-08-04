import { CancelledPromiseError, makeCancellableAxiosRequest } from './cancellationUtils';

// Creates a Promise that will settle on the next tick of the event loop
function makeEventLoopPromise(signal: AbortSignal): Promise<{ message: string }> {
    return new Promise((resolve, reject) => {
        process.nextTick(() => {
            if (signal.aborted) {
                reject(new Error('canceled'));
            } else {
                resolve({ message: 'success' });
            }
        });
    });
}

describe('makeCancellableAxiosRequest', () => {
    it('should resolve the Promise when the cancel function is not called', () => {
        const { request } = makeCancellableAxiosRequest((signal) => makeEventLoopPromise(signal));
        return expect(request).resolves.toStrictEqual({ message: 'success' });
    });

    it('should reject the Promise when the cancel function is called before the promise resolves', () => {
        const { request, cancel } = makeCancellableAxiosRequest((signal) =>
            makeEventLoopPromise(signal)
        );
        cancel();
        return expect(request).rejects.toBeInstanceOf(CancelledPromiseError);
    });
});
