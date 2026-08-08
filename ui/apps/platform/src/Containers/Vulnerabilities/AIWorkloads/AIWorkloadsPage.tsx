import { Route, Routes } from 'react-router-dom-v5-compat';

import AIWorkloadsOverviewPage from './AIWorkloadsOverviewPage';

function AIWorkloadsPage() {
    return (
        <Routes>
            <Route index element={<AIWorkloadsOverviewPage />} />
        </Routes>
    );
}

export default AIWorkloadsPage;
