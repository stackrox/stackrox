import React, { ReactElement } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import { entityPathSegment } from './accessControlPaths';
import AccessControlRouteNotFound from './AccessControlRouteNotFound';
import AccessScopes from './AccessScopes/AccessScopes';
import AuthProviders from './AuthProviders/AuthProviders';
import PermissionSets from './PermissionSets/PermissionSets';
import Roles from './Roles/Roles';

const paramId = ':entityId?';

function AccessControl(): ReactElement {
    return (
        <>
            <Routes>
                <Route index element={<Navigate to={entityPathSegment.AUTH_PROVIDER} />} />
                <Route
                    path={`${entityPathSegment.AUTH_PROVIDER}/${paramId}`}
                    element={<AuthProviders />}
                />
                <Route path={`${entityPathSegment.ROLE}/${paramId}`} element={<Roles />} />
                <Route
                    path={`${entityPathSegment.PERMISSION_SET}/${paramId}`}
                    element={<PermissionSets />}
                />
                <Route
                    path={`${entityPathSegment.ACCESS_SCOPE}/${paramId}`}
                    element={<AccessScopes />}
                />
                <Route path="*" element={<AccessControlRouteNotFound />} />
            </Routes>
        </>
    );
}

export default AccessControl;
