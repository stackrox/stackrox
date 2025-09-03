import withAuth from '../helpers/basicAuth';

describe('Basic Telemetry Configuration Checks', () => {
    withAuth();

    it('should call the correct configuration endpoints on app startup', () => {
        cy.intercept('GET', '/v1/config/public', (req) => {
            req.on('response', (res) => {
                // We only fetch telemetry config if the initial public config call shows that telemetry is enabled
                res.body.telemetry = { enabled: true, lastSetTime: null };
            });
        });

        // Ensure that the follow up telemetry config call is made
        cy.intercept('GET', '/v1/telemetry/config').as('telemetryConfig');
        cy.visit('/');
        cy.wait('@telemetryConfig');
    });

    it('should report a page view event when a page is loaded', () => {
        cy.spyTelemetry();
        cy.visit('/');
        cy.getTelemetryEvents().should((telemetryEvents) => {
            // Initial page view to '/'
            expect(telemetryEvents.page[0].type).to.equal('Page Viewed');
            expect(telemetryEvents.page[0].properties.path).to.equal('/');

            // Capture automatic redirect to '/main/dashboard'
            expect(telemetryEvents.page[1].type).to.equal('Page Viewed');
            expect(telemetryEvents.page[1].properties.path).to.equal('/main/dashboard');

            // Expect no events other than page views
            expect(telemetryEvents.track).to.have.length(0);
        });
    });
});
