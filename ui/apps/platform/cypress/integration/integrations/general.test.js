import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    assertIntegrationsTable,
    clickIntegrationTileOnTab,
    visitIntegrationsDashboard,
    visitIntegrationsDashboardFromLeftNav,
    visitIntegrationsTab,
} from './integrations.helpers';

describe('Integrations Dashboard', () => {
    withAuth();

    it('should visit via link in left nav', () => {
        visitIntegrationsDashboardFromLeftNav();
    });

    it('should have title', () => {
        visitIntegrationsDashboard();

        cy.title().should('match', getRegExpForTitleWithBranding('Integrations'));
    });

    it('Plugin tiles should all be the same height', () => {
        visitIntegrationsDashboard();

        let value = null;
        cy.get('a[data-testid="integration-tile"]').each(($el) => {
            if (value) {
                expect($el[0].clientHeight).to.equal(value);
            } else {
                value = $el[0].clientHeight;
            }
        });
    });

    // Page address segments are the source of truth for integrationSource and integrationType.

    it('should go to the table for a type of imageIntegrations', () => {
        const integrationSource = 'imageIntegrations';
        const integrationType = 'docker';

        visitIntegrationsDashboard();

        clickIntegrationTileOnTab(integrationSource, integrationType);

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for a type of notifiers', () => {
        const integrationSource = 'notifiers';
        const integrationType = 'slack';

        visitIntegrationsTab(integrationSource);

        clickIntegrationTileOnTab(integrationSource, integrationType);

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for a type of backups', () => {
        const integrationSource = 'backups';
        const integrationType = 's3';

        visitIntegrationsTab(integrationSource);

        clickIntegrationTileOnTab(integrationSource, integrationType);

        assertIntegrationsTable(integrationSource, integrationType);
    });

    it('should go to the table for apitoken type of authProviders', () => {
        const integrationSource = 'authProviders';
        const integrationType = 'apitoken';

        visitIntegrationsTab(integrationSource);

        clickIntegrationTileOnTab(integrationSource, integrationType);

        assertIntegrationsTable(integrationSource, integrationType);
    });
});
