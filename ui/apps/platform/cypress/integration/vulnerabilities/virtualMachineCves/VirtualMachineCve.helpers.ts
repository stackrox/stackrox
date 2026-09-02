import type { RouteHandler, RouteMatcherOptions } from 'cypress/types/net-stubbing';

import { visit, visitWithStaticResponseForPermissions } from '../../../helpers/visit';

export const listVirtualMachinesAlias = 'listVirtualMachines';
export const getVirtualMachineAlias = 'getVirtualMachine';
export const getVirtualMachineCveComponentsAlias = 'getVirtualMachineCveComponents';
export const listVirtualMachineCvesAlias = 'listVirtualMachineCves';
export const listVirtualMachineComponentsAlias = 'listVirtualMachineComponents';

export const routeMatcherMapForVirtualMachines = {
    [listVirtualMachinesAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines/vms?*',
    },
};

export const routeMatcherMapForVirtualMachine = {
    [getVirtualMachineAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines/*',
    },
};

export const routeMatcherMapForVirtualMachineVulnerabilities = {
    ...routeMatcherMapForVirtualMachine,
    [listVirtualMachineCvesAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines/*/cves?*',
    },
};

export const routeMatcherMapForVirtualMachineComponents = {
    [listVirtualMachineComponentsAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines/*/components?*',
    },
};

export const routeMatcherMapForVirtualMachineCveComponents = {
    [getVirtualMachineCveComponentsAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines/*/cves/*/components',
    },
};

function virtualMachineCvesOverviewPath(params?: Record<string, string>): string {
    const query = params ? `?${new URLSearchParams(params).toString()}` : '';
    return `/main/vulnerabilities/virtual-machine-cves${query}`;
}

export function visitVirtualMachineCvesOverviewPage(
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>,
    params?: Record<string, string>
) {
    visit(virtualMachineCvesOverviewPath(params), routeMatcherMap, staticResponseMap);
}

export function visitVirtualMachineCvesOverviewPageWithStaticPermissions(
    resourceToAccess: Record<string, string>,
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>,
    params?: Record<string, string>
) {
    visitWithStaticResponseForPermissions(
        virtualMachineCvesOverviewPath(params),
        { body: { resourceToAccess } },
        routeMatcherMap,
        staticResponseMap
    );
}

export function visitVirtualMachinePage(
    virtualMachineId: string,
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>
) {
    visit(
        `/main/vulnerabilities/virtual-machine-cves/virtualmachines/${virtualMachineId}`,
        routeMatcherMap,
        staticResponseMap
    );
}

export function visitVirtualMachinePageWithStaticPermissions(
    virtualMachineId: string,
    resourceToAccess: Record<string, string>,
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>
) {
    visitWithStaticResponseForPermissions(
        `/main/vulnerabilities/virtual-machine-cves/virtualmachines/${virtualMachineId}`,
        { body: { resourceToAccess } },
        routeMatcherMap,
        staticResponseMap
    );
}
