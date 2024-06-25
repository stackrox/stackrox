import { useContext } from 'react';
import { useHistory } from 'react-router-dom';

import { generatePathWithQuery } from 'utils/searchUtils';

import { ScanConfigurationsContext } from '../ScanConfigurationsProvider';

const useScanConfigRouter = () => {
    const { selectedScanConfigName } = useContext(ScanConfigurationsContext);
    const history = useHistory();

    function generatePathWithScanConfig(
        path,
        pathParams: Partial<Record<string, unknown>> = {},
        searchFilter = {}
    ) {
        return generatePathWithQuery(path, pathParams, {
            customParams: selectedScanConfigName ? { scanSchedule: selectedScanConfigName } : {},
            searchFilter,
        });
    }

    function navigateWithScanConfigQuery(
        path,
        pathParams: Partial<Record<string, unknown>> = {},
        searchFilter = {}
    ) {
        const generatedPath = generatePathWithScanConfig(path, pathParams, searchFilter);
        history.push(generatedPath);
    }

    return { navigateWithScanConfigQuery, generatePathWithScanConfig };
};

export default useScanConfigRouter;
