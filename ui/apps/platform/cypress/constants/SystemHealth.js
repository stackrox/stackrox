const baseURL = '/main/system-health';

export const url = {
    dashboard: baseURL,
};

export const selectors = {
    clusters: {
        categoryCount: '[data-testid="count"]',
        categoryLabel: '[data-testid="label"]',
        healthyText: '[data-testid="healthy-text"]',
        healthySubtext: '[data-testid="healthy-subtext"]',
        problemCount: '[data-testid="problem-count"]',
        viewAllButton: '[data-testid="cluster-health"] a:contains("View All")',
        widgets: {
            clusterOverview: '[data-testid="cluster-overview"]',
            collectorStatus: '[data-testid="collector-status"]',
            sensorStatus: '[data-testid="sensor-status"]',
            sensorUpgrade: '[data-testid="sensor-upgrade"]',
            credentialExpiration: '[data-testid="credential-expiration"]',
        },
    },
    integrations: {
        viewAllButton: 'a:contains("View All")',
        widgets: {
            imageIntegrations: '[data-testid="image-integrations"]',
            pluginIntegrations: '[data-testid="plugin-integrations"]',
            backupIntegrations: '[data-testid="backup-integrations"]',
        },
    },
};
