import React from 'react';
import PoliciesTablePage from './PoliciesTablePage';

function PoliciesPage() {
    return (
        <div data-testid="policies-placeholder">
            Policies
            <PoliciesTablePage />
        </div>
    );
}

export default PoliciesPage;
