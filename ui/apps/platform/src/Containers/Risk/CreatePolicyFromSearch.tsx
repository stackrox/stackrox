import { useNavigate } from 'react-router-dom-v5-compat';
import { Button } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied, getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { policiesBasePath } from 'routePaths';

function CreatePolicyFromSearch() {
    const navigate = useNavigate();
    const { searchFilter } = useURLSearch();

    function createPolicyFromSearch() {
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
