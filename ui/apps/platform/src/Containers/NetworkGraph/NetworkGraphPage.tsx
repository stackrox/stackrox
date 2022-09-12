import React from 'react';
import { useLocation } from 'react-router-dom';

import parseURL from 'utils/URLParser';

function NetworkGraphPage() {
    const location = useLocation();
    const { pathname, search } = location.pathname;

    const workflowState = parseURL({ pathname, search });
    return (
        <div>
            <h1>Network Graph</h1>
        </div>
    );
}

export default NetworkGraphPage;
