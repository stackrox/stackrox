import { useCallback, useEffect, useState } from 'react';
import { CancellableRequest } from 'services/cancellationUtils';

export type UseRestQueryReturn<ReturnType, ErrorType extends Error> = {
    data: ReturnType | undefined;
    loading: boolean;
    error?: ErrorType;
    refetch: () => void;
};

export default function useRestQuery<ReturnType, ErrorType extends Error>(
    cancellableRequestFn: () => CancellableRequest<ReturnType>
): UseRestQueryReturn<ReturnType, ErrorType> {
    const [data, setData] = useState<ReturnType>();
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<ErrorType | undefined>();

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
