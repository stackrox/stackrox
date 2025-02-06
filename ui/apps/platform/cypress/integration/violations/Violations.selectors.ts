export const selectors = {
    actions: {
        btn: 'td .pf-v5-c-menu-toggle[aria-label="Kebab toggle"]', // via ActionsColumn element
        excludeDeploymentBtn: 'button:contains("Exclude deployment")',
        resolveBtn: 'button:contains("Mark as resolved")',
        resolveAndAddToBaselineBtn: 'button:contains("Resolve and add to process baseline")',
    },
    details: {
        title: 'h1.pf-v5-c-title',
        subtitle: 'h2.pf-v5-c-title',
        tabs: 'li.pf-v5-c-tabs__item',
        violationTab: 'li.pf-v5-c-tabs__item:contains("Violation")',
        enforcementTab: 'li.pf-v5-c-tabs__item:contains("Enforcement")',
        deploymentTab: 'li.pf-v5-c-tabs__item:contains("Deployment")',
        policyTab: 'li.pf-v5-c-tabs__item:contains("Policy")',
        networkPoliciesTab: 'li.pf-v5-c-tabs__item:contains("Network policies")',
    },
    enforcement: {
        detailMessage: '[aria-label="Enforcement detail message"]',
        explanationMessage: '[aria-label="Enforcement explanation message"]',
    },
    deployment: {
        overview: `[aria-label="Deployment details"]:has('h3:contains("Deployment overview")')`,
        containerConfiguration: `[aria-label="Deployment details"]:has('h3:contains("Container configuration")')`,
        securityContext: `[aria-label="Deployment details"]:has('h3:contains("Security context")')`,
        portConfiguration: `[aria-label="Deployment details"]:has('h3:contains("Port configuration")')`,
    },
};
