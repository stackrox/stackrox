import { selectors } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import {
    integrationHealthVulnDefinitionsAlias,
    setClock,
    visitSystemHealth,
} from '../../helpers/systemHealth';

const nbsp = '\u00A0';

describe('System Health Vulnerability Definitions', () => {
    withAuth();

    it('should have widget and up to date text', () => {
        const currentDatetime = new Date('2020-12-10T02:04:59.377369440Z'); // exactly 23 hours after last updated
        const lastUpdatedTimestamp = '2020-12-09T03:04:59.377369440Z';

        const staticResponseMap = {
            [integrationHealthVulnDefinitionsAlias]: { body: { lastUpdatedTimestamp } },
        };

        setClock(currentDatetime); // call before visit
        visitSystemHealth(staticResponseMap);

        const { vulnDefinitions } = selectors;
        cy.get(vulnDefinitions.header).should('have.text', 'Vulnerability Definitions');
        cy.get(vulnDefinitions.text).should(
            'have.text',
            `Vulnerability definitions are up${nbsp}to${nbsp}date`
        );
    });

    it('should have widget and out of date text and time', () => {
        const currentDatetime = new Date('2020-12-10T03:04:59.377369440Z'); // exactly 24 hours after last updated
        const lastUpdatedTimestamp = '2020-12-09T03:04:59.377369440Z';

        const staticResponseMap = {
            [integrationHealthVulnDefinitionsAlias]: { body: { lastUpdatedTimestamp } },
        };

        setClock(currentDatetime); // call before visit
        visitSystemHealth(staticResponseMap);

        const { vulnDefinitions } = selectors;
        cy.get(vulnDefinitions.header).should('have.text', 'Vulnerability Definitions');
        cy.get(vulnDefinitions.text).should(
            'have.text',
            `Vulnerability definitions are out${nbsp}of${nbsp}date`
        );
    });
});
