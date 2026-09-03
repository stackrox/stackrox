import { interceptRequests } from '../../../helpers/request';
import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { assertCannotFindThePage } from '../../../helpers/visit';

import {
    getVirtualMachineAlias,
    getVirtualMachineCveComponentsAlias,
    listVirtualMachineComponentsAlias,
    listVirtualMachineCvesAlias,
    routeMatcherMapForVirtualMachineComponents,
    routeMatcherMapForVirtualMachineCveComponents,
    routeMatcherMapForVirtualMachineVulnerabilities,
    visitVirtualMachinePage,
    visitVirtualMachinePageWithStaticPermissions,
} from './VirtualMachineCve.helpers';

const vmId = 'vm-001';
const fixturePathGetVM = 'vulnerabilities/virtualMachineCves/getVM';
const fixturePathListCves = 'vulnerabilities/virtualMachineCves/listVirtualMachineCves';
const fixturePathListComponents = 'vulnerabilities/virtualMachineCves/listVirtualMachineComponents';
const fixturePathCveComponents = 'vulnerabilities/virtualMachineCves/getVMCVEComponents';

const staticResponseMapForVirtualMachineVulnerabilities = {
    [getVirtualMachineAlias]: { fixture: fixturePathGetVM },
    [listVirtualMachineCvesAlias]: { fixture: fixturePathListCves },
};

function visitWithVulnFixtures() {
    visitVirtualMachinePage(
        vmId,
        routeMatcherMapForVirtualMachineVulnerabilities,
        staticResponseMapForVirtualMachineVulnerabilities
    );
}

function visitComponentsTab() {
    visitWithVulnFixtures();
    interceptRequests(routeMatcherMapForVirtualMachineComponents, {
        [listVirtualMachineComponentsAlias]: { fixture: fixturePathListComponents },
    });
    cy.get('button').contains('Components').click();
    cy.wait(`@${listVirtualMachineComponentsAlias}`);
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
            routeMatcherMapForVirtualMachineVulnerabilities,
            staticResponseMapForVirtualMachineVulnerabilities
        );
        cy.get('h1').contains('cypress-vm-1');
    });

    describe('Vulnerabilities tab', () => {
        it('should render CVE rows from fixture data', () => {
            visitWithVulnFixtures();

            cy.get('tbody tr:not([class*="expandable"])').should('have.length', 3);

            cy.get('tbody tr:not([class*="expandable"])')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="CVE"]').contains('CVE-2024-0001');
                    cy.get('td[data-label="Top CVE severity"]').contains('Critical');
                    cy.get('td[data-label="CVE status"]').contains('Fixable');
                    cy.get('td[data-label="Top CVSS"]').contains('9.8');
                    cy.get('td[data-label="EPSS probability"]').contains('85.000%');
                    cy.get('td[data-label="Affected components"]').contains('1 component');
                });
        });

        it('should display the correct number of affected components', () => {
            visitWithVulnFixtures();

            cy.get('tbody tr:not([class*="expandable"])')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="CVE"]').contains('CVE-2024-0001');
                    cy.get('td[data-label="Affected components"]').contains('1 component');
                });

            cy.get('tbody tr:not([class*="expandable"])')
                .eq(2)
                .within(() => {
                    cy.get('td[data-label="CVE"]').contains('CVE-2024-0003');
                    cy.get('td[data-label="Affected components"]').contains('1 component');
                });
        });

        it('should expand a row to show affected component details', () => {
            visitWithVulnFixtures();

            // Components for a single CVE are fetched lazily on row expand, so stub that
            // endpoint after the initial page load rather than in the visit route map.
            interceptRequests(routeMatcherMapForVirtualMachineCveComponents, {
                [getVirtualMachineCveComponentsAlias]: { fixture: fixturePathCveComponents },
            });

            cy.get('tbody tr:not([class*="expandable"])').eq(0).find('td button').first().click();
            cy.wait(`@${getVirtualMachineCveComponentsAlias}`);

            cy.get('tbody tr[class*="expandable"]')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="Component"]').contains('openssl');
                    cy.get('td[data-label="Version"]').contains('3.0.7-20.el9');
                });
        });

        it('should display an empty state when the VM has no vulnerabilities', () => {
            visitVirtualMachinePage(vmId, routeMatcherMapForVirtualMachineVulnerabilities, {
                [getVirtualMachineAlias]: {
                    body: {
                        id: vmId,
                        namespace: 'default',
                        name: 'empty-vm',
                        clusterId: 'cluster-001',
                        clusterName: 'production-cluster',
                        guestOs: 'Red Hat Enterprise Linux 9.2',
                        lastUpdated: '2025-04-15T10:30:00.000Z',
                        facts: {},
                        annotations: {},
                        labels: {},
                        vsockCid: 3,
                        notes: [],
                        state: 'VM_STATE_RUNNING',
                        agentStatus: 'AGENT_STATUS_ACTIVE',
                    },
                },
                [listVirtualMachineCvesAlias]: {
                    body: { cves: [], totalCount: 0 },
                },
            });

            cy.get('body').contains('No CVEs were detected for this virtual machine');
        });
    });

    describe('Components tab', () => {
        it('should render component rows from fixture data', () => {
            visitComponentsTab();

            cy.get('tbody tr').should('have.length', 3);

            cy.get('tbody tr')
                .eq(0)
                .within(() => {
                    cy.get('td[data-label="Name"]').contains('openssl');
                    cy.get('td[data-label="Version"]').contains('3.0.7-20.el9');
                    cy.get('td[data-label="Status"]').contains('Scanned');
                    cy.get('td[data-label="Source"]').contains('OS');
                });

            cy.get('tbody tr')
                .eq(1)
                .within(() => {
                    cy.get('td[data-label="Name"]').contains('curl');
                    cy.get('td[data-label="Version"]').contains('7.76.1-26.el9');
                });

            cy.get('tbody tr')
                .eq(2)
                .within(() => {
                    cy.get('td[data-label="Name"]').contains('systemd');
                    cy.get('td[data-label="Version"]').contains('252-18.el9');
                    cy.get('td[data-label="Status"]').contains('Not scanned');
                });
        });

        it('should show scanned and unscanned statuses', () => {
            visitComponentsTab();

            cy.get('td[data-label="Status"]').then(($cells) => {
                const statuses = $cells.map((_, el) => el.innerText.trim()).get();
                expect(statuses).to.include('Scanned');
                expect(statuses).to.include('Not scanned');
            });
        });
    });

    describe('Breadcrumb navigation', () => {
        it('should display VM name in breadcrumb', () => {
            visitWithVulnFixtures();

            cy.get('.pf-v6-c-breadcrumb__item').should('contain.text', 'Virtual Machines');
            cy.get('.pf-v6-c-breadcrumb__item').should('contain.text', 'cypress-vm-1');
        });

        it('should link back to the overview page', () => {
            visitWithVulnFixtures();

            cy.get('.pf-v6-c-breadcrumb__item a')
                .contains('Virtual Machines')
                .should('have.attr', 'href')
                .and('include', '/main/vulnerabilities/virtual-machine-cves');
        });
    });
});
