import useURLParameter from 'hooks/useURLParameter';

import { DetailsTab, isDetailsTab } from '../types';

export type UseDetailsTabParameterReturn = [DetailsTab, (newTab: DetailsTab) => void];

export default function useDetailsTabParameter(): UseDetailsTabParameterReturn {
    const [detailsTab, setDetailsTab] = useURLParameter('detailsTab', 'Vulnerabilities');
    const tabValue = isDetailsTab(detailsTab) ? detailsTab : 'Vulnerabilities';
    return [tabValue, setDetailsTab];
}
