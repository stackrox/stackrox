import { useContext } from 'react';
import { useHistory } from 'react-router-dom';

import { generatePathWithQuery } from 'utils/searchUtils';

import { ScanConfigurationsContext } from '../ScanConfigurationsProvider';

const useScanConfigRouter = () => {
    const { selectedScanConfigName } = useContext(ScanConfigurationsContext);
    const history = useHistory();

    function generatePathWithScanConfig(path, pathParams: Partial<Record<string, unknown>> = {}) {
        return generatePathWithQuery(
            path,
            pathParams,
            selectedScanConfigName ? { scanSchedule: selectedScanConfigName } : {}
        );
    }

    function navigateWithScanConfigQuery(path, pathParams: Partial<Record<string, unknown>> = {}) {
        const generatedPath = generatePathWithScanConfig(path, pathParams);
        history.push(generatedPath);
    }

    return { navigateWithScanConfigQuery, generatePathWithScanConfig };
};

export default useScanConfigRouter;
