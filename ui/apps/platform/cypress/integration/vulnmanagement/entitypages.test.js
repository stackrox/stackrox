import * as api from '../../constants/apiEndpoints';
import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';

describe('Entities single views', () => {
    withAuth();

    it('related entities tile links should unset search params upon navigation', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('CLUSTER')).as('getClusters');
        cy.visit(url.list.clusters);
        cy.wait('@getClusters');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('CLUSTER', 'CVE')).as('getClusterCVE');
        cy.get(`${selectors.tableBodyRows} ${selectors.fixableCvesLink}:eq(0)`).click();
        cy.wait('@getClusterCVE');

        cy.intercept('POST', api.vulnMgmt.graphqlEntity('CLUSTER')).as('getCluster');
        cy.get(selectors.backButton).click();
        cy.wait('@getCluster');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('CLUSTER', 'DEPLOYMENT')).as(
            'getClusterDEPLOYMENT'
        );
        cy.get(`${selectors.deploymentTileLink} ${selectors.tileLinkSuperText}`)
            .invoke('text')
            .then((numDeployments) => {
                cy.get(selectors.deploymentTileLink).click();
                cy.wait('@getClusterDEPLOYMENT');

                cy.get(`[data-testid="side-panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(
                            parseInt(numDeployments, 10)
                        );
                    });
            });
    });

    it('related entities table header should not say "0 entities" or have "page 0 of 0" if there are rows in the table', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('POLICY')).as('getPolicies');
        cy.visit(url.list.policies);
        cy.wait('@getPolicies');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('POLICY', 'DEPLOYMENT')).as(
            'getPolicyDEPLOYMENT'
        );
        cy.get(`${selectors.tableBodyRows} ${selectors.failingDeploymentCountLink}:eq(0)`).click();
        cy.wait('@getPolicyDEPLOYMENT');

        cy.get(selectors.sidePanelTableBodyRows).then((value) => {
            const { length: numRows } = value;
            if (numRows) {
                // TODO positive tests for the numbers are more robust, pardon pun.
                cy.get(selectors.entityRowHeader)
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

    it('should scope deployment data based on selected policy from table row click', () => {
        // policy -> related deployments list should scope policy status column by the policy x deployment row
        // in both side panel and entity page
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('POLICY')).as('getPolicies');
        cy.visit(url.list.policies);
        cy.wait('@getPolicies');

        cy.intercept('POST', api.vulnMgmt.graphqlEntity('POLICY')).as('getPolicy');
        cy.get(`${selectors.tableBodyRows}:eq(0) ${selectors.statusChips}`)
            .invoke('text')
            .then((firstPolicyStatus) => {
                cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
                cy.wait('@getPolicy');

                cy.get(`${selectors.sidePanel} ${selectors.statusChips}:eq(0)`)
                    .invoke('text')
                    .then((selectedPolicyStatus) => {
                        expect(firstPolicyStatus).to.equal(selectedPolicyStatus);
                    });

                if (firstPolicyStatus === 'pass') {
                    cy.get(
                        `${selectors.emptyFindingsSection}:contains("No deployments have failed across this policy")`
                    );

                    cy.intercept('POST', api.vulnMgmt.graphqlEntities2('POLICY', 'DEPLOYMENT')).as(
                        'getPolicyDEPLOYMENT'
                    );
                    cy.get(`${selectors.deploymentTileLink}:eq(0)`).click();
                    cy.wait('@getPolicyDEPLOYMENT');

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
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('POLICY')).as('getPolicies');
        cy.visit(url.list.policies);
        cy.wait('@getPolicies');

        // Assume at least one policy has failing deployments.
        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('POLICY', 'DEPLOYMENT')).as(
            'getPolicyDEPLOYMENT'
        );
        cy.get(`${selectors.failingDeploymentCountLink}:eq(0)`).click();
        cy.wait('@getPolicyDEPLOYMENT');

        cy.get(`${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`).should('exist');
        cy.get(`${selectors.sidePanel} ${selectors.statusChips}:contains('pass')`).should(
            'not.exist'
        );
    });

    // TODO: track fixing this test under this bug ticket, https://stack-rox.atlassian.net/browse/ROX-8705
    it.skip('should scope deployment data based on selected policy from entity page tab sublist', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('POLICY')).as('getPolicies');
        cy.visit(url.list.policies);
        cy.wait('@getPolicies');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('POLICY', 'DEPLOYMENT')).as(
            'getPolicyDEPLOYMENT'
        );
        cy.get(`${selectors.failingDeploymentCountLink}:eq(0)`).click();
        cy.wait('@getPolicyDEPLOYMENT');

        cy.get(selectors.sidePanelExpandButton).click();
        cy.wait('@getPolicyDEPLOYMENT');

        // Entity single page, not side panel.
        cy.get(`${selectors.tableBodyRows} ${selectors.statusChips}:contains('fail')`).should(
            'exist'
        );
        cy.get(`${selectors.tableBodyRows} ${selectors.statusChips}:contains('pass')`).should(
            'not.exist'
        );
    });

    // test skipped because we are not currently showing the Policy (count) column, until and if performance can be improved
    it.skip('should have consistent policy count number from namespace list to policy sublist for a specific namespace', () => {
        cy.visit(url.list.namespaces);

        cy.get(selectors.policyCountLink)
            .eq(2)
            .invoke('text')
            .then((policyCountText) => {
                cy.get(selectors.tableBodyRows).eq(2).click();
                cy.get(selectors.policyTileLink, { timeout: 1000 })
                    .invoke('text')
                    .then((relatedPolicyCountText) => {
                        expect(relatedPolicyCountText.toLowerCase().trim()).to.equal(
                            policyCountText.replace(' ', '')
                        );
                    });
                cy.get(selectors.policyTileLink).click({ force: true });
                cy.get(selectors.entityRowHeader, { timeout: 1000 })
                    .invoke('text')
                    .then((paginationText) => {
                        expect(paginationText).to.equal(policyCountText);
                    });
            });
    });

    it('should have filtered deployments list in 3rd level of side panel (namespaces -> policies -> deployments)', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('NAMESPACE')).as('getNamespaces');
        cy.visit(url.list.namespaces);
        cy.wait('@getNamespaces');

        cy.get(`${selectors.deploymentCountLink}:eq(0)`).as('firstDeploymentCountLink');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('NAMESPACE', 'DEPLOYMENT')).as(
            'getNamespaceDEPLOYMENT'
        );
        cy.get('@firstDeploymentCountLink').click();
        cy.wait('@getNamespaceDEPLOYMENT');

        cy.intercept('POST', api.vulnMgmt.graphqlEntity('NAMESPACE')).as('getNamespace');
        cy.get(selectors.parentEntityInfoHeader).click();
        cy.wait('@getNamespace');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('NAMESPACE', 'POLICY')).as(
            'getNamespacePOLICY'
        );
        cy.get(selectors.policyTileLink).click();
        cy.wait('@getNamespacePOLICY');

        cy.get('@firstDeploymentCountLink')
            .invoke('text')
            .then((deploymentCountText) => {
                cy.intercept('POST', api.vulnMgmt.graphqlEntity('POLICY')).as('getPolicy');
                cy.get(`${selectors.sidePanelTableBodyRows}:eq(0)`).click();
                cy.wait('@getPolicy');

                cy.get(selectors.deploymentTileLink)
                    .invoke('text')
                    .then((relatedDeploymentCountText) => {
                        expect(relatedDeploymentCountText.toLowerCase().trim()).to.equal(
                            deploymentCountText.replace(' ', '')
                        );
                    });
                cy.intercept('POST', api.vulnMgmt.graphqlEntities2('POLICY', 'DEPLOYMENT')).as(
                    'getPolicyDEPLOYMENT'
                );
                cy.get(selectors.deploymentTileLink).click();
                cy.wait('@getPolicyDEPLOYMENT');

                cy.get(selectors.entityRowHeader)
                    .invoke('text')
                    .then((paginationText) => {
                        expect(paginationText).to.equal(deploymentCountText);
                    });
            });
    });

    // @TODO, test needs to be re-structured
    it.skip('should filter deployment count in failing policies section in namespace findings by namespace', () => {
        cy.visit(url.list.namespaces);

        cy.get(`${selectors.deploymentCountLink}:eq(0)`).as('firstDeploymentCountLink');

        // in side panel
        cy.get('@firstDeploymentCountLink')
            .invoke('text')
            .then((listDeploymentCountText) => {
                cy.get('@firstDeploymentCountLink').click({ force: true });

                cy.get(selectors.parentEntityInfoHeader, { timeout: 5000 }).click({ force: true });

                cy.get(selectors.deploymentCountText, { timeout: 16000 })
                    .eq(0)
                    .invoke('text')
                    .then((sidePanelDeploymentCountText) => {
                        expect(listDeploymentCountText).to.equal(sidePanelDeploymentCountText);

                        // in entity page
                        cy.get(selectors.sidePanelExpandButton).click({ force: true });
                        cy.get(selectors.deploymentCountText, { timeout: 16000 })
                            .eq(0)
                            .invoke('text')
                            .then((entityDeploymentCountText) => {
                                expect(sidePanelDeploymentCountText).to.equal(
                                    entityDeploymentCountText
                                );
                            });
                    });
            });
    });

    // TODO: track fixing this test under this bug ticket, https://stack-rox.atlassian.net/browse/ROX-8705
    it.skip('should filter component count in images list and image overview by cve when coming from cve list', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('CVE')).as('getCves');
        cy.visit(url.list.cves);
        cy.wait('@getCves');

        cy.intercept('POST', api.vulnMgmt.graphqlEntities2('CVE', 'IMAGE')).as('getCveIMAGE');
        cy.get(`${selectors.imageCountLink}:eq(0)`).click();
        cy.wait('@getCveIMAGE');

        cy.intercept('POST', api.vulnMgmt.graphqlEntity('CVE')).as('getCve');
        cy.get(selectors.parentEntityInfoHeader).click();
        cy.wait('@getCve');

        cy.get(selectors.imageTileLink).click();
        cy.wait('@getCveIMAGE');

        cy.get(`${selectors.sidePanel} ${selectors.componentCountLink}:eq(0)`)
            .invoke('text')
            .then((componentCountText) => {
                cy.intercept('POST', api.vulnMgmt.graphqlEntity('IMAGE')).as('getImage');
                cy.get(`${selectors.sidePanelTableBodyRows}:eq(0)`).click();
                cy.wait('@getImage');

                cy.get(selectors.componentTileLink)
                    .invoke('text')
                    .then((relatedComponentCountText) => {
                        expect(relatedComponentCountText.toLowerCase().trim()).to.equal(
                            componentCountText.replace(' ', '')
                        );
                    });
            });
    });

    it('should show a CVE description in overview when coming from cve list', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('CVE')).as('getCves');
        cy.visit(url.list.cves);
        cy.wait('@getCves');

        cy.get(`${selectors.tableBodyRowGroups}:eq(0) ${selectors.cveDescription}`)
            .invoke('text')
            .then((descriptionInList) => {
                cy.intercept('POST', api.vulnMgmt.graphqlEntity('CVE')).as('getCve');
                cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
                cy.wait('@getCve');

                cy.get(`${selectors.entityOverview} ${selectors.cveDescription}`)
                    .invoke('text')
                    .then((descriptionInSidePanel) => {
                        expect(descriptionInSidePanel).to.equal(descriptionInList);
                    });
            });
    });

    it('should not filter cluster entity page regardless of entity context', () => {
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('NAMESPACE')).as('getNamespaces');
        cy.visit(url.list.namespaces);
        cy.wait('@getNamespaces');

        cy.intercept('POST', api.vulnMgmt.graphqlEntity('NAMESPACE')).as('getNamespace');
        cy.get(`${selectors.tableRows}:contains("No deployments"):eq(0)`).click();
        cy.wait('@getNamespace');

        cy.intercept('POST', api.vulnMgmt.graphqlEntity('CLUSTER')).as('getCluster');
        cy.get(`${selectors.metadataClusterValue} a`).click();
        cy.wait('@getCluster');

        cy.get(`${selectors.sidePanel} ${selectors.tableRows}`).should('exist');
        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains("No deployments")`).should(
            'not.exist'
        );
    });

    it('should show the active state in Component overview when scoped under a deployment', () => {
        const getDeploymentCOMPONENT = api.graphql(api.vulnMgmt.graphqlOps.getDeploymentCOMPONENT);
        cy.intercept('POST', getDeploymentCOMPONENT).as('getDeploymentCOMPONENT');

        cy.visit(url.list.deployments);

        // click on the first deployment in the list
        cy.get(`${selectors.tableRows}`, { timeout: 10000 }).eq(1).click();
        // now, go the components for that deployment
        cy.get(selectors.componentTileLink).click();
        // click on the first component in that list
        cy.get(`[data-testid="side-panel"] ${selectors.tableRows}`, { timeout: 10000 })
            .eq(1)
            .click();

        cy.wait('@getDeploymentCOMPONENT');

        cy.get(`[data-testid="Active status-value"]`)
            .invoke('text')
            .then((activeStatusText) => {
                expect(activeStatusText).to.be.oneOf(['Active', 'Inactive', 'Undetermined']);
            });
    });

    // TODO: when active status for CVEs becomes available
    // unskip the following test
    it.skip('should show the active state in the fixable CVES widget for a single deployment', () => {
        const getFixableCvesForEntity = api.graphql(
            api.vulnMgmt.graphqlOps.getFixableCvesForEntity
        );
        cy.intercept('POST', getFixableCvesForEntity, {
            fixture: 'vulnerabilities/fixableCvesForEntity.json',
        }).as('getFixableCvesForEntity');

        cy.visit(url.list.deployments);

        cy.get(`${selectors.tableRows}`, { timeout: 10000 }).eq(1).click();
        cy.get('button:contains("Fixable CVEs")').click();
        cy.wait('@getFixableCvesForEntity');
        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains("CVE-2021-20231")`).contains(
            'Active'
        );
        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains("CVE-2021-20232")`).contains(
            'Inactive'
        );
    });
});
