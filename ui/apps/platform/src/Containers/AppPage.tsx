import React, { ReactElement } from 'react';
import { Route, Routes } from 'react-router-dom';

import {
    loginPath,
    testLoginResultsPath,
    authResponsePrefix,
    authorizeRoxctlPath,
} from 'routePaths';
import LoadingSection from 'Components/PatternFly/LoadingSection';
import AuthenticatedRoutes from 'Containers/MainPage/AuthenticatedRoutes';
import LoginPage from 'Containers/Login/LoginPage';
import TestLoginResultsPage from 'Containers/Login/TestLoginResultsPage';
import AppPageTitle from 'Containers/AppPageTitle';
import AppPageFavicon from 'Containers/AppPageFavicon';

function AppPage(): ReactElement {
    return (
        <>
            <AppPageTitle />
            <AppPageFavicon />
            <Routes>
                <Route path={loginPath} element={<LoginPage />} />
                <Route path={authorizeRoxctlPath} element={<LoginPage authorizeRoxctlMode />} />
                <Route path={testLoginResultsPath} element={<TestLoginResultsPage />} />
                <Route path={authResponsePrefix} element={<LoadingSection />} />
                <Route path="*" element={<AuthenticatedRoutes />} />
            </Routes>
        </>
    );
}

export default AppPage;
