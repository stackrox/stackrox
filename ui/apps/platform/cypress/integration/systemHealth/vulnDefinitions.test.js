import { selectors } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import { setClock, visitSystemHealth } from '../../helpers/systemHealth';

const nbsp = '\u00A0';

describe('System Health Vulnerability Definitions without fixture', () => {
    withAuth();

    it('should have widget and up to date text', () => {
        visitSystemHealth();

        const { vulnDefinitions } = selectors;
        cy.get(vulnDefinitions.header).should('have.text', 'Vulnerability Definitions');
        cy.get(vulnDefinitions.text).should(
            'have.text',
            `Vulnerability definitions are up${nbsp}to${nbsp}date`
        );
    });
});

describe('System Health Vulnerability Definitions with fixture', () => {
    withAuth();

    it('should have widget and out of date text and time', () => {
        const currentDatetime = new Date('2020-12-10T03:04:59.377369440Z'); // exactly 24 hours after last updated
        const lastUpdatedTimestamp = '2020-12-09T03:04:59.377369440Z';

        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            'integrationhealth/vulndefinitions': { body: { lastUpdatedTimestamp } },
        });

        const { vulnDefinitions } = selectors;
        cy.get(vulnDefinitions.header).should('have.text', 'Vulnerability Definitions');
        cy.get(vulnDefinitions.text).should(
            'have.text',
            `Vulnerability definitions are out${nbsp}of${nbsp}date`
        );
    });
});
