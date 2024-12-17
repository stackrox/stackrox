import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import PoliciesPage from 'Containers/Policies/PoliciesPage';
import PolicyCategoriesPage from 'Containers/PolicyCategories/PolicyCategoriesPage';

function PolicyManagementPage() {
    return (
        <Routes>
            <Route index element={<Navigate to="policies" />} />
            <Route path="policies/:policyId?/:command?" element={<PoliciesPage />} />
            <Route path="policy-categories" element={<PolicyCategoriesPage />} />
        </Routes>
    );
}

export default PolicyManagementPage;
