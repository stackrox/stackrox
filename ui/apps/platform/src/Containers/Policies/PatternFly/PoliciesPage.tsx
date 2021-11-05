import React from 'react';
import { useLocation, useParams } from 'react-router-dom';

import { parsePoliciesSearchString } from './policies.utils';
import PoliciesTablePage from './PoliciesTablePage';
import PolicyPage from './PolicyPage';

function PoliciesPage() {
    /*
     * TODO ROX-8488: Support filter and action in search query string of policies URL
     *
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
    const { search } = useLocation();
    const { action } = parsePoliciesSearchString(search);
    const { policyId } = useParams();

    return policyId || action ? (
        <PolicyPage action={action} policyId={policyId} />
    ) : (
        <PoliciesTablePage />
    );
}

export default PoliciesPage;
