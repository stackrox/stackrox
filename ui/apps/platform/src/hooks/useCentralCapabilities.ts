import { useMemo } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';
import { CentralCapabilitiesFlags } from 'services/MetadataService';

type UseCentralCapabilityResult = {
    isCentralCapabilityAvailable: (centralCapabilityFlag: CentralCapabilitiesFlags) => boolean;
};

function useCentralCapabilities(): UseCentralCapabilityResult {
    const centralCapabilities = useSelector(selectors.getCentralCapabilities);

    const isCentralCapabilityAvailable = useMemo(
        () =>
            (centralCapabilityFlag: CentralCapabilitiesFlags): boolean => {
                const centralCapacity = centralCapabilities[centralCapabilityFlag];
                if (centralCapacity === 'CapabilityAvailable') {
                    return true;
                }

                return false;
            },
        [centralCapabilities]
    );

    return { isCentralCapabilityAvailable };
}

export default useCentralCapabilities;
