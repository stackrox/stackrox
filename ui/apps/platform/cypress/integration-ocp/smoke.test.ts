import { visitFromConsoleLeftNavExpandable } from '../helpers/nav';
import { withOcpAuth } from '../helpers/ocpAuth';

describe('Basic tests of the OCP plugin', () => {
    it('should open the OCP web console', () => {
        withOcpAuth();

        cy.visit('/');

        cy.get('h1:contains("Overview")');
    });

    describe('Plugin version', () => {
        beforeEach(() => {
            withOcpAuth();
        });

        it('should display correct plugin version in the plugin manifest', () => {
            // The plugin manifest is served at the basePath configured in the ConsolePlugin resource
            // which is /api/plugins/<plugin-name>/plugin-manifest.json
            cy.request('/api/plugins/advanced-cluster-security/plugin-manifest.json').then(
                (response) => {
                    expect(response.status).to.equal(200);
                    expect(response.body).to.have.property('name', 'advanced-cluster-security');
                    expect(response.body).to.have.property('version');

                    expect(response.body.version).to.match(/^[1-9]\d*\.\d+\.\d+/);
                }
            );
        });

        it('should display plugin information in cluster settings', () => {
            visitFromConsoleLeftNavExpandable('Administration', 'Dynamic Plugins');

            cy.get('td[data-label="name"]:contains("advanced-cluster-security")')
                .parent()
                .within(() => {
                    cy.get('td[data-label="version"]')
                        .invoke('text')
                        .should('match', /^[1-9]\d*\.\d+\.\d+/);
                });
        });
    });
});
