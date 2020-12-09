import { url, selectors } from '../../constants/SystemHealth';
import {
    integrationHealth as integrationHealthApi,
    integrations as integrationsApi,
} from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

describe('System Health Integrations local deployment', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', integrationHealthApi.imageIntegrations).as('GetImageIntegrationsHealth');
        cy.route('GET', integrationHealthApi.notifiers).as('GetNotifiersHealth');
        cy.route('GET', integrationHealthApi.externalBackups).as('GetExternalBackupsHealth');
        cy.route('GET', integrationsApi.imageIntegrations).as('GetImageIntegrations');
        cy.route('GET', integrationsApi.notifiers).as('GetNotifiers');
        cy.route('GET', integrationsApi.externalBackups).as('GetExternalBackups');
    });

    const allApis = [
        '@GetImageIntegrationsHealth',
        '@GetNotifiersHealth',
        '@GetExternalBackupsHealth',
        '@GetImageIntegrations',
        '@GetNotifiers',
        '@GetExternalBackups',
    ];

    it('should go from left navigation to Dashboard and have widgets', () => {
        cy.visit('/');
        cy.get('nav.left-navigation a:contains("Platform Configuration")').click();
        cy.get('[data-testid="configure-subnav"] a:contains("System Health")').click();
        cy.wait(allApis);

        cy.get('[data-testid="header-text"]').should('have.text', 'System Health');

        Object.entries({
            imageIntegrations: 'Image Integrations',
            pluginIntegrations: 'Plugin Integrations',
            backupIntegrations: 'Backup Integrations',
        }).forEach(([key, text]) => {
            cy.get(`${selectors.integrations.widgets[key]} [data-testid="widget-header"]`).should(
                'have.text',
                text
            );
        });
    });

    it('should go to Images anchor on Integrations page via click View All', () => {
        cy.visit(url.dashboard);
        cy.wait('@GetImageIntegrations');

        cy.get(
            `${selectors.integrations.widgets.imageIntegrations} ${selectors.integrations.viewAllButton}`
        ).click();
        cy.wait('@GetImageIntegrations');

        cy.get('[data-testid="header-text"]').should('have.text', 'Integrations');
        cy.get('#image-integrations h2:contains("Images")').should('be.visible');
    });

    it('should go to Plugins anchor on Integrations page via click View All', () => {
        cy.visit(url.dashboard);
        cy.wait('@GetNotifiers');

        cy.get(
            `${selectors.integrations.widgets.pluginIntegrations} ${selectors.integrations.viewAllButton}`
        ).click();
        cy.wait('@GetNotifiers');

        cy.get('[data-testid="header-text"]').should('have.text', 'Integrations');
        cy.get('#plugin-integrations h2:contains("Plugins")').should('be.visible');
    });

    it('should go to External Backups anchor on Integrations page via click View All', () => {
        cy.visit(url.dashboard);
        cy.wait('@GetExternalBackups');

        cy.get(
            `${selectors.integrations.widgets.backupIntegrations} ${selectors.integrations.viewAllButton}`
        ).click();
        cy.wait('@GetExternalBackups');

        cy.get('[data-testid="header-text"]').should('have.text', 'Integrations');
        cy.get('#backup-integrations h2:contains("External Backups")').should('be.visible');
    });
});

describe('System Health Integrations fixtures', () => {
    withAuth();

    const { integrations } = selectors;

    it('should have counts in healthy text', () => {
        cy.server();

        // 2 image integrations are healthy
        cy.route('GET', integrationHealthApi.imageIntegrations, {
            integrationHealth: [
                {
                    id: '05fea766-e2f8-44b3-9959-eaa61a4f7466',
                    name: 'Public GCR',
                    type: 'IMAGE_INTEGRATION',
                    status: 'UNINITIALIZED',
                    errorMessage: '',
                    lastTimestamp: '2020-12-09T15:11:16.942655900Z',
                },
                {
                    id: '10d3b4dc-8295-41bc-bb50-6da5484cdb1a',
                    name: 'Public DockerHub',
                    type: 'IMAGE_INTEGRATION',
                    status: 'HEALTHY',
                    errorMessage: '',
                    lastTimestamp: '2020-12-09T15:15:19.318789700Z',
                },
                {
                    id: '169b0d3f-8277-4900-bbce-1127077defae',
                    name: 'Stackrox Scanner',
                    type: 'IMAGE_INTEGRATION',
                    status: 'HEALTHY',
                    errorMessage: '',
                    lastTimestamp: '2020-12-09T15:15:38.327627700Z',
                },
            ],
        }).as('GetImageIntegrationsHealth');
        cy.route('GET', integrationsApi.imageIntegrations, {
            integrations: [
                {
                    id: '05fea766-e2f8-44b3-9959-eaa61a4f7466',
                    type: 'docker',
                    // omit irrelevant properties
                },
                {
                    id: '10d3b4dc-8295-41bc-bb50-6da5484cdb1a',
                    type: 'docker',
                    // omit irrelevant properties
                },
                {
                    id: '169b0d3f-8277-4900-bbce-1127077defae',
                    type: 'clairify',
                    // omit irrelevant properties
                },
            ],
        }).as('GetImageIntegrations');

        // 1 plugin integration is healthy
        cy.route('GET', integrationHealthApi.notifiers, {
            integrationHealth: [
                {
                    id: '4af2a32d-adeb-40ad-b509-0b191faecf7b',
                    name: 'Slack',
                    type: 'NOTIFIER',
                    status: 'HEALTHY',
                    errorMessage: '',
                    lastTimestamp: '2020-12-09T17:52:18.743384877Z',
                },
            ],
        }).as('GetNotifiersHealth');
        cy.route('GET', integrationsApi.notifiers, {
            notifiers: [
                {
                    id: '4af2a32d-adeb-40ad-b509-0b191faecf7b',
                    type: 'slack',
                    // omit irrelevant properties
                },
            ],
        }).as('GetNotifiers');

        cy.visit(url.dashboard);
        cy.wait([
            '@GetImageIntegrationsHealth',
            '@GetNotifiersHealth',
            '@GetImageIntegrations',
            '@GetNotifiers',
        ]);

        const { healthyText, widgets } = selectors.integrations;

        Object.entries({
            imageIntegrations: '2 image integrations are healthy',
            pluginIntegrations: '1 plugin integration is healthy',
            backupIntegrations: '0 backup integrations are healthy',
        }).forEach(([key, text]) => {
            cy.get(`${widgets[key]} ${healthyText}`).should('have.text', text);
        });
    });

    it('should have a list with 1 Unhealthy image integration', () => {
        cy.server();
        cy.route('GET', integrationHealthApi.imageIntegrations, {
            integrationHealth: [
                {
                    id: '169b0d3f-8277-4900-bbce-1127077defae',
                    name: 'StackRox Scanner',
                    type: 'IMAGE_INTEGRATION',
                    status: 'UNHEALTHY',
                    errorMessage:
                        'Error scanning "docker.io/library/nginx:latest" with scanner "Stackrox Scanner": dial tcp 10.0.1.229:5432: connect: connection refused',
                    lastTimestamp: '2020-12-04T00:38:17.906318735Z',
                },
            ],
        }).as('GetImageIntegrationsHealth');
        cy.route('GET', integrationsApi.imageIntegrations, {
            integrations: [
                {
                    id: '169b0d3f-8277-4900-bbce-1127077defae',
                    name: 'StackRox Scanner',
                    type: 'clairify',
                },
            ],
        }).as('GetImageIntegrations');

        cy.visit(url.dashboard);
        cy.wait(['@GetImageIntegrationsHealth', '@GetImageIntegrations']);

        const widgetSelector = integrations.widgets.imageIntegrations;
        [
            {
                name: 'StackRox Scanner',
                label: 'StackRox Scanner',
                errorMessage:
                    'Error scanning "docker.io/library/nginx:latest" with scanner "Stackrox Scanner": dial tcp 10.0.1.229:5432: connect: connection refused',
            },
        ].forEach(({ name, label, errorMessage }, i) => {
            const itemSelector = `${widgetSelector} li:nth-child(${i + 1})`;
            cy.get(`${itemSelector} ${integrations.integrationName}`).should('have.text', name);
            cy.get(`${itemSelector} ${integrations.integrationLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${integrations.errorMessage}`).should(
                'have.text',
                errorMessage
            );
        });
    });
});
