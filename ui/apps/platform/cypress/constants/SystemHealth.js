export const systemHealthUrl = '/main/system-health';

export const selectors = {
    bundle: {
        startingDate: '#startingDate',
        startingTime: '#startingTime',
        filterByClusters: '#filterByClusters',
        isDatabaseDiagnosticsOnly: '#isDatabaseDiagnosticsOnly',
        includeComplianceOperatorResources: '#includeComplianceOperatorResources',
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
