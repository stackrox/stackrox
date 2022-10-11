import { useCallback, useEffect, useState } from 'react';
import { CancellableRequest } from 'services/cancellationUtils';

export type UseRestQueryReturn<ReturnType> = {
    data: ReturnType | undefined;
    loading: boolean;
    error?: Error;
    refetch: () => void;
};

export default function useRestQuery<ReturnType>(
    cancellableRequestFn: () => CancellableRequest<ReturnType>
): UseRestQueryReturn<ReturnType> {
    const [data, setData] = useState<ReturnType>();
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<Error | undefined>();

    const execFetch = useCallback(() => {
        const { request, cancel } = cancellableRequestFn();

        setError(undefined);

        request
            .then((result) => {
                setData(result);
                setLoading(false);
                setError(undefined);
            })
            .catch((err) => {
                setLoading(true);
                setError(err);
            });

        return cancel;
    }, [cancellableRequestFn]);

    useEffect(execFetch, [execFetch]);

    return { data, loading, error, refetch: execFetch };
}
