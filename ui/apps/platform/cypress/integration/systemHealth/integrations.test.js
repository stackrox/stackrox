import { selectors } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import { interactAndVisitIntegrationsDashboardForSource } from '../integrations/integrations.helpers';
import { visitSystemHealth } from '../../helpers/systemHealth';

describe('System Health Integrations local deployment', () => {
    withAuth();

    it('should go to Image Integrations anchor on Integrations page via click View All', () => {
        visitSystemHealth();

        interactAndVisitIntegrationsDashboardForSource(() => {
            cy.get(
                `${selectors.integrations.widgets.imageIntegrations} ${selectors.integrations.viewAllButton}`
            ).click();
        }, 'imageIntegrations');
    });

    it('should go to Notifier Integrations anchor on Integrations page via click View All', () => {
        visitSystemHealth();

        interactAndVisitIntegrationsDashboardForSource(() => {
            cy.get(
                `${selectors.integrations.widgets.notifierIntegrations} ${selectors.integrations.viewAllButton}`
            ).click();
        }, 'notifiers');
    });

    it('should go to Backup Integrations anchor on Integrations page via click View All', () => {
        visitSystemHealth();

        interactAndVisitIntegrationsDashboardForSource(() => {
            cy.get(
                `${selectors.integrations.widgets.backupIntegrations} ${selectors.integrations.viewAllButton}`
            ).click();
        }, 'backups');
    });
});

describe('System Health Integrations fixtures', () => {
    withAuth();
    it('should not have count in healthy text for backup integrations', () => {
        const externalBackups = [];
        const integrationHealth = [];
        visitSystemHealth({
            externalbackups: { body: { externalBackups } },
            'integrationhealth/externalbackups': { body: { integrationHealth } },
        });

        const { healthyText, widgets } = selectors.integrations;
        cy.get(`${widgets.backupIntegrations} ${healthyText}`).should(
            'have.text',
            'No configured integrations'
        );
    });

    it('should have counts in healthy text for image integrations', () => {
        const integrations = [
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
        ];
        const integrationHealth = [
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
        ];
        visitSystemHealth({
            imageintegrations: { body: { integrations } },
            'integrationhealth/imageintegrations': { body: { integrationHealth } },
        });

        const { healthyText, widgets } = selectors.integrations;
        cy.get(`${widgets.imageIntegrations} ${healthyText}`).should(
            'have.text',
            '2 / 3 healthy integrations'
        );
    });

    it('should have count in healthy text for notifier integrations', () => {
        const notifiers = [
            {
                id: '4af2a32d-adeb-40ad-b509-0b191faecf7b',
                type: 'slack',
                // omit irrelevant properties
            },
        ];
        const integrationHealth = [
            {
                id: '4af2a32d-adeb-40ad-b509-0b191faecf7b',
                name: 'Slack',
                type: 'NOTIFIER',
                status: 'HEALTHY',
                errorMessage: '',
                lastTimestamp: '2020-12-09T17:52:18.743384877Z',
            },
        ];
        visitSystemHealth({
            notifiers: { body: { notifiers } },
            'integrationhealth/notifiers': { body: { integrationHealth } },
        });

        const { healthyText, widgets } = selectors.integrations;
        cy.get(`${widgets.notifierIntegrations} ${healthyText}`).should(
            'have.text',
            '1 healthy integration'
        );
    });

    it('should have a list with 1 Unhealthy image integration', () => {
        const integrations = [
            {
                id: '169b0d3f-8277-4900-bbce-1127077defae',
                name: 'StackRox Scanner',
                type: 'clairify',
            },
        ];
        const integrationHealth = [
            {
                id: '169b0d3f-8277-4900-bbce-1127077defae',
                name: 'StackRox Scanner',
                type: 'IMAGE_INTEGRATION',
                status: 'UNHEALTHY',
                errorMessage:
                    'Error scanning "docker.io/library/nginx:latest" with scanner "Stackrox Scanner": dial tcp 10.0.1.229:5432: connect: connection refused',
                lastTimestamp: '2020-12-04T00:38:17.906318735Z',
            },
        ];
        visitSystemHealth({
            imageintegrations: { body: { integrations } },
            'integrationhealth/imageintegrations': { body: { integrationHealth } },
        });

        const { integrationLabel, integrationName, widgets } = selectors.integrations;

        [
            {
                name: 'StackRox Scanner',
                label: 'StackRox Scanner',
                errorMessage:
                    'Error scanning "docker.io/library/nginx:latest" with scanner "Stackrox Scanner": dial tcp 10.0.1.229:5432: connect: connection refused',
            },
        ].forEach(({ name }, i) => {
            const itemSelector = `${widgets.imageIntegrations} li:nth-child(${i + 1})`;
            cy.get(`${itemSelector} ${integrationName}`).should('have.text', name);
            cy.get(`${itemSelector} ${integrationLabel}`).should('not.exist'); // because redundant
        });
    });

    it('should have a list with 1 declarative configuration error', () => {
        const healthName = 'Config Map declarative-configuration';
        const errorMessageText = 'this is error message';
        const integrationHealth = [
            {
                id: '169b0d3f-8277-4900-bbce-1127077defae',
                name: healthName,
                type: 'DECLARATIVE_CONFIG',
                status: 'UNHEALTHY',
                errorMessage: errorMessageText,
                lastTimestamp: '2020-12-04T00:38:17.906318735Z',
            },
        ];

        visitSystemHealth({
            'integrationhealth/declarativeconfigs': { body: { integrationHealth } },
        });

        const { widgets } = selectors.integrations;
        const itemSelector = `${widgets.declarativeConfigs} tr:first`;
        cy.get(`${itemSelector} td[data-label="Name"]`).should('have.text', healthName);
        cy.get(`${itemSelector} td[data-label="Error"]`).should('have.text', errorMessageText);
    });

    it('should have no declarative configuration errors displayed', () => {
        const healthName = 'Config Map declarative-configuration';
        const integrationHealth = [
            {
                id: '169b0d3f-8277-4900-bbce-1127077defae',
                name: healthName,
                type: 'DECLARATIVE_CONFIG',
                status: 'HEALTHY',
                errorMessage: '',
                lastTimestamp: '2020-12-04T00:38:17.906318735Z',
            },
        ];

        visitSystemHealth({
            'integrationhealth/declarativeconfigs': { body: { integrationHealth } },
        });
        const { widgets } = selectors.integrations;
        const itemSelector = `${widgets.declarativeConfigs} tr:first`;
        cy.get(`${itemSelector} td[data-label="Name"]`).should('not.exist');
    });
});
