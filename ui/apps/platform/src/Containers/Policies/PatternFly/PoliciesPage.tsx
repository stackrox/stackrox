import React from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { getHasReadPermission, getHasReadWritePermission } from 'reducers/roles';
import { policiesBasePathPatternFly } from 'routePaths';
import { SearchFilter } from 'types/search';

import { getSearchStringForFilter, parsePoliciesSearchString } from './policies.utils';
import PoliciesTablePage from './Table/PoliciesTablePage';
import PolicyPage from './PolicyPage';

const permissionsSelector = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions,
});

function PoliciesPage() {
    /*
     * Examples of urls for PolicyPage:
     * /main/policies/:policyId
     * /main/policies/:policyId?action=edit
     * /main/policies?action=create
     *
     * Examples of urls for PolicyTablePage:
     * /main/policies
     * /main/policies?s[Lifecycle Stage]=BUILD
     * /main/policies?s[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY
     * /main/policies?s[Lifecycle State]=RUNTIME&s[Severity]=CRITICAL_SEVERITY
     */
    const history = useHistory();
    const { search } = useLocation();
    const { pageAction, searchFilter } = parsePoliciesSearchString(search);
    const { policyId } = useParams();

    const { userRolePermissions } = useSelector(permissionsSelector);
    const hasReadAccessForPolicy = getHasReadPermission('Policy', userRolePermissions);
    const hasWriteAccessForPolicy = getHasReadWritePermission('Policy', userRolePermissions);

    function handleChangeSearchFilter(changedSearchFilter: SearchFilter) {
        // Browser history has only the most recent search filter.
        console.log('handleChangeSearchFilter', changedSearchFilter);
        history.replace({
            pathname: policiesBasePathPatternFly,
            search: getSearchStringForFilter(changedSearchFilter),
        });
    }

    if (!hasReadAccessForPolicy) {
        return <div>TODO</div>;
    }

    if (pageAction || policyId) {
        return (
            <PolicyPage
                hasWriteAccessForPolicy={hasWriteAccessForPolicy}
                pageAction={pageAction}
                policyId={policyId}
            />
        );
    }

    return (
        <PoliciesTablePage
            hasWriteAccessForPolicy={hasWriteAccessForPolicy}
            handleChangeSearchFilter={handleChangeSearchFilter}
            searchFilter={searchFilter}
        />
    );
}

export default PoliciesPage;
