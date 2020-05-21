import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

describe('Entities single views', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();

    it('related entities tile links should unset search params upon navigation', () => {
        // arrange
        cy.visit(url.list.clusters);

        cy.get(selectors.tableRows, { timeout: 4000 })
            .find(selectors.fixableCvesLink)
            .eq(0)
            .click({ force: true });

        cy.get(selectors.backButton).click();

        // act
        cy.get(selectors.deploymentTileLink, { timeout: 1000 })
            .find(selectors.tileLinkSuperText)
            .invoke('text')
            .then((numDeployments) => {
                cy.log('numDeployments', numDeployments);
                cy.get(selectors.deploymentTileLink)
                    // force: true option needed because this open issue for cypress
                    //   https://github.com/cypress-io/cypress/issues/4856
                    .click({ force: true });

                cy.get(`[data-testid="side-panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(
                            parseInt(numDeployments, 10)
                        );
                    });
            });

        // assert
    });

    it('related entities table header should not say "0 entities" or have "page 0 of 0" if there are rows in the table', () => {
        cy.visit(url.list.policies);

        cy.get(`${selectors.deploymentCountLink}:eq(0)`).click({ force: true });

        cy.get(selectors.sidePanelTableBodyRows, { timeout: 9000 }).then((value) => {
            const { length: numRows } = value;
            if (numRows) {
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
        cy.visit(url.list.policies);

        cy.get(selectors.statusChips)
            .eq(0)
            .invoke('text')
            .then((firstPolicyStatus) => {
                cy.get(selectors.tableBodyRows).eq(0).click();

                cy.get(`${selectors.sidePanel} ${selectors.statusChips}`, { timeout: 9000 })
                    .eq(0)
                    .invoke('text')
                    .then((selectedPolicyStatus) => {
                        expect(firstPolicyStatus).to.equal(selectedPolicyStatus);
                    });

                if (firstPolicyStatus === 'pass') {
                    cy.get(selectors.emptyFindingsSection).then((sectionElm) => {
                        expect(sectionElm).to.have.length(1);
                    });

                    cy.get(selectors.deploymentTileLink).eq(0).click({ force: true });

                    cy.get(
                        `${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`
                    ).should('not.exist');
                }
            });
    });

    it('should scope deployment data based on selected policy from table count link click', () => {
        cy.visit(url.list.policies);

        cy.get(selectors.statusChips)
            .eq(0)
            .invoke('text')
            .then((selectedPolicyStatus) => {
                cy.get(`${selectors.deploymentCountLink}:eq(0)`).click({ force: true });

                if (selectedPolicyStatus === 'pass') {
                    cy.get(`${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`, {
                        timeout: 1000,
                    }).should('not.exist');
                }
            });
    });

    it('should scope deployment data based on selected policy from entity page tab sublist', () => {
        cy.visit(url.list.policies);

        cy.get(selectors.statusChips)
            .eq(0)
            .invoke('text')
            .then((selectedPolicyStatus) => {
                cy.get(`${selectors.deploymentCountLink}:eq(0)`).click({ force: true });

                cy.get(selectors.sidePanelExpandButton, { timeout: 5000 }).click();
                if (selectedPolicyStatus === 'pass') {
                    cy.get(`${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`, {
                        timeout: 1500,
                    }).should('not.exist');
                }
            });
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
        cy.visit(url.list.namespaces);

        cy.get(`${selectors.deploymentCountLink}:eq(0)`).as('firstDeploymentCountLink');

        cy.get('@firstDeploymentCountLink').click({ force: true });
        cy.get(selectors.parentEntityInfoHeader).click({ force: true });
        cy.get(selectors.policyTileLink, { timeout: 16000 }).click({ force: true });

        cy.get('@firstDeploymentCountLink')
            .invoke('text')
            .then((deploymentCountText) => {
                cy.get(selectors.sidePanelTableBodyRows).eq(0).click();

                cy.get(selectors.deploymentTileLink, { timeout: 10000 })
                    .invoke('text')
                    .then((relatedDeploymentCountText) => {
                        expect(relatedDeploymentCountText.toLowerCase().trim()).to.equal(
                            deploymentCountText.replace(' ', '')
                        );
                    });
                cy.get(selectors.deploymentTileLink, { timeout: 10000 }).click({ force: true });

                cy.get(selectors.entityRowHeader, { timeout: 8000 })
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

    it('should filter component count in images list and image overview by cve when coming from cve list', () => {
        cy.visit(url.list.cves);

        cy.get(selectors.imageCountLink, { timeout: 5000 }).eq(0).click({ force: true });
        cy.get(selectors.parentEntityInfoHeader).click();
        cy.get(selectors.imageTileLink).click({ force: true });

        cy.get(`${selectors.sidePanel} ${selectors.componentCountLink}`)
            .eq(0)
            .invoke('text')
            .then((componentCountText) => {
                cy.get(selectors.sidePanelTableBodyRows).eq(0).click();
                cy.get(selectors.componentTileLink, { timeout: 5000 })
                    .invoke('text')
                    .then((relatedComponentCountText) => {
                        expect(relatedComponentCountText.toLowerCase().trim()).to.equal(
                            componentCountText.replace(' ', '')
                        );
                    });
            });
    });

    it('should show a CVE description in overview when coming from cve list', () => {
        cy.visit(url.list.cves);

        cy.get(selectors.tableRows, { timeout: 5000 }).eq(1).click();
        cy.get(selectors.entityOverview)
            .find(selectors.cveDescription)
            .invoke('text')
            .then((value) => {
                expect(value.length).to.be.greaterThan(2);
                expect(value).not.to.include('No description available');
            });
    });

    it('should not filter cluster entity page regardless of entity context', () => {
        cy.visit(url.list.namespaces);

        cy.get(`${selectors.tableRows}:contains(No deployments)`, { timeout: 10000 }).eq(0).click();
        cy.get(selectors.metadataClusterValue).click();
        cy.wait(1000);
        cy.get(`${selectors.sidePanel} ${selectors.tableRows}:contains(No deployments)`).should(
            'not.exist'
        );
    });
});
