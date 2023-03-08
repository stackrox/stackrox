import useURLParameter from 'hooks/useURLParameter';

import { CveStatusTab, isValidCveStatusTab } from '../types';

export type UseCveStatusTabParameterReturn = [CveStatusTab, (newTab: CveStatusTab) => void];

export default function useCveStatusTabParameter(): UseCveStatusTabParameterReturn {
    const [cveStatusTab, setCveStatusTab] = useURLParameter('cveStatusTab', 'Observed');
    const tabValue = isValidCveStatusTab(cveStatusTab) ? cveStatusTab : 'Observed';
    return [tabValue, setCveStatusTab];
}
