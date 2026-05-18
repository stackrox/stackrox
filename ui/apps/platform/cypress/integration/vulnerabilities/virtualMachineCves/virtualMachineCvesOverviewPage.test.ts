import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { assertCannotFindThePage } from '../../../helpers/visit';
import { interceptAndWatchRequests } from '../../../helpers/request';
import { paginateNext, paginatePrevious, sortByTableHeader } from '../../../helpers/tableHelpers';

import {
    listVirtualMachinesAlias,
    routeMatcherMapForVirtualMachines,
    visitVirtualMachineCvesOverviewPage,
    visitVirtualMachineCvesOverviewPageWithStaticPermissions,
} from './VirtualMachineCve.helpers';

const fixturePathListVMs = 'vulnerabilities/virtualMachineCves/listVirtualMachines';

describe('Virtual Machine CVEs - Overview Page', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VIRTUAL_MACHINES')) {
            this.skip();
        }
    });

    it('should restrict access to users without "Cluster" permission', () => {
        visitVirtualMachineCvesOverviewPageWithStaticPermissions({});
        assertCannotFindThePage();
    });

    it('should allow access to users with "Cluster" permission', () => {
        visitVirtualMachineCvesOverviewPageWithStaticPermissions(
            { Cluster: 'READ_ACCESS' },
            routeMatcherMapForVirtualMachines,
            {
                [listVirtualMachinesAlias]: {
                    fixture: fixturePathListVMs,
                },
            }
        );
        cy.get('h1').contains('Virtual machine vulnerabilities');
    });

    it('should render the overview page heading and description', () => {
        visitVirtualMachineCvesOverviewPage(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: {
                fixture: fixturePathListVMs,
            },
        });

        cy.get('h1').contains('Virtual machine vulnerabilities');
        cy.get('body').contains('Prioritize and remediate observed CVEs across virtual machines');
    });

    it('should render VM rows from fixture data', () => {
        visitVirtualMachineCvesOverviewPage(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: {
                fixture: fixturePathListVMs,
            },
        });

        cy.get('tbody tr').should('have.length', 3);

        cy.get('tbody tr')
            .eq(0)
            .within(() => {
                cy.get('td[data-label="Virtual machine"]').contains('cypress-vm-1');
                cy.get('td[data-label="Cluster"]').contains('production-cluster');
                cy.get('td[data-label="Namespace"]').contains('default');
            });

        cy.get('tbody tr')
            .eq(1)
            .within(() => {
                cy.get('td[data-label="Virtual machine"]').contains('cypress-vm-2');
                cy.get('td[data-label="Namespace"]').contains('monitoring');
            });

        cy.get('tbody tr')
            .eq(2)
            .within(() => {
                cy.get('td[data-label="Virtual machine"]').contains('cypress-vm-3');
                cy.get('td[data-label="Cluster"]').contains('staging-cluster');
            });
    });

    it('should link VM names to the correct detail page', () => {
        visitVirtualMachineCvesOverviewPage(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: {
                fixture: fixturePathListVMs,
            },
        });

        cy.get('tbody tr td[data-label="Virtual machine"] a')
            .first()
            .then(($link) => {
                const href = $link.attr('href');
                expect(href).to.match(
                    /\/main\/vulnerabilities\/virtual-machine-cves\/virtualmachines\/vm-001$/
                );
            });
    });

    it('should display an empty state when no VMs are returned', () => {
        visitVirtualMachineCvesOverviewPage(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: {
                body: { virtualMachines: [], totalCount: 0 },
            },
        });

        cy.get('body').contains('No CVEs have been detected');
    });

    it('should sort by the Virtual machine column', () => {
        interceptAndWatchRequests(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: {
                fixture: fixturePathListVMs,
            },
        }).then(({ waitForRequests }) => {
            visitVirtualMachineCvesOverviewPage();
            waitForRequests();

            sortByTableHeader('Virtual machine');
            cy.wait(`@${listVirtualMachinesAlias}`).then((interception) => {
                const { url } = interception.request;
                expect(url).to.include('Virtual Machine Name');
            });
        });
    });

    it('should paginate through results', () => {
        const paginatedFixture = {
            body: {
                virtualMachines: Array.from({ length: 20 }, (_, i) => ({
                    id: `vm-${String(i + 1).padStart(3, '0')}`,
                    namespace: 'default',
                    name: `cypress-vm-${i + 1}`,
                    clusterId: 'cluster-001',
                    clusterName: 'production-cluster',
                    lastUpdated: '2025-04-15T10:30:00.000Z',
                    vsockCid: i + 3,
                    state: 'RUNNING',
                })),
                totalCount: 50,
            },
        };

        interceptAndWatchRequests(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: paginatedFixture,
        }).then(({ waitForRequests }) => {
            visitVirtualMachineCvesOverviewPage();
            waitForRequests();

            paginateNext();
            cy.wait(`@${listVirtualMachinesAlias}`).then((interception) => {
                const { url } = interception.request;
                expect(url).to.include('query.pagination.offset=20');
            });

            paginatePrevious();
            cy.wait(`@${listVirtualMachinesAlias}`).then((interception) => {
                const { url } = interception.request;
                expect(url).to.include('query.pagination.offset=0');
            });
        });
    });
});
