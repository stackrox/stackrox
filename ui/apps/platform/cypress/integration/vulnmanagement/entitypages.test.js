import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag, hasOrchestratorFlavor } from '../../helpers/features';
import {
    interactAndWaitForVulnerabilityManagementEntity,
    interactAndWaitForVulnerabilityManagementSecondaryEntities,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

describe('Entities single views', () => {
    withAuth();

    // Some tests might fail in local deployment.

    // TODO skip pending more robust criterion than deployment count
    // deploymentTileLink selector is obsolete
    it.skip('related entities tile links should unset search params upon navigation', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        const entitiesKey1 = 'clusters';

        visitVulnerabilityManagementEntities(entitiesKey1);

        // Specify td elements for Image CVEs instead of Node CVEs or Platform CVEs.
        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(`.rt-td:nth-child(3) [data-testid="fixableCvesLink"]:eq(0)`).click();
            },
            entitiesKey1,
            'image-cves'
        );

        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(selectors.backButton).click();
        }, entitiesKey1);

        cy.get(`${selectors.deploymentTileLink} [data-testid="tileLinkSuperText"]`)
            .invoke('text')
            .then((numDeployments) => {
                interactAndWaitForVulnerabilityManagementSecondaryEntities(
                    () => {
                        cy.get(selectors.deploymentTileLink).click();
                    },
                    entitiesKey1,
                    'deployments'
                );

                cy.get(
                    `[data-testid="side-panel"] [data-testid="panel-header"]:contains("${numDeployments}")`
                );
            });
    });

    it('should show a CVE description in overview when coming from cve list', () => {
        const entitiesKey = 'image-cves';
        visitVulnerabilityManagementEntities(entitiesKey);

        cy.get(`${selectors.tableBodyRowGroups}:eq(0) ${selectors.cveDescription}`)
            .invoke('text')
            .then((descriptionInList) => {
                interactAndWaitForVulnerabilityManagementEntity(() => {
                    cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
                }, entitiesKey);

                cy.get(`[data-testid="entity-overview"] ${selectors.metadataDescription}`)
                    .invoke('text')
                    .then((descriptionInSidePanel) => {
                        expect(descriptionInSidePanel).to.equal(descriptionInList);
                    });
            });
    });

    it('should show the active state in Component overview when scoped under a deployment', () => {
        const activeVulnEnabled = hasFeatureFlag('ROX_ACTIVE_VULN_MGMT');
        const entitiesKey1 = 'deployments';
        const entitiesKey2 = 'image-components';
        visitVulnerabilityManagementEntities(entitiesKey1);

        // click on the first deployment in the list
        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(
                `${selectors.tableBodyRows}:has(.rt-td:eq(2) a:contains("CVE")) .rt-td:nth-child(2)`
            )
                .first()
                .click();
        }, entitiesKey1);

        // now, go to the components for that deployment
        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(
                    'h2:contains("Related entities") ~ div ul li a:contains("image component")'
                ).click();
            },
            entitiesKey1,
            entitiesKey2
        );

        // click on the first component in that list
        // TODO Get value from cell in Active column to compare below?
        // TODO How to assert only the following 3 values in the cells?
        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(`[data-testid="side-panel"] ${selectors.tableBodyRows}:eq(0)`).click();
        }, entitiesKey2);

        if (activeVulnEnabled) {
            cy.get(`[data-testid="Active status-value"]`)
                .invoke('text')
                .then((activeStatusText) => {
                    expect(activeStatusText).to.be.oneOf(['Active', 'Inactive', 'Undetermined']);
                });
        } else {
            cy.get('.rt-th')
                .invoke('text')
                .then((tableHeaderText) => {
                    expect(tableHeaderText).not.to.contain('Active');
                });
        }
    });

    it('should show the active state in the fixable CVES widget for a single deployment', () => {
        const activeVulnEnabled = hasFeatureFlag('ROX_ACTIVE_VULN_MGMT');
        const entitiesKey = 'deployments';

        const fixableCvesFixture = 'vulnerabilities/fixableCvesForEntity.json';
        const getFixableCvesForEntity = api.graphql('getFixableCvesForEntity');
        cy.intercept('POST', getFixableCvesForEntity, {
            fixture: fixableCvesFixture,
        }).as('getFixableCvesForEntity');

        visitVulnerabilityManagementEntities(entitiesKey);

        interactAndWaitForVulnerabilityManagementEntity(() => {
            // TODO Replace .eq(1) method with :eq(0) pseudo-selector?
            // TODO Index 1 instead of 0 because row selector not limited to table body?
            cy.get(`${selectors.tableRows}`).eq(1).click();
        }, entitiesKey);

        cy.wait('@getFixableCvesForEntity');

        if (activeVulnEnabled) {
            cy.get(
                `${selectors.sidePanel} ${selectors.tableRows}:contains("CVE-2021-20231")`
            ).contains('Active');
            cy.get(
                `${selectors.sidePanel} ${selectors.tableRows}:contains("CVE-2021-20232")`
            ).contains('Inactive');
        } else {
            cy.get('.rt-th')
                .invoke('text')
                .then((tableHeaderText) => {
                    expect(tableHeaderText).not.to.contain('Active');
                });
        }
    });
});
