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
