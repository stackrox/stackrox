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
        searchParams = {}
    ) {
        return generatePathWithQuery(
            path,
            pathParams,
            selectedScanConfigName
                ? { ...searchParams, scanSchedule: selectedScanConfigName }
                : searchParams
        );
    }

    function navigateWithScanConfigQuery(
        path,
        pathParams: Partial<Record<string, unknown>> = {},
        searchParams = {}
    ) {
        const generatedPath = generatePathWithScanConfig(path, pathParams, searchParams);
        history.push(generatedPath);
    }

    return { navigateWithScanConfigQuery, generatePathWithScanConfig };
};

export default useScanConfigRouter;
