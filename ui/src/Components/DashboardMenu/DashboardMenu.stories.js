import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import DashboardMenu from './DashboardMenu';

export default {
    title: 'DashboardMenu',
    component: DashboardMenu,
};

export const withApplicationTypes = () => {
    const options = [
        { label: 'Cluster', link: '/main/configmanagement/clusters' },
        { label: 'Namespacce', link: '/main/configmanagement/namespaces' },
        { label: 'Node', link: '/main/configmanagement/nodes' },
    ];
    return (
        <MemoryRouter>
            <DashboardMenu text="Application & Infrastructure" options={options} />
        </MemoryRouter>
    );
};

export const withRBACTypes = () => {
    const options = [
        { label: 'Users & Groups', link: '/main/configmanagement/subjects' },
        { label: 'Service Accounts', link: '/main/configmanagement/serviceaccounts' },
        { label: 'Roles', link: '/main/configmanagement/roles' },
    ];
    return (
        <MemoryRouter>
            <DashboardMenu text="RBAC Visibility & Configuration" options={options} />
        </MemoryRouter>
    );
};
