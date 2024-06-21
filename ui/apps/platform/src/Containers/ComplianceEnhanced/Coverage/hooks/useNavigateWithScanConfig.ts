import { useContext } from 'react';
import { useHistory } from 'react-router-dom';

import { generatePathWithQuery } from 'utils/searchUtils';

import { ScanConfigurationsContext } from '../ScanConfigurationsProvider';

const useScanConfigRouter = () => {
    const { selectedScanConfig } = useContext(ScanConfigurationsContext);
    const history = useHistory();

    function generatePathWithScanConfig(path, pathParams: Partial<Record<string, unknown>> = {}) {
        return generatePathWithQuery(
            path,
            pathParams,
            selectedScanConfig ? { scanSchedule: selectedScanConfig } : {}
        );
    }

    function navigateWithScanConfigQuery(path, pathParams: Partial<Record<string, unknown>> = {}) {
        const generatedPath = generatePathWithScanConfig(path, pathParams);
        history.push(generatedPath);
    }

    return { navigateWithScanConfigQuery, generatePathWithScanConfig };
};

export default useScanConfigRouter;
