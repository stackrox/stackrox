import { url, selectors } from '../../constants/SystemHealth';
import { integrationHealth as integrationHealthApi } from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

const nbsp = '\u00A0';

describe('System Health Vulnerability Definitions local deployment', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', integrationHealthApi.vulnDefinitions).as('GetVulnerabilityDefinitionsInfo');
    });

    it('should have widget and up to date text', () => {
        cy.visit(url.dashboard);
        cy.wait('@GetVulnerabilityDefinitionsInfo');

        const { vulnDefinitions } = selectors;
        cy.get(vulnDefinitions.header).should('have.text', 'Vulnerability Definitions');
        cy.get(vulnDefinitions.text).should(
            'have.text',
            `Vulnerability definitions are up${nbsp}to${nbsp}date`
        );
    });
});

describe('System Health Vulnerability Definitions fixtures', () => {
    withAuth();

    it('should have widget and out of date text and time', () => {
        const currentDatetime = new Date('2020-12-10T03:04:59.377369440Z'); // exactly 24 hours
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.server();
        cy.route('GET', integrationHealthApi.vulnDefinitions, {
            lastUpdatedTimestamp: '2020-12-09T03:04:59.377369440Z',
        }).as('GetVulnerabilityDefinitionsInfo');

        cy.visit(url.dashboard);
        cy.wait('@GetVulnerabilityDefinitionsInfo');

        const { vulnDefinitions } = selectors;
        cy.get(vulnDefinitions.header).should('have.text', 'Vulnerability Definitions');
        cy.get(vulnDefinitions.text).should(
            'have.text',
            `Vulnerability definitions are out${nbsp}of${nbsp}date`
        );
    });
});
