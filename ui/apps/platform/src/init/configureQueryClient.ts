import { QueryClient } from '@tanstack/react-query';

export default function configureQueryClient(): QueryClient {
    return new QueryClient({
        defaultOptions: {
            queries: {
                staleTime: 30_000,
                gcTime: 300_000,
                retry: 1,
                refetchOnWindowFocus: false,
            },
        },
    });
}
