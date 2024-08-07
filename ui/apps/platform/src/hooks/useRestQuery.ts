import { useCallback, useEffect, useState } from 'react';
import noop from 'lodash/noop';
import { CancellableRequest, isCancellableRequest } from 'services/cancellationUtils';

export type UseRestQueryReturn<ReturnType> = {
    data: ReturnType | undefined;
    isLoading: boolean;
    error?: Error;
    refetch: () => void;
};

export default function useRestQuery<ReturnType>(
    requestFn: () => CancellableRequest<ReturnType> | Promise<ReturnType>
): UseRestQueryReturn<ReturnType> {
    const [data, setData] = useState<ReturnType>();
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<Error | undefined>();

    const execFetch = useCallback(() => {
        let isMounted = true;
        const requestResult = requestFn();
        const request = isCancellableRequest(requestResult) ? requestResult.request : requestResult;
        const cancel = isCancellableRequest(requestResult) ? requestResult.cancel : noop;

        setError(undefined);
        setIsLoading(true);

        request
            .then((result) => {
                if (isMounted) {
                    setData(result);
                    setIsLoading(false);
                    setError(undefined);
                }
            })
            .catch((err) => {
                if (isMounted) {
                    setIsLoading(false);
                    setError(err);
                }
            });

        return () => {
            isMounted = false;
            cancel();
        };
    }, [requestFn]);

    useEffect(execFetch, [execFetch]);

    return { data, isLoading, error, refetch: execFetch };
}
