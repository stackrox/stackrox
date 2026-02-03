import { createContext, useContext } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { Alert, Gallery, GalleryItem, Skeleton } from '@patternfly/react-core';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

const LoadingContext = createContext<{ isLoading: boolean }>({
    isLoading: false,
});

export type SummaryCardProps<T> = {
    data: T;
    loadingText: string;
    renderer: ({ data }: { data: NonNullable<T> }) => ReactNode;
};

/**
 * Component that handles rendering of individual summary cards. This will get the loading state
 * from the parent context and render a skeleton if the data is not yet available.
 */
export function SummaryCard<T>({ loadingText, renderer, data }: SummaryCardProps<T>) {
    const { isLoading } = useContext(LoadingContext);
    return (
        <GalleryItem>
            {isLoading || !data ? (
                <Skeleton height="100%" screenreaderText={loadingText} />
            ) : (
                renderer({ data })
            )}
        </GalleryItem>
    );
}

// Responsive widths for the summary cards, taking into account the gutter spacing
const oneThirdWidth = 'calc(33.3% - var(--pf-t--global--spacer--gutter--default))';
const oneHalfWidth = 'calc(50% - var(--pf-v5-global--gutter))';
const fullWidth = '100%';

export type SummaryCardLayoutProps = {
    error: unknown;
    isLoading: boolean;
    children: ReactNode;
    errorAlertTitle?: string;
};

/**
 * Component that encapsulates the layout for Vulnerability Management summary cards. This includes
 * handling of loading and error states, and providing non-nullable data to the summary cards.
 */
export function SummaryCardLayout({
    error,
    isLoading,
    children,
    errorAlertTitle = 'There was an error loading the summary data for this entity',
}: SummaryCardLayoutProps): ReactElement {
    return (
        <LoadingContext.Provider value={{ isLoading }}>
            {error ? (
                <Alert title={errorAlertTitle} component="p" isInline variant="danger">
                    {getAxiosErrorMessage(error)}
                </Alert>
            ) : (
                <Gallery
                    hasGutter
                    style={{ minHeight: '120px' }}
                    minWidths={{ '2xl': oneThirdWidth, md: oneHalfWidth, sm: fullWidth }}
                >
                    {children}
                </Gallery>
            )}
        </LoadingContext.Provider>
    );
}
