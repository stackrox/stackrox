import type { RouteHandler, RouteMatcherOptions } from 'cypress/types/net-stubbing';

import { graphql } from '../../../constants/apiEndpoints';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
} from '../../../helpers/request';
import { visit, visitWithStaticResponseForPermissions } from '../../../helpers/visit';

export const nodeCveBaseUrl = '/main/vulnerabilities/node-cves/cves';

// Source of truth for keys in routeMatcherMap and staticResponseMap objects.
// Overview page
export const getNodesOpname = 'getNodes';
export const getNodeCvesOpname = 'getNodeCVEs';

export const getEntityTypeCountsOpname = 'getNodeCVEEntityCounts';

// Node CVE page
export const getNodeCveMetadataOpname = 'getNodeCVEMetadata';

// Node page
export const getNodeMetadataOpname = 'getNodeMetadata';
export const getNodeVulnSummaryOpname = 'getNodeVulnSummary';
export const getNodeVulnerabilitiesOpname = 'getNodeVulnerabilities';

export const routeMatcherMapForNodes = {
    [getNodesOpname]: {
        method: 'POST',
        url: graphql(getNodesOpname),
    },
};

export const routeMatcherMapForNodeCves = {
    [getNodeCvesOpname]: {
        method: 'POST',
        url: graphql(getNodeCvesOpname),
    },
};

export const routeMatcherMapForEntityCounts = {
    [getEntityTypeCountsOpname]: {
        method: 'POST',
        url: graphql(getEntityTypeCountsOpname),
    },
};

export const routeMatcherMapForNodeCveMetadata = {
    [getNodeCveMetadataOpname]: {
        method: 'POST',
        url: graphql(getNodeCveMetadataOpname),
    },
};

export const routeMatcherMapForNodePage = {
    [getNodeMetadataOpname]: {
        method: 'POST',
        url: graphql(getNodeMetadataOpname),
    },
    [getNodeVulnSummaryOpname]: {
        method: 'POST',
        url: graphql(getNodeVulnSummaryOpname),
    },
    [getNodeVulnerabilitiesOpname]: {
        method: 'POST',
        url: graphql(getNodeVulnerabilitiesOpname),
    },
};

// visit
export function visitNodeCveOverviewPage(
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>,
    params?: Record<string, string>
) {
    const paramString = params ? `?${new URLSearchParams(params).toString()}` : '';
    visit(`/main/vulnerabilities/node-cves${paramString}`, routeMatcherMap, staticResponseMap);
}

export function visitNodeCvePageWithStaticPermissions(
    mockCveName: string,
    resourceToAccess: Record<string, string>,
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>
) {
    const mockNodeCvePageUrl = `${nodeCveBaseUrl}/${mockCveName}`;

    return visitWithStaticResponseForPermissions(
        mockNodeCvePageUrl,
        {
            body: { resourceToAccess },
        },
        routeMatcherMap,
        staticResponseMap
    );
}

export function visitFirstNodeLinkFromTable(): Cypress.Chainable<string> {
    // Get the name of the first node in the table and pass it to the caller
    return cy
        .get('tbody tr td[data-label="Node"] a')
        .first()
        .then(($link) => {
            interactAndWaitForResponses(
                () => cy.wrap($link).click(),
                getRouteMatcherMapForGraphQL(['getNodeMetadata'])
            );
            return cy.wrap($link.text());
        });
}
