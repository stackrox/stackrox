import React from 'react';
import { useLocation, useParams } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';
import { getQueryObject, BasePageAction } from 'utils/queryStringUtils';

import ScanConfigsTablePage from './Table/ScanConfigsTablePage';
import ScanConfigPage from './ScanConfigPage';

function SchedulingPage() {
    /*
     * Examples of urls for ScanConfigPage:
     * /main/policymanagement/policies/:policyId
     * /main/policymanagement/policies/:policyId?action=edit
     * /main/policymanagement/policies?action=create
     *
     * Examples of urls for PolicyTablePage:
     * /main/policymanagement/policies
     * /main/policymanagement/policies?s[Lifecycle Stage]=BUILD
     * /main/policymanagement/policies?s[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY
     * /main/policymanagement/policies?s[Lifecycle State]=RUNTIME&s[Severity]=CRITICAL_SEVERITY
     */
    const location = useLocation();
    const { search } = location;
    const { searchFilter, setSearchFilter } = useURLSearch();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { scanConfigId } = useParams();

    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');

    if (action || scanConfigId) {
        return (
            <ScanConfigPage
                hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                pageAction={action as BasePageAction}
                scanConfigId={scanConfigId}
            />
        );
    }

    return (
        <ScanConfigsTablePage
            hasWriteAccessForCompliance={hasWriteAccessForCompliance}
            handleChangeSearchFilter={setSearchFilter}
            searchFilter={searchFilter}
        />
    );
}

export default SchedulingPage;
