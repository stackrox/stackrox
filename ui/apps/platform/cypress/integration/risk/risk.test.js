import { selectors as RiskPageSelectors, url, errorMessages } from '../../constants/RiskPage';
import { selectors as searchSelectors } from '../../constants/SearchPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    visitRiskDeployments,
    viewRiskDeploymentByName,
    viewRiskDeploymentInNetworkGraph,
} from '../../helpers/risk';

describe('Risk page', () => {
    withAuth();

    describe('with mock API', () => {
        beforeEach(() => {
            cy.intercept('GET', api.risks.riskyDeployments, {
                fixture: 'risks/riskyDeployments.json',
            }).as('deployments');
            cy.intercept('GET', api.risks.deploymentsCount).as('deploymentsCount');

            cy.visit(url);
            cy.wait('@deployments');
            cy.wait('@deploymentsCount');
        });

        const mockGetDeployment = () => {
            cy.intercept('GET', api.risks.fetchDeploymentWithRisk, {
                fixture: 'risks/firstDeployment.json',
            }).as('firstDeployment');
        };

        it('should have selected item in nav bar', () => {
            cy.get(RiskPageSelectors.risk).should('have.class', 'pf-m-current');
        });

        it('should sort priority in the table', () => {
            cy.get(RiskPageSelectors.table.column.priority).click({ force: true }); // ascending
            cy.get(RiskPageSelectors.table.column.priority).click({ force: true }); // descending
            cy.get(RiskPageSelectors.table.row.firstRow).should('contain', '3');
        });

        it('should highlight selected deployment row', () => {
            cy.get(RiskPageSelectors.table.row.firstRow)
                .click({ force: true })
                .should('have.class', 'row-active');
        });

        it('should display deployment error message in panel', () => {
            cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
            cy.get(RiskPageSelectors.errMgBox).contains(errorMessages.deploymentNotFound);
        });

        it('should display error message in process discovery tab', () => {
            mockGetDeployment();
            cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
            cy.wait('@firstDeployment');

            cy.get(RiskPageSelectors.panelTabs.processDiscovery).click();
            cy.get(RiskPageSelectors.errMgBox).contains(errorMessages.processNotFound);
            cy.get(RiskPageSelectors.cancelButton).click();
        });

        it('should open the panel to view risk indicators', () => {
            mockGetDeployment();
            cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
            cy.wait('@firstDeployment');

            cy.get(RiskPageSelectors.panelTabs.riskIndicators);
            cy.get(RiskPageSelectors.cancelButton).click();
        });

        it('should open the panel to view deployment details', () => {
            mockGetDeployment();
            cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
            cy.wait('@firstDeployment');

            cy.get(RiskPageSelectors.panelTabs.deploymentDetails);
            cy.get(RiskPageSelectors.cancelButton).click();
        });

        it('should navigate from Risk Page to Vulnerability Management Image Page', () => {
            mockGetDeployment();
            cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
            cy.wait('@firstDeployment');

            cy.get(RiskPageSelectors.panelTabs.deploymentDetails).click({ force: true });
            cy.get(RiskPageSelectors.imageLink).first().click({ force: true });
            cy.url().should('contain', '/main/vulnerability-management/image');
        });

        it.skip('should close the side panel on search filter', () => {
            mockGetDeployment();
            cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
            cy.wait('@firstDeployment');

            // The side panel opens to display the first deployment.
            // Use tabs as the criterion, because both the main and side panels have
            // [data-testid="panel"] nor [data-testid="panel-header"]
            cy.get(RiskPageSelectors.sidePanel.tabs);

            // TODO skip this test because Platform UI does not close the side panel,
            // even if the deployment does not match the search filter.
            // Assuming that the behavior changes, to make this test work,
            // it is necessary to mock the requests with search filter.
            // See the corresponding test in violations/violations.test.js
            cy.get(searchSelectors.pageSearch.input).type('Cluster:{enter}', { force: true });
            cy.get(searchSelectors.pageSearch.input).type('remote{enter}', { force: true });
            cy.get(searchSelectors.pageSearch.input).type('{esc}'); // close the drop-down menu
            cy.get(RiskPageSelectors.sidePanel.tabs).should('not.exist');
        });
    });

    describe('with actual API', () => {
        it('should navigate to network page with selected deployment', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('central');
            viewRiskDeploymentInNetworkGraph();

            cy.location('pathname').should('match', /^\/main\/network\/[-0-9a-z]+$/);
        });
    });

    describe('search with URL parameters, actual API', () => {
        beforeEach(() => {
            cy.intercept('GET', api.risks.riskyDeployments).as('deployments');
            cy.intercept('GET', api.risks.deploymentsCount).as('deploymentsCount');

            cy.visit(url);
            cy.wait('@deployments');
            cy.wait('@deploymentsCount');
        });

        it('should not have anything in search bar when URL has no search params', () => {
            cy.get(RiskPageSelectors.search.searchLabels).should('not.exist');
        });

        it('should have a single URL search param key/value pair in its search bar', () => {
            const nsOption = 'Namespace';
            const nsValue = 'stackrox';
            cy.get(RiskPageSelectors.table.dataRows)
                .filter(`:contains("${nsValue}")`)
                .then((stackroxDeps) => {
                    const stackroxCount = stackroxDeps.length;

                    const urlWithSearch = `${url}?s[${nsOption}]=${nsValue}`;
                    cy.visit(urlWithSearch);
                    cy.get(RiskPageSelectors.search.searchLabels)
                        .should('have.length', 2)
                        .each(($el, index) => {
                            if (index === 0) {
                                expect($el.text()).to.equal(`${nsOption}:`);
                            } else {
                                expect($el.text()).to.equal(nsValue);
                            }
                        });

                    cy.get(RiskPageSelectors.table.dataRows).should('have.length', stackroxCount);
                });
        });

        it('should have multiple URL search param key/value pairs in its search bar', () => {
            const nsOption = 'Namespace';
            const nsValue = 'stackrox';
            const deployOption = 'Deployment';
            const deployValue = 'scanner';
            cy.get(RiskPageSelectors.table.dataRows)
                .filter(`:contains("${deployValue}")`)
                .then((staticDeps) => {
                    const staticCount = staticDeps.length;

                    const urlWithSearch = `${url}?s[${nsOption}]=${nsValue}&s[${deployOption}]=${deployValue}`;
                    cy.visit(urlWithSearch);

                    cy.get(RiskPageSelectors.search.searchLabels)
                        .should('have.length', 4)
                        .each(($el, index) => {
                            // $el is a wrapped jQuery element
                            switch (index) {
                                case 0: {
                                    expect($el.text()).to.equal(`${nsOption}:`);
                                    break;
                                }
                                case 1: {
                                    expect($el.text()).to.equal(`${nsValue}`);
                                    break;
                                }
                                case 2: {
                                    expect($el.text()).to.equal(`${deployOption}:`);
                                    break;
                                }
                                case 3:
                                default: {
                                    expect($el.text()).to.equal(`${deployValue}`);
                                    break;
                                }
                            }
                        });

                    cy.get(RiskPageSelectors.table.dataRows).should('have.length', staticCount);
                });
        });

        it('should not use invalid URL search param key/value pair in its search bar', () => {
            const sillyOption = 'Wingardium';
            const sillyValue = 'leviosa';
            cy.get(RiskPageSelectors.table.dataRows).then((stackroxDeps) => {
                const allCount = stackroxDeps.length;

                const urlWithSearch = `${url}?s[${sillyOption}]=${sillyValue}`;
                cy.visit(urlWithSearch);

                cy.get(RiskPageSelectors.search.searchLabels).should('not.exist');

                cy.get(RiskPageSelectors.table.dataRows).should('have.length', allCount);
            });
        });
    });
});
