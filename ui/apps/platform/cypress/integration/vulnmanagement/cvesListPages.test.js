import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns, allChecksForEntities } from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('CVEs list Page and its entity detail page, sub list validations ', () => {
    withAuth();

    describe('with VM updates OFF', () => {
        before(function beforeHook() {
            if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        it('should display all the columns and links expected in cves list page', () => {
            visitVulnerabilityManagementEntities('cves');
            hasExpectedHeaderColumns(
                [
                    'CVE',
                    'Type',
                    'Fixable',
                    'CVSS Score',
                    'Env. Impact',
                    'Impact Score',
                    'Entities',
                    'Discovered Time',
                    'Published',
                ],
                1 // skip 1 additional column to account for checkbox column
            );
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                    allChecksForEntities(url.list.cves, 'Deployment');
                }
                if (columnValue !== 'no images' && columnValue.includes('image')) {
                    allChecksForEntities(url.list.cves, 'image');
                }
                if (columnValue !== 'no components' && columnValue.includes('component')) {
                    allChecksForEntities(url.list.cves, 'component');
                }
            });

            // special check for CVE list only, for description in 2nd line of row
            cy.get(selectors.cveDescription, { timeout: 6000 })
                .eq(0)
                .invoke('text')
                .then((value) => {
                    expect(value).not.to.include('No description available');
                });
        });

        it('should display Discovered in Image time column when appropriate', () => {
            visitVulnerabilityManagementEntities('cves');
            cy.get(`${selectors.tableColumn}`)
                .invoke('text')
                .then((text) => {
                    expect(text).not.to.include('Discovered in Image');
                });

            visitVulnerabilityManagementEntities('images');
            cy.get(`${selectors.allCVEColumnLink}:eq(0)`).click({ force: true });
            cy.get(`[data-testid="side-panel"] ${selectors.tableColumn}`)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('Discovered in Image');
                });

            visitVulnerabilityManagementEntities('components');
            cy.get(`${selectors.allCVEColumnLink}:eq(0)`).click({ force: true });
            cy.get(`[data-testid="side-panel"] ${selectors.tableColumn}`)
                .invoke('text')
                .then((text) => {
                    expect(text).not.to.include('Discovered in Image');
                });
        });

        it('should display correct CVE type', () => {
            visitVulnerabilityManagementEntities('cves');

            cy.get(`${selectors.cveTypes}:first`)
                .invoke('text')
                .then((cveTypeText) => {
                    cy.get(`${selectors.cveTypes}:first`).click({
                        force: true,
                    });

                    cy.get(selectors.cveType)
                        .invoke('text')
                        .then((overviewCveTypeText) => {
                            expect(overviewCveTypeText).to.contain(cveTypeText);
                        });
                });
        });

        it('should suppress CVE', () => {
            visitVulnerabilityManagementEntities('cves');
            cy.get(selectors.cveSuppressPanelButton).should('be.disabled');

            // Obtain the CVE to verify in suppressed view
            cy.get(selectors.tableBodyRows)
                .first()
                .find(`.rt-td`)
                .eq(2)
                .then((value) => {
                    const cve = value.text();

                    cy.get(selectors.tableBodyRows)
                        .first()
                        .get(selectors.tableRowCheckbox)
                        .check({ force: true });
                    cy.get(selectors.cveSuppressPanelButton)
                        .click()
                        .get(selectors.suppressOneDayOption)
                        .click({ force: true });

                    // toggle to suppressed view
                    cy.get(selectors.suppressToggleViewPanelButton).click({ force: true });

                    // Verify that the suppressed CVE shows up in the table
                    cy.get(selectors.tableBodyRows, { timeout: 4500 }).contains(cve);
                });
        });

        it.skip('should unsuppress suppressed CVE', () => {
            visitVulnerabilityManagementEntities('cves', '?s[CVE%20Snoozed]=true');
            cy.get(selectors.cveUnsuppressPanelButton).should('be.disabled');

            // Obtain the CVE to verify in unsuppressed view
            cy.get(selectors.tableBodyRows)
                .first()
                .find(`.rt-td`)
                .eq(2)
                .then((value) => {
                    const cve = value.text();

                    cy.get(selectors.tableBodyRows)
                        .first()
                        .find(selectors.cveUnsuppressRowButton)
                        .click({ force: true });

                    // toggle to unsuppressed view
                    cy.get(selectors.suppressToggleViewPanelButton).click();

                    // Verify that the unsuppressed CVE shows up in the table
                    cy.get(selectors.tableBodyRows, { timeout: 4500 }).contains(cve);
                });
        });
    });

    describe('with VM updates ON', () => {
        before(function beforeHook() {
            if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        // @TODO: This test fails. Reference: https://prow.ci.openshift.org/view/gs/origin-ci-test/pr-logs/pull/stackrox_stackrox/2327/pull-ci-stackrox-stackrox-master-gke-ui-e2e-tests/1546974612569460736
        describe('Image CVE type list', () => {
            it('should display all the columns and links expected in cves list page', () => {
                visitVulnerabilityManagementEntities('image-cves');
                hasExpectedHeaderColumns(
                    [
                        'CVE',
                        'Operating System',
                        'Fixable',
                        'Severity',
                        'CVSS Score',
                        'Env. Impact',
                        'Impact Score',
                        'Entities',
                        'Discovered Time',
                        'Published',
                    ],
                    1 // skip 1 additional column to account for checkbox column
                );

                cy.get(selectors.tableBodyColumn).each(($el) => {
                    const columnValue = $el.text().toLowerCase();
                    if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                        allChecksForEntities(url.list['image-cves'], 'Deployment');
                    }
                    if (
                        columnValue !== 'no image components' &&
                        columnValue.includes('image component')
                    ) {
                        allChecksForEntities(url.list['image-cves'], 'image component');
                    }
                    // TODO: uncomment the components check after component changes are integrated
                    // if (columnValue !== 'no components' && columnValue.includes('component')) {
                    //     allChecksForEntities(url.list['image-cves'], 'component');
                    // }
                });

                // special check for CVE list only, for description in 2nd line of row
                cy.get(selectors.cveDescription, { timeout: 6000 })
                    .eq(0)
                    .invoke('text')
                    .then((value) => {
                        expect(value).not.to.include('No description available');
                    });
            });

            it('should add CVEs to new policies', () => {
                visitVulnerabilityManagementEntities('image-cves');

                cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

                cy.get(`${selectors.tableRowCheckbox}:first`)
                    .wait(100)
                    .get(`${selectors.tableRowCheckbox}:first`)
                    .click();
                cy.get(selectors.cveAddToPolicyButton).click();

                // TODO: finish testing with react-select, that evil component
                // cy.get(selectors.cveAddToPolicyShortForm.select).click().type('cypress-test-policy');
            });
        });

        describe('Node CVE type list', () => {
            it('should display all the columns and links expected in cves list page', () => {
                visitVulnerabilityManagementEntities('node-cves');
                hasExpectedHeaderColumns(
                    [
                        'CVE',
                        'Operating System',
                        'Fixable',
                        'Severity',
                        'CVSS Score',
                        'Env. Impact',
                        'Impact Score',
                        'Entities',
                        'Discovered Time',
                        'Published',
                    ],
                    1 // skip 1 additional column to account for checkbox column
                );
                cy.get(selectors.tableBodyColumn).each(($el) => {
                    const columnValue = $el.text().toLowerCase();
                    if (columnValue !== 'no nodes' && columnValue.includes('node')) {
                        allChecksForEntities(url.list['node-cves'], 'nodes');
                    }
                    // TODO: uncomment the components check after component changes are integrated
                    // if (columnValue !== 'no components' && columnValue.includes('component')) {
                    //     allChecksForEntities(url.list['node-cves'], 'component');
                    // }
                });

                // special check for CVE list only, for description in 2nd line of row
                cy.get(selectors.cveDescription, { timeout: 6000 })
                    .eq(0)
                    .invoke('text')
                    .then((value) => {
                        expect(value).not.to.include('No description available');
                    });
            });
        });

        describe('Cluster (Platform) CVE type list', () => {
            it('should display all the columns and links expected in cves list page', () => {
                visitVulnerabilityManagementEntities('cluster-cves');
                hasExpectedHeaderColumns(
                    [
                        'CVE',
                        'Type',
                        'Fixable',
                        'CVSS Score',
                        'Env. Impact',
                        'Impact Score',
                        'Entities',
                        'Published',
                    ],
                    1 // skip 1 additional column to account for checkbox column
                );
                cy.get(selectors.tableBodyColumn).each(($el) => {
                    const columnValue = $el.text().toLowerCase();
                    if (columnValue !== 'no nodes' && columnValue.includes('node')) {
                        allChecksForEntities(url.list['node-cves'], 'node');
                    }
                    // TODO: uncomment the components check after component changes are integrated
                    // if (columnValue !== 'no components' && columnValue.includes('component')) {
                    //     allChecksForEntities(url.list['node-cves'], 'component');
                    // }
                });

                // special check for CVE list only, for description in 2nd line of row
                cy.get(selectors.cveDescription, { timeout: 6000 })
                    .eq(0)
                    .invoke('text')
                    .then((value) => {
                        expect(value).not.to.include('No description available');
                    });
            });
        });

        // @TODO: Rework this test. Seems like each of these do the same thing
        describe.skip('adding selected CVEs to policy', () => {
            it('should add CVEs to new policies', () => {
                visitVulnerabilityManagementEntities('cves');

                cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

                cy.get(`${selectors.tableRowCheckbox}:first`)
                    .wait(100)
                    .get(`${selectors.tableRowCheckbox}:first`)
                    .click();
                cy.get(selectors.cveAddToPolicyButton).click();

                // TODO: finish testing with react-select, that evil component
                // cy.get(selectors.cveAddToPolicyShortForm.select).click().type('cypress-test-policy');
            });

            it('should add CVEs to existing policies', () => {
                visitVulnerabilityManagementEntities('cves');

                cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

                cy.get(`${selectors.tableRowCheckbox}:first`)
                    .wait(100)
                    .get(`${selectors.tableRowCheckbox}:first`)
                    .click();
                cy.get(selectors.cveAddToPolicyButton).click();

                // TODO: finish testing with react-select, that evil component
                // cy.get(selectors.cveAddToPolicyShortForm.select).click();
                // cy.get(selectors.cveAddToPolicyShortForm.selectValue).eq(1).click();
            });

            it('should add CVEs to existing policies with CVEs', () => {
                visitVulnerabilityManagementEntities('cves');

                cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

                cy.get(`${selectors.tableRowCheckbox}:first`)
                    .wait(100)
                    .get(`${selectors.tableRowCheckbox}:first`)
                    .click();
                cy.get(selectors.cveAddToPolicyButton).click();

                // TODO: finish testing with react-select, that evil component
                // cy.get(selectors.cveAddToPolicyShortForm.select).click();
                // cy.get(selectors.cveAddToPolicyShortForm.selectValue).first().click();
            });
        });
    });

    // TODO to be fixed after back end sorting is fixed
    // validateSortForCVE(selectors.cvesCvssScoreCol);
});
