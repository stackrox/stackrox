import * as api from '../constants/apiEndpoints';
import { clustersUrl } from '../constants/ClustersPage';
import { url as dashboardUrl } from '../constants/DashboardPage';
import navigation from '../selectors/navigation';

// Navigation

export function visitClustersFromLeftNav() {
    cy.intercept('POST', api.graphql(api.general.graphqlOps.summaryCounts)).as('getSummaryCounts');
    cy.visit(dashboardUrl);
    cy.wait('@getSummaryCounts');

    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.get(navigation.navExpandablePlatformConfiguration).click();
    cy.get(
        `${navigation.navExpandablePlatformConfiguration} + ${navigation.nestedNavLinks}:contains("Clusters")`
    ).click();
    cy.wait('@getClusters');
}

export function visitClusters() {
    cy.intercept('GET', api.clusters.list).as('getClusters');
    cy.visit(clustersUrl);
    cy.wait('@getClusters');
}
