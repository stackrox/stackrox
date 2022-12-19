import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from '../../helpers/request';

import {
    interactAndWaitForVulnerabilityManagementEntity,
    interactAndWaitForVulnerabilityManagementSecondaryEntities,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

describe('Entities single views', () => {
    withAuth();

    // Some tests might fail in local deployment.

    it('related entities tile links should unset search params upon navigation', () => {
        const entitiesKey1 = 'clusters';
        const usingVMUpdates = hasFeatureFlag('ROX_POSTGRES_DATASTORE');

        visitVulnerabilityManagementEntities(entitiesKey1);

        // Specify td elements for Image CVEs instead of Node CVEs or Platform CVEs.
        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(`.rt-td:nth-child(3) ${selectors.fixableCvesLink}:eq(0)`).click();
            },
            entitiesKey1,
            usingVMUpdates ? 'image-cves' : 'cves'
        );

        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(selectors.backButton).click();
        }, entitiesKey1);

        cy.get(`${selectors.deploymentTileLink} ${selectors.tileLinkSuperText}`)
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

    it('related entities table header should not say "0 entities" or have "page 0 of 0" if there are rows in the table', () => {
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

        cy.get(selectors.sidePanelTableBodyRows).then((value) => {
            const { length: numRows } = value;
            if (numRows) {
                // TODO positive tests for the numbers are more robust, pardon pun.
                cy.get(selectors.entityRowHeader)
                    .invoke('text')
                    .then((headerText) => {
                        expect(headerText).not.to.equal('0 deployments');
                    });

                cy.get(`${selectors.sidePanel} [data-testid="pagination-header"]`)
                    .invoke('text')
                    .then((paginationText) => {
                        expect(paginationText).not.to.contain('of 0');
                    });
            }
        });
    });

    it('should scope deployment data based on selected policy from table row click', () => {
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
                        `${selectors.emptyFindingsSection}:contains("No deployments have failed across this policy")`
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

    it('should scope deployment data based on selected policy from table count link click', () => {
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

    it('should scope deployment data based on selected policy from entity page tab sublist', () => {
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

        cy.get(selectors.sidePanelExpandButton).click();

        // Entity single page, not side panel.
        cy.get(`${selectors.tableBodyRows} ${selectors.statusChips}:contains('fail')`).should(
            'exist'
        );
        cy.get(`${selectors.tableBodyRows} ${selectors.statusChips}:contains('pass')`).should(
            'not.exist'
        );
    });

    it('should have filtered deployments list in 3rd level of side panel (namespaces -> policies -> deployments)', () => {
        const entitiesKey1 = 'namespaces';
        visitVulnerabilityManagementEntities('namespaces');

        const firstDeploymentCountLinkSelector = `${selectors.deploymentCountLink}:eq(0)`;
        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(firstDeploymentCountLinkSelector).click();
            },
            entitiesKey1,
            'deployments'
        );

        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(selectors.parentEntityInfoHeader).click();
        }, entitiesKey1);

        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(selectors.policyTileLink).click();
            },
            entitiesKey1,
            'policies'
        );

        cy.get(firstDeploymentCountLinkSelector)
            .invoke('text')
            .then((deploymentCountText) => {
                interactAndWaitForVulnerabilityManagementEntity(() => {
                    cy.get(`${selectors.sidePanelTableBodyRows}:eq(0)`).click();
                }, 'policies');

                cy.get(selectors.deploymentTileLink)
                    .invoke('text')
                    .then((relatedDeploymentCountText) => {
                        expect(relatedDeploymentCountText.toLowerCase().trim()).to.equal(
                            deploymentCountText.replace(' ', '')
                        );
                    });

                interactAndWaitForVulnerabilityManagementSecondaryEntities(
                    () => {
                        cy.get(selectors.deploymentTileLink).click();
                    },
                    'policies',
                    'deployments'
                );

                cy.get(selectors.entityRowHeader)
                    .invoke('text')
                    .then((paginationText) => {
                        expect(paginationText).to.equal(deploymentCountText);
                    });
            });
    });

    it('should show a CVE description in overview when coming from cve list', () => {
        const usingVMUpdates = hasFeatureFlag('ROX_POSTGRES_DATASTORE');
        const entitiesKey = usingVMUpdates ? 'image-cves' : 'cves';
        visitVulnerabilityManagementEntities(entitiesKey);

        cy.get(`${selectors.tableBodyRowGroups}:eq(0) ${selectors.cveDescription}`)
            .invoke('text')
            .then((descriptionInList) => {
                interactAndWaitForVulnerabilityManagementEntity(() => {
                    cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
                }, entitiesKey);

                cy.get(`${selectors.entityOverview} ${selectors.metadataDescription}`)
                    .invoke('text')
                    .then((descriptionInSidePanel) => {
                        expect(descriptionInSidePanel).to.equal(descriptionInList);
                    });
            });
    });

    it('should not filter cluster entity page regardless of entity context', () => {
        const entitiesKey = 'namespaces';
        visitVulnerabilityManagementEntities(entitiesKey);

        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(`${selectors.tableRows}:contains("No deployments"):eq(0)`).click();
        }, entitiesKey);

        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(`${selectors.metadataClusterValue} a`).click();
        }, 'clusters');

        cy.get(`${selectors.sidePanel} ${selectors.tableRows}`).should('exist');
        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains("No deployments")`).should(
            'not.exist'
        );
    });

    it('should show the active state in Component overview when scoped under a deployment', () => {
        const usingVMUpdates = hasFeatureFlag('ROX_POSTGRES_DATASTORE');
        const entitiesKey1 = 'deployments';
        const entitiesKey2 = usingVMUpdates ? 'image-components' : 'components';
        visitVulnerabilityManagementEntities(entitiesKey1);

        // click on the first deployment in the list
        interactAndWaitForVulnerabilityManagementEntity(() => {
            cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
        }, entitiesKey1);

        // now, go to the components for that deployment
        interactAndWaitForVulnerabilityManagementSecondaryEntities(
            () => {
                cy.get(
                    usingVMUpdates ? selectors.imageComponentTileLink : selectors.componentTileLink
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

        cy.get(`[data-testid="Active status-value"]`)
            .invoke('text')
            .then((activeStatusText) => {
                expect(activeStatusText).to.be.oneOf(['Active', 'Inactive', 'Undetermined']);
            });
    });

    it('should show the active state in the fixable CVES widget for a single deployment', () => {
        const entitiesKey = 'deployments';

        visitVulnerabilityManagementEntities(entitiesKey);

        interactAndWaitForVulnerabilityManagementEntity(() => {
            // TODO Replace .eq(1) method with :eq(0) pseudo-selector?
            // TODO Index 1 instead of 0 because row selector not limited to table body?
            cy.get(`${selectors.tableRows}`).eq(1).click();
        }, entitiesKey);

        const opname = 'getFixableCvesForEntity';
        const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);
        const usingVMUpdates = hasFeatureFlag('ROX_POSTGRES_DATASTORE');
        const fixableCvesFixture = usingVMUpdates
            ? 'vulnerabilities/fixableCvesForEntity.json'
            : 'vulnerabilities/fixableCvesForEntityLegacy.json';
        const staticResponseMap = {
            [opname]: {
                fixture: fixableCvesFixture,
            },
        };
        interactAndWaitForResponses(
            () => {
                cy.get('button:contains("Fixable CVEs")').click();
            },
            routeMatcherMap,
            staticResponseMap
        );

        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains("CVE-2021-20231")`).contains(
            'Active'
        );
        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains("CVE-2021-20232")`).contains(
            'Inactive'
        );
    });
});
