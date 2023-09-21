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

    // ROX-15888 ROX-15985: skip until decision whether valid to assume high severity violations.
    it.skip('related entities table header should not say "0 entities" or have "page 0 of 0" if there are rows in the table', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        const entitiesKey1 = 'policies';
        const entitiesKey2 = 'deployments';
        visitVulnerabilityManagementEntities(entitiesKey1);

        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(
                    `${selectors.tableBodyRows} ${selectors.failingDeploymentCountLink}:eq(0)`
                ).click();
            },
            entitiesKey1,
            entitiesKey2
        );

        cy.get('[data-testid="side-panel"] .rt-tbody .rt-tr').then((value) => {
            const { length: numRows } = value;
            if (numRows) {
                // TODO positive tests for the numbers are more robust, pardon pun.
                cy.get('[data-testid="side-panel"] [data-testid="panel-header"]')
                    .invoke('text')
                    .then((headerText) => {
                        expect(headerText).not.to.equal('0 deployments');
                    });

                cy.get(`${selectors.sidePanel} ${selectors.paginationHeader}`)
                    .invoke('text')
                    .then((paginationText) => {
                        expect(paginationText).not.to.contain('of 0');
                    });
            }
        });
    });

    // ROX-15985: skip until decision whether valid to assume high severity violations.
    // TODO if the test survives, rewrite as described below.
    // deploymentTileLink selector is obsolete
    it.skip('should scope deployment data based on selected policy from table row click', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        const entitiesKey1 = 'policies';
        const entitiesKey2 = 'deployments';
        // policy -> related deployments list should scope policy status column by the policy x deployment row
        // in both side panel and entity page
        visitVulnerabilityManagementEntities(entitiesKey1);

        // TODO Replace first row and conditional assertion with first row which has pass?
        // That is, rewrite this test as a counterpart to the following test?
        cy.get(`${selectors.tableBodyRows}:eq(0) ${selectors.statusChips}`)
            .invoke('text')
            .then((firstPolicyStatus) => {
                interactAndWaitForVulnerabilityManagementEntity(() => {
                    cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
                }, entitiesKey1);

                cy.get(`${selectors.sidePanel} ${selectors.statusChips}:eq(0)`)
                    .invoke('text')
                    .then((selectedPolicyStatus) => {
                        expect(firstPolicyStatus).to.equal(selectedPolicyStatus);
                    });

                if (firstPolicyStatus === 'pass') {
                    cy.get(
                        '[data-testid="results-message"]:contains("No deployments have failed across this policy")'
                    );

                    interactAndWaitForVulnerabilityManagementSecondaryEntities(
                        () => {
                            cy.get(`${selectors.deploymentTileLink}:eq(0)`).click();
                        },
                        entitiesKey1,
                        entitiesKey2
                    );

                    cy.get(
                        `${selectors.sidePanel} ${selectors.statusChips}:contains('pass')`
                    ).should('exist');
                    cy.get(
                        `${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`
                    ).should('not.exist');
                }
            });
    });

    // ROX-15889 ROX-15985: skip until decision whether valid to assume high severity violations.
    it.skip('should scope deployment data based on selected policy from table count link click', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        const entitiesKey1 = 'policies';
        const entitiesKey2 = 'deployments';
        visitVulnerabilityManagementEntities(entitiesKey1);

        // Assume at least one policy has failing deployments.
        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(`${selectors.failingDeploymentCountLink}:eq(0)`).click();
            },
            entitiesKey1,
            entitiesKey2
        );

        cy.get(`${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`).should('exist');
        cy.get(`${selectors.sidePanel} ${selectors.statusChips}:contains('pass')`).should(
            'not.exist'
        );
    });

    // ROX-15934 ROX-15985: skip until decision whether valid to assume high severity violations.
    it.skip('should scope deployment data based on selected policy from entity page tab sublist', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        const entitiesKey1 = 'policies';
        const entitiesKey2 = 'deployments';
        visitVulnerabilityManagementEntities(entitiesKey1);

        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(`${selectors.failingDeploymentCountLink}:eq(0)`).click();
            },
            entitiesKey1,
            entitiesKey2
        );

        cy.get(selectors.sidePanelExternalLinkButton).click();

        // Entity single page, not side panel.
        cy.get(`${selectors.tableBodyRows} ${selectors.statusChips}:contains('fail')`).should(
            'exist'
        );
        cy.get(`${selectors.tableBodyRows} ${selectors.statusChips}:contains('pass')`).should(
            'not.exist'
        );
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
            cy.get(`${selectors.tableBodyRows}:eq(0) .rt-td:nth-child(2)`).click();
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
