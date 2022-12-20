export const selectors = {
    actions: {
        btn: 'td.pf-c-table__action button[aria-label="Actions"]',
        excludeDeploymentBtn: 'button:contains("Exclude deployment")',
        resolveBtn: 'button:contains("Mark as resolved")',
        resolveAndAddToBaselineBtn: 'button:contains("Resolve and add to process baseline")',
    },
    details: {
        page: '[data-testid="violation-details-page"]',
        title: 'h1.pf-c-title',
        subtitle: 'h2.pf-c-title',
        tabs: 'li.pf-c-tabs__item',
        violationTab: 'li.pf-c-tabs__item:contains("Violation")',
        enforcementTab: 'li.pf-c-tabs__item:contains("Enforcement")',
        deploymentTab: 'li.pf-c-tabs__item:contains("Deployment")',
        policyTab: 'li.pf-c-tabs__item:contains("Policy")',
    },
    enforcement: {
        detailMessage: '[data-testid="enforcement-detail-message"]',
        explanationMessage: '[data-testid="enforcement-explanation-message"]',
    },
    deployment: {
        overview: '[data-testid="deployment-details"] [data-testid="deployment-overview"]',
        containerConfiguration:
            '[data-testid="deployment-details"] [data-testid="container-configuration"]',
        securityContext: '[data-testid="deployment-details"] [data-testid="security-context"]',
        portConfiguration: '[data-testid="deployment-details"] [data-testid="port-configuration"]',
    },
};
