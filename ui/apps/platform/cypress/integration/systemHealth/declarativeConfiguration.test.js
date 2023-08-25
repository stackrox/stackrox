import withAuth from '../../helpers/basicAuth';
import {
    visitSystemHealthWithStaticResponseForCapabilities,
    integrationHealthDeclarativeConfigsAlias,
} from '../../helpers/systemHealth';

describe('System Health Declarative Configuration', () => {
    withAuth();
    const declarativeConfigHeadingSelector = 'h2:contains("Declarative configuration")';

    it('should display declarative configuration when capability is available', () => {
        visitSystemHealthWithStaticResponseForCapabilities({
            body: {
                centralCanDisplayDeclarativeConfigHealth: 'CapabilityAvailable',
            },
        });

        cy.get(declarativeConfigHeadingSelector);
    });

    it('should display declarative configuration when central capabilities return an empty object', () => {
        visitSystemHealthWithStaticResponseForCapabilities({
            body: {},
        });

        cy.get(declarativeConfigHeadingSelector);
    });

    it('should not display declarative configuration when capability is disabled', () => {
        visitSystemHealthWithStaticResponseForCapabilities(
            {
                body: {
                    centralCanDisplayDeclarativeConfigHealth: 'CapabilityDisabled',
                },
            },
            [integrationHealthDeclarativeConfigsAlias]
        );

        cy.get(declarativeConfigHeadingSelector).should('not.exist');
    });
});
