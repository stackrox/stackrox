import React from 'react';
import { PageSection } from '@patternfly/react-core';
import { Route, Routes } from 'react-router-dom';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ExceptionRequestsPage from './ExceptionRequestsPage';
import ExceptionRequestDetailsPage from './ExceptionRequestDetailsPage';

function ExceptionManagementPage() {
    return (
        <Routes>
            <Route path="requests/:requestId" element={<ExceptionRequestDetailsPage />} />
            <Route path="*" element={<ExceptionRequestsPage />} />
            <Route
                element={
                    <PageSection variant="light">
                        <PageTitle title="Exception requests - Not Found" />
                        <PageNotFound />
                    </PageSection>
                }
            />
        </Routes>
    );
}

export default ExceptionManagementPage;
