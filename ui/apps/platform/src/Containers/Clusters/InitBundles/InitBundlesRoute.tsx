import React, { ReactElement } from 'react';
import { useLocation, useParams } from 'react-router-dom-v5-compat';
import qs from 'qs';

import useAuthStatus from 'hooks/useAuthStatus'; // TODO after 4.4 release

import InitBundleForm from './InitBundleForm';
import InitBundlePage from './InitBundlePage';
import InitBundlesPage from './InitBundlesPage';

function hasCreateAction(search: string) {
    const { action } = qs.parse(search, { ignoreQueryPrefix: true });
    return action === 'create';
}

function InitBundlesRoute(): ReactElement {
    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected
    const hasWriteAccessForInitBundles = hasAdminRole; // TODO after 4.4 release becomes redundant

    const { search } = useLocation();
    const { id } = useParams(); // see clustersInitBundlesPathWithParam in routePaths.ts

    /*
    // TODO after 4.4 release
    if (!hasAdminRole) {
        return <NotFoundPage />; // factor out reusable component from Body.tsx file
    }
    */

    const isCreateAction = hasWriteAccessForInitBundles && hasCreateAction(search);

    if (id) {
        return (
            <InitBundlePage hasWriteAccessForInitBundles={hasWriteAccessForInitBundles} id={id} />
        );
    }

    if (hasWriteAccessForInitBundles && isCreateAction) {
        return <InitBundleForm />;
    }

    return <InitBundlesPage hasWriteAccessForInitBundles={hasWriteAccessForInitBundles} />;
}

export default InitBundlesRoute;
