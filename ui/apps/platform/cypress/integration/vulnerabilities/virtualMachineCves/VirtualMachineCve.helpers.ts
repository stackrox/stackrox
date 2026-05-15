import type { RouteHandler, RouteMatcherOptions } from 'cypress/types/net-stubbing';

import { visit, visitWithStaticResponseForPermissions } from '../../../helpers/visit';

export const listVirtualMachinesAlias = 'listVirtualMachines';
export const getVirtualMachineAlias = 'getVirtualMachine';

export const routeMatcherMapForVirtualMachines = {
    [listVirtualMachinesAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines?*',
    },
};

export const routeMatcherMapForVirtualMachine = {
    [getVirtualMachineAlias]: {
        method: 'GET',
        url: '/v2/virtualmachines/*',
    },
};

export function visitVirtualMachineCvesOverviewPage(
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>
) {
    visit('/main/vulnerabilities/virtual-machine-cves', routeMatcherMap, staticResponseMap);
}

export function visitVirtualMachineCvesOverviewPageWithStaticPermissions(
    resourceToAccess: Record<string, string>,
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>
) {
    visitWithStaticResponseForPermissions(
        '/main/vulnerabilities/virtual-machine-cves',
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
