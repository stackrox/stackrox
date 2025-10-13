import withAuth from '../../../helpers/basicAuth';

import { assertClustersPage, visitClusters } from '../Clusters.helpers';
import {
    assertDiscoveredClustersPage,
    assertSortByColumn,
    sortByColumn,
    visitDiscoveredClusters,
} from './DiscoveredClusters.helpers';

describe('Discovered clusters', () => {
    withAuth();

    it('visits from clusters page', () => {
        visitClusters();

        cy.get('a:contains("Discovered clusters")').click();

        assertDiscoveredClustersPage();
    });

    it('visits clusters from breadcrumb link', () => {
        visitDiscoveredClusters();

        cy.get('.pf-v5-c-breadcrumb__item:nth-child(2):contains("Discovered clusters")');
        cy.get('.pf-v5-c-breadcrumb__item:nth-child(1) a:contains("Clusters")').click();

        assertClustersPage();
    });

    it('renders table head cells', () => {
        visitDiscoveredClusters();

        cy.get('th:contains("Cluster")');
        cy.get('th:contains("Status")');
        cy.get('th:contains("Type")');
        cy.get('th:contains("Provider (region)")');
        cy.get('th:contains("Cloud source")');
        cy.get('th:contains("First discovered")');
    });

    it('renders no discovered clusters', () => {
        visitDiscoveredClusters();

        // CI has no cloud sources integration.
        // RegExp distinguish phrase with or without found.
        cy.contains('h2', /^No discovered clusters$/);
    });

    // TODO it('renders table data cells with mock response', () => {});

    // TODO it('filters by Status', () => {}); // Unsecured and then All statuses

    // TODO it('filters by Type', () => {}); // AKS and then EKS and then All types

    it('sorts by Cluster', () => {
        visitDiscoveredClusters();

        const text = 'Cluster';

        sortByColumn(text);
        assertSortByColumn(
            text,
            'descending',
            '?sortOption[field]=Cluster&sortOption[direction]=desc'
        );

        sortByColumn(text);
        assertSortByColumn(
            text,
            'ascending',
            '?sortOption[field]=Cluster&sortOption[direction]=asc'
        );
    });

    it('sorts by First discovered', () => {
        visitDiscoveredClusters();

        const text = 'First discovered';

        sortByColumn(text);
        assertSortByColumn(
            text,
            'descending',
            '?sortOption[field]=Cluster%20Discovered%20Time&sortOption[direction]=desc'
        );

        sortByColumn(text);
        assertSortByColumn(
            text,
            'ascending',
            '?sortOption[field]=Cluster%20Discovered%20Time&sortOption[direction]=asc'
        );
    });
});
