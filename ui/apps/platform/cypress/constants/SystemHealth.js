export const systemHealthUrl = '/main/system-health';

export const selectors = {
    bundle: {
        filterByStartingTime: '#filterByStartingTime',
        startingTimeMessage: '[data-testid="starting-time-message"]',
    },
    integrations: {
        errorMessage: '[data-label="Error messsage"]',
        healthyText: '[data-testid="healthy-text"]',
        integrationName: '[data-testid="integration-name"]',
        integrationLabel: '[data-testid="integration-label"]',
        lastContact: '[data-testid="last-contact"]',
        viewAllButton: 'a:contains("View All")',
        widgets: {
            declarativeConfigs: '[data-testid="declarative-configs"]',
        },
    },
};
