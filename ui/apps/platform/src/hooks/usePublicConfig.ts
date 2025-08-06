import { createContext, useContext } from 'react';

import type { PublicConfigContextType } from 'providers/PublicConfigProvider';
import type { PublicConfig } from 'types/config.proto';

export const PublicConfigContext = createContext<PublicConfigContextType | undefined>(undefined);

type UsePublicConfigResult = {
    publicConfig: PublicConfig | undefined;
    isLoadingPublicConfig: boolean;
    error: Error | undefined;
    refetchPublicConfig: () => void;
};

function usePublicConfig(): UsePublicConfigResult {
    const context = useContext(PublicConfigContext);
    if (context === undefined) {
        throw new Error('usePublicConfig must be used within a PublicConfigProvider');
    }

    return {
        publicConfig: context.publicConfig,
        isLoadingPublicConfig: context.isLoadingPublicConfig,
        error: context.error,
        refetchPublicConfig: context.refetchPublicConfig,
    };
}

export default usePublicConfig;
