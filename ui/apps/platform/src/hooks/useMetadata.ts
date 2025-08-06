import { createContext, useContext } from 'react';

import type { MetadataContextType } from 'providers/MetadataProvider';
import type { Metadata } from 'types/metadataService.proto';

export const MetadataContext = createContext<MetadataContextType | undefined>(undefined);

type UseMetadataResult = Metadata & {
    metadata: Metadata;
    isLoadingMetadata: boolean;
    error: Error | undefined;
    isOutdatedVersion: boolean;
    refetchMetadata: () => void;
};

function useMetadata(): UseMetadataResult {
    const context = useContext(MetadataContext);
    if (context === undefined) {
        throw new Error('useMetadata must be used within a MetadataProvider');
    }

    return {
        ...context.metadata, // Spread metadata properties for backward compatibility
        metadata: context.metadata,
        isLoadingMetadata: context.isLoadingMetadata,
        error: context.error,
        isOutdatedVersion: context.isOutdatedVersion,
        refetchMetadata: context.refetchMetadata,
    };
}

export default useMetadata;
