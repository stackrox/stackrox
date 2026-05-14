import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { assertCannotFindThePage } from '../../../helpers/visit';

import {
    getVirtualMachineAlias,
    routeMatcherMapForVirtualMachine,
    visitVirtualMachinePage,
    visitVirtualMachinePageWithStaticPermissions,
} from './VirtualMachineCve.helpers';

const vmId = 'vm-001';
const fixturePathGetVM = 'vulnerabilities/virtualMachineCves/getVirtualMachine';

function visitWithFixture() {
    visitVirtualMachinePage(vmId, routeMatcherMapForVirtualMachine, {
        [getVirtualMachineAlias]: { fixture: fixturePathGetVM },
    });
}

describe('Virtual Machine CVEs - Virtual Machine Page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VIRTUAL_MACHINES')) {
            this.skip();
        }
    });

    it('should restrict access to users without "Cluster" permission', () => {
        visitVirtualMachinePageWithStaticPermissions(vmId, {});
        assertCannotFindThePage();
    });

    it('should allow access to users with "Cluster" permission', () => {
        visitVirtualMachinePageWithStaticPermissions(
            vmId,
            { Cluster: 'READ_ACCESS' },
            routeMatcherMapForVirtualMachine,
            {
                [getVirtualMachineAlias]: { fixture: fixturePathGetVM },
            }
        );
        cy.get('h1').contains('cypress-vm-1');
    });

    describe('Vulnerabilities tab', () => {
        it('should render CVE rows from fixture data', () => {
            visitWithFixture();

            cy.get('tbody tr:not([class*="expandable"])').should('have.length', 3);

            cy.get('tbody tr:not([class*="expandable"])')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="CVE"]').contains('CVE-2024-0001');
                    cy.get('td[data-label="CVE severity"]').should('exist');
                    cy.get('td[data-label="CVE status"]').should('exist');
                    cy.get('td[data-label="CVSS"]').should('exist');
                    cy.get('td[data-label="EPSS probability"]').should('exist');
                    cy.get('td[data-label="Affected components"]').contains('openssl');
                });
        });

        it('should display the correct number of affected components', () => {
            visitWithFixture();

            cy.get('tbody tr:not([class*="expandable"])')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="Affected components"]').contains('openssl');
                });

            cy.get('tbody tr:not([class*="expandable"])')
                .eq(2)
                .within(() => {
                    cy.get('td[data-label="Affected components"]').contains('curl');
                });
        });

        it('should expand a row to show the components sub-table', () => {
            visitWithFixture();

            cy.get('tbody tr:not([class*="expandable"])').eq(0).find('td button').first().click();

            cy.get('tbody tr[class*="expandable"]')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="Component"]').contains('openssl');
                    cy.get('td[data-label="Version"]').contains('3.0.7-20.el9');
                    cy.get('td[data-label="CVE fixed in"]').contains('3.0.7-25.el9');
                    cy.get('td[data-label="Advisory"]').contains('RHSA-2024:0001');
                });
        });

        it('should display an empty state when the VM has no vulnerabilities', () => {
            visitVirtualMachinePage(vmId, routeMatcherMapForVirtualMachine, {
                [getVirtualMachineAlias]: {
                    body: {
                        id: vmId,
                        namespace: 'default',
                        name: 'empty-vm',
                        clusterId: 'cluster-001',
                        clusterName: 'production-cluster',
                        scan: {
                            scanTime: '2025-04-15T10:30:00.000Z',
                            operatingSystem: 'rhel:9',
                            components: [],
                            notes: [],
                        },
                        lastUpdated: '2025-04-15T10:30:00.000Z',
                        vsockCid: 3,
                        state: 'RUNNING',
                    },
                },
            });

            cy.get('body').contains('No CVEs were detected for this virtual machine');
        });
    });

    describe('Components tab', () => {
        it('should render component rows from fixture data', () => {
            visitWithFixture();

            cy.get('button').contains('Components').click();

            cy.get('tbody tr').should('have.length', 3);

            cy.get('tbody tr')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="Name"]').should('exist');
                    cy.get('td[data-label="Version"]').should('exist');
                    cy.get('td[data-label="Status"]').should('exist');
                });
        });

        it('should show scanned and unscanned statuses', () => {
            visitWithFixture();

            cy.get('button').contains('Components').click();

            cy.get('td[data-label="Status"]').then(($cells) => {
                const statuses = $cells.map((_, el) => el.innerText.trim()).get();
                expect(statuses).to.include('Scanned');
                expect(statuses).to.include('Not scanned');
            });
        });
    });

    describe('Breadcrumb navigation', () => {
        it('should display VM name in breadcrumb', () => {
            visitWithFixture();

            cy.get('.pf-v6-c-breadcrumb__item').should('contain.text', 'Virtual Machines');
            cy.get('.pf-v6-c-breadcrumb__item').should('contain.text', 'cypress-vm-1');
        });

        it('should link back to the overview page', () => {
            visitWithFixture();

            cy.get('.pf-v6-c-breadcrumb__item a')
                .contains('Virtual Machines')
                .should('have.attr', 'href')
                .and('include', '/main/vulnerabilities/virtual-machine-cves');
        });
    });
});
