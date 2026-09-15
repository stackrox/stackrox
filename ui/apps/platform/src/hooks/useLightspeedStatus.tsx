import { createContext, useContext } from 'react';

import type { LightspeedStatusContextType } from 'providers/LightspeedStatusProvider';

export const LightspeedStatusContext = createContext<LightspeedStatusContextType | undefined>(
    undefined
);

function useLightspeedStatus(): LightspeedStatusContextType {
    const context = useContext(LightspeedStatusContext);
    if (context === undefined) {
        return {
            isAvailable: false,
            isLoading: false,
            error: new Error(
                'useLightspeedStatus must be used within a LightspeedStatusProvider - returning static default values'
            ),
        };
    }

    return context;
}

export default useLightspeedStatus;
