import { useContext } from 'react';
import { useHistory } from 'react-router-dom';

import { generatePathWithQuery } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';

import { ScanConfigurationsContext } from '../ScanConfigurationsProvider';

const useScanConfigRouter = () => {
    const { selectedScanConfigName } = useContext(ScanConfigurationsContext);
    const history = useHistory();

    function generatePathWithScanConfig(
        path,
        pathParams: Partial<Record<string, unknown>> = {},
        options: {
            customParams?: Record<string, string>;
            searchFilter?: SearchFilter;
        } = {}
    ) {
        const queryParams = selectedScanConfigName
            ? { ...options.customParams, scanSchedule: selectedScanConfigName }
            : options.customParams;

        return generatePathWithQuery(path, pathParams, {
            customParams: queryParams,
            searchFilter: options.searchFilter,
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
