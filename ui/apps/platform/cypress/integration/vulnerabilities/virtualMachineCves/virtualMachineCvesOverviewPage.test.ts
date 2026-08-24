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

const fixturePathListVMs = 'vulnerabilities/virtualMachineCves/listVMs';
const virtualMachineTabParams = { entityTab: 'VirtualMachine' };

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
            },
            virtualMachineTabParams
        );
        cy.get('h1').contains('Virtual machine vulnerabilities');
    });

    it('should render the overview page heading and description', () => {
        visitVirtualMachineCvesOverviewPage(
            routeMatcherMapForVirtualMachines,
            {
                [listVirtualMachinesAlias]: {
                    fixture: fixturePathListVMs,
                },
            },
            virtualMachineTabParams
        );

        cy.get('h1').contains('Virtual machine vulnerabilities');
        cy.get('body').contains('Prioritize and remediate observed CVEs across virtual machines');
    });

    it('should render VM rows from fixture data', () => {
        visitVirtualMachineCvesOverviewPage(
            routeMatcherMapForVirtualMachines,
            {
                [listVirtualMachinesAlias]: {
                    fixture: fixturePathListVMs,
                },
            },
            virtualMachineTabParams
        );

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
        visitVirtualMachineCvesOverviewPage(
            routeMatcherMapForVirtualMachines,
            {
                [listVirtualMachinesAlias]: {
                    fixture: fixturePathListVMs,
                },
            },
            virtualMachineTabParams
        );

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
        visitVirtualMachineCvesOverviewPage(
            routeMatcherMapForVirtualMachines,
            {
                [listVirtualMachinesAlias]: {
                    body: { vms: [], totalCount: 0 },
                },
            },
            virtualMachineTabParams
        );

        cy.get('body').contains('No CVEs have been detected');
    });

    it('should sort by the Virtual machine column', () => {
        interceptAndWatchRequests(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: {
                fixture: fixturePathListVMs,
            },
        }).then(({ waitForRequests }) => {
            visitVirtualMachineCvesOverviewPage(
                undefined,
                undefined,
                virtualMachineTabParams
            );
            waitForRequests();

            sortByTableHeader('Virtual machine');
            cy.wait(`@${listVirtualMachinesAlias}`).then((interception) => {
                const { url } = interception.request;
                expect(decodeURIComponent(url)).to.include('Virtual Machine Name');
            });
        });
    });

    it('should paginate through results', () => {
        const paginatedFixture = {
            body: {
                vms: Array.from({ length: 20 }, (_, i) => ({
                    id: `vm-${String(i + 1).padStart(3, '0')}`,
                    namespace: 'default',
                    name: `cypress-vm-${i + 1}`,
                    clusterId: 'cluster-001',
                    clusterName: 'production-cluster',
                    guestOs: 'Red Hat Enterprise Linux 9.2',
                    lastUpdated: '2025-04-15T10:30:00.000Z',
                    scanTime: '2025-04-15T10:30:00.000Z',
                    vsockCid: i + 3,
                    state: 'VM_STATE_RUNNING',
                    cveSeverityCounts: {
                        critical: { total: 0, fixable: 0 },
                        important: { total: 0, fixable: 0 },
                        moderate: { total: 0, fixable: 0 },
                        low: { total: 0, fixable: 0 },
                        unknown: { total: 0, fixable: 0 },
                    },
                    componentScanCount: { scanned: 0, total: 0 },
                })),
                totalCount: 50,
            },
        };

        interceptAndWatchRequests(routeMatcherMapForVirtualMachines, {
            [listVirtualMachinesAlias]: paginatedFixture,
        }).then(({ waitForRequests }) => {
            visitVirtualMachineCvesOverviewPage(
                undefined,
                undefined,
                virtualMachineTabParams
            );
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
