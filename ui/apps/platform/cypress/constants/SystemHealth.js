export const systemHealthUrl = '/main/system-health';

export const selectors = {
    bundle: {
        filterByStartingTime: '#filterByStartingTime',
        startingTimeMessage: '[data-testid="starting-time-message"]',
    },
    integrations: {
        errorMessage: '[data-testid="error-message"]',
        healthyText: '[data-testid="healthy-text"]',
        integrationName: '[data-testid="integration-name"]',
        integrationLabel: '[data-testid="integration-label"]',
        lastContact: '[data-testid="last-contact"]',
        viewAllButton: 'a:contains("View All")',
        widgets: {
            imageIntegrations: '[data-testid="image-integrations"]',
            notifierIntegrations: '[data-testid="notifier-integrations"]',
            backupIntegrations: '[data-testid="backup-integrations"]',
            declarativeConfigs: '[data-testid="declarative-configs"]',
        },
    },
};
