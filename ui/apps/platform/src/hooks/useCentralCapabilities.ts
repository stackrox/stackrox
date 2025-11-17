import { useMemo } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';
import type { CentralCapabilitiesFlags } from 'services/MetadataService';

export type IsCentralCapabilityAvailable = (
    centralCapabilityFlag: CentralCapabilitiesFlags
) => boolean;

type UseCentralCapabilityResult = {
    isCentralCapabilityAvailable: IsCentralCapabilityAvailable;
};

function useCentralCapabilities(): UseCentralCapabilityResult {
    const centralCapabilities = useSelector(selectors.getCentralCapabilities);

    const isCentralCapabilityAvailable = useMemo(
        () =>
            (centralCapabilityFlag: CentralCapabilitiesFlags): boolean => {
                const centralCapabilitiesStatus = centralCapabilities[centralCapabilityFlag];
                if (centralCapabilitiesStatus === 'CapabilityDisabled') {
                    return false;
                }

                return true;
            },
        [centralCapabilities]
    );

    return { isCentralCapabilityAvailable };
}

export default useCentralCapabilities;
