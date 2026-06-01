import { useQuery, type UseQueryOptions, type QueryKey } from '@tanstack/react-query';

import { isCancellableRequest, type CancellableRequest } from 'services/cancellationUtils';

type ServiceFn<T> = (() => CancellableRequest<T>) | (() => Promise<T>);

/**
 * Wraps an existing service function for use with React Query.
 * Handles both regular Promise-based services and CancellableRequest services.
 * For CancellableRequest, wires the cancel function to React Query's abort signal.
 */
export function useServiceQuery<T>(
    queryKey: QueryKey,
    serviceFn: ServiceFn<T>,
    options?: Omit<UseQueryOptions<T>, 'queryKey' | 'queryFn'>
) {
    return useQuery<T>({
        queryKey,
        queryFn: ({ signal }) => {
            const result = serviceFn();

            if (isCancellableRequest(result)) {
                signal?.addEventListener('abort', () => {
                    result.cancel();
                });
                return result.request;
            }

            return result;
        },
        ...options,
    });
}
