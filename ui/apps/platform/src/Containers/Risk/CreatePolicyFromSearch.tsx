import { useNavigate } from 'react-router-dom-v5-compat';
import { Button } from '@patternfly/react-core';

import useAnalytics, { RISK_CREATE_POLICY_CLICKED } from 'hooks/useAnalytics';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { policiesBasePath } from 'routePaths';

function CreatePolicyFromSearch() {
    const navigate = useNavigate();
    const { analyticsTrack } = useAnalytics();
    const { searchFilter } = useURLSearch();

    function createPolicyFromSearch() {
        analyticsTrack(RISK_CREATE_POLICY_CLICKED);
        const searchFilterQueryString = getUrlQueryStringForSearchFilter(searchFilter);
        navigate(`${policiesBasePath}?action=generate&${searchFilterQueryString}`);
    }

    const isDisabled = !getHasSearchApplied(searchFilter);

    return (
        <Button variant="secondary" onClick={createPolicyFromSearch} isDisabled={isDisabled}>
            Create policy
        </Button>
    );
}

export default CreatePolicyFromSearch;
