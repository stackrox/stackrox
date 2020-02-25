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

    // @TODO, uncomment when counts are available on entity page sub-list queries
    it('related entities tile links should unset search params upon navigation', () => {
        // arrange
        cy.visit(url.list.clusters);

        cy.get(selectors.tableRows)
            .eq(0)
            .get(selectors.fixableCvesLink)
            .click({ force: true });

        cy.get(selectors.backButton).click();
        cy.wait(1000);

        // act
        cy.get(selectors.deploymentTileLink)
            .find(selectors.tileLinkSuperText)
            .invoke('text')
            .then(numDeployments => {
                cy.get(selectors.deploymentTileLink)
                    // force: true option needed because this open issue for cypress
                    //   https://github.com/cypress-io/cypress/issues/4856
                    .click({ force: true });

                cy.get(`[data-test-id="side-panel"] [data-test-id="panel-header"]`)
                    .invoke('text')
                    .then(panelHeaderText => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(
                            parseInt(numDeployments, 10)
                        );
                    });
            });

        // assert
    });

    it('related entities table header should not say "0 entities" or have "page 0 of 0" if there are rows in the table', () => {
        cy.visit(url.list.policies);

        cy.get(selectors.deploymentCountLink)
            .eq(0)
            .click({ force: true });
        cy.wait(1000);

        cy.get(selectors.sidePanelTableBodyRows).then(value => {
            const { length: numRows } = value;
            if (numRows) {
                cy.get(selectors.entityRowHeader)
                    .invoke('text')
                    .then(headerText => {
                        expect(headerText).not.to.equal('0 deployments');
                    });

                cy.get(`${selectors.sidePanel} ${selectors.paginationHeader}`)
                    .invoke('text')
                    .then(paginationText => {
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
            .then(firstPolicyStatus => {
                cy.get(selectors.tableBodyRows)
                    .eq(0)
                    .click();
                cy.get(`${selectors.sidePanel} ${selectors.statusChips}`)
                    .eq(0)
                    .invoke('text')
                    .then(selectedPolicyStatus => {
                        expect(firstPolicyStatus).to.equal(selectedPolicyStatus);
                    });

                if (firstPolicyStatus === 'pass') {
                    cy.get(selectors.emptyFindingsSection).then(sectionElm => {
                        expect(sectionElm).to.have.length(1);
                    });

                    cy.get(selectors.deploymentTileLink)
                        .eq(0)
                        .click({ force: true });

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
            .then(selectedPolicyStatus => {
                cy.get(selectors.deploymentCountLink)
                    .eq(0)
                    .click({ force: true });
                cy.wait(1000);

                if (selectedPolicyStatus === 'pass') {
                    cy.get(
                        `${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`
                    ).should('not.exist');
                }
            });
    });

    it('should scope deployment data based on selected policy from entity page tab sublist', () => {
        cy.visit(url.list.policies);

        cy.get(selectors.statusChips)
            .eq(0)
            .invoke('text')
            .then(selectedPolicyStatus => {
                cy.get(selectors.deploymentCountLink)
                    .eq(0)
                    .click({ force: true });
                cy.wait(1000);

                cy.get(selectors.sidePanelExpandButton).click();
                cy.wait(1500);

                if (selectedPolicyStatus === 'pass') {
                    cy.get(
                        `${selectors.sidePanel} ${selectors.statusChips}:contains('fail')`
                    ).should('not.exist');
                }
            });
    });

    // test skipped because we are not currently showing the Policy (count) column, until and if performance can be improved
    it.skip('should have consistent policy count number from namespace list to policy sublist for a specific namespace', () => {
        cy.visit(url.list.namespaces);

        cy.get(selectors.policyCountLink)
            .eq(2)
            .invoke('text')
            .then(policyCountText => {
                cy.get(selectors.tableBodyRows)
                    .eq(2)
                    .click();
                cy.wait(1000);
                cy.get(selectors.policyTileLink)
                    .invoke('text')
                    .then(relatedPolicyCountText => {
                        expect(relatedPolicyCountText.toLowerCase().trim()).to.equal(
                            policyCountText.replace(' ', '')
                        );
                    });
                cy.get(selectors.policyTileLink).click({ force: true });
                cy.wait(1000);
                cy.get(selectors.entityRowHeader)
                    .invoke('text')
                    .then(paginationText => {
                        expect(paginationText).to.equal(policyCountText);
                    });
            });
    });

    it('should have filtered deployments list in 3rd level of side panel (namespaces -> policies -> deployments)', () => {
        cy.visit(url.list.namespaces);
        cy.wait(1000);

        cy.get(selectors.deploymentCountLink)
            .eq(0)
            .as('firstDeploymentCountLink');

        cy.get('@firstDeploymentCountLink').click();
        cy.get(selectors.parentEntityInfoHeader).click();
        cy.get(selectors.policyTileLink).click({ force: true });

        cy.get('@firstDeploymentCountLink')
            .invoke('text')
            .then(deploymentCountText => {
                cy.get(selectors.sidePanelTableBodyRows)
                    .eq(0)
                    .click();
                cy.wait(1000);
                cy.get(selectors.deploymentTileLink)
                    .invoke('text')
                    .then(relatedDeploymentCountText => {
                        expect(relatedDeploymentCountText.toLowerCase().trim()).to.equal(
                            deploymentCountText.replace(' ', '')
                        );
                    });
                cy.get(selectors.deploymentTileLink).click({ force: true });
                cy.wait(1000);
                cy.get(selectors.entityRowHeader)
                    .invoke('text')
                    .then(paginationText => {
                        expect(paginationText).to.equal(deploymentCountText);
                    });
            });
    });

    it('should filter deployment count in failing policies section in namespace findings by namespace', () => {
        cy.visit(url.list.namespaces);

        cy.get(selectors.deploymentCountLink)
            .eq(0)
            .as('firstDeploymentCountLink');

        // in side panel
        cy.get('@firstDeploymentCountLink')
            .invoke('text')
            .then(listDeploymentCountText => {
                cy.get('@firstDeploymentCountLink').click();
                cy.wait(1000);

                cy.get(selectors.parentEntityInfoHeader).click();
                cy.wait(1000);
                cy.get(selectors.deploymentCountText)
                    .eq(0)
                    .invoke('text')
                    .then(sidePanelDeploymentCountText => {
                        expect(listDeploymentCountText).to.equal(sidePanelDeploymentCountText);

                        // in entity page
                        cy.get(selectors.sidePanelExpandButton).click();
                        cy.get(selectors.deploymentCountText)
                            .eq(0)
                            .invoke('text')
                            .then(entityDeploymentCountText => {
                                expect(sidePanelDeploymentCountText).to.equal(
                                    entityDeploymentCountText
                                );
                            });
                    });
            });
    });
});
