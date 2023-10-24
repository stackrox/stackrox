export const selectors = {
    actions: {
        btn: 'td .pf-c-dropdown button[aria-label="Actions"]', // via ActionsColumn element
        excludeDeploymentBtn: 'button:contains("Exclude deployment")',
        resolveBtn: 'button:contains("Mark as resolved")',
        resolveAndAddToBaselineBtn: 'button:contains("Resolve and add to process baseline")',
    },
    details: {
        title: 'h1.pf-c-title',
        subtitle: 'h2.pf-c-title',
        tabs: 'li.pf-c-tabs__item',
        violationTab: 'li.pf-c-tabs__item:contains("Violation")',
        enforcementTab: 'li.pf-c-tabs__item:contains("Enforcement")',
        deploymentTab: 'li.pf-c-tabs__item:contains("Deployment")',
        policyTab: 'li.pf-c-tabs__item:contains("Policy")',
        networkPoliciesTab: 'li.pf-c-tabs__item:contains("Network policies")',
    },
    enforcement: {
        detailMessage: '[aria-label="Enforcement detail message"]',
        explanationMessage: '[aria-label="Enforcement explanation message"]',
    },
    deployment: {
        overview: `[aria-label="Deployment details"] article:has('h3:contains("Deployment overview")')`,
        containerConfiguration: `[aria-label="Deployment details"] article:has('h3:contains("Container configuration")')`,
        securityContext: `[aria-label="Deployment details"] article:has('h3:contains("Security context")')`,
        portConfiguration: `[aria-label="Deployment details"] article:has('h3:contains("Port configuration")')`,
    },
};
