export const selectors = {
    actions: {
        btn: 'td .pf-v6-c-menu-toggle[aria-label="Kebab toggle"]', // via ActionsColumn element
        excludeDeploymentBtn:
            '.pf-v6-c-menu__list button:contains("Exclude deployment from policy")',
        resolveBtn: '.pf-v6-c-menu__list button:contains("Mark as resolved")',
        resolveAndAddToBaselineBtn:
            '.pf-v6-c-menu__list button:contains("Resolve and add to process baseline")',
    },
    details: {
        title: '[data-ouia-component-id="PageHeader-title"]',
        subtitle: '[data-ouia-component-id="PageHeader-subtitle"]',
        tabs: 'li.pf-v6-c-tabs__item',
        violationTab: 'li.pf-v6-c-tabs__item:contains("Violation")',
        enforcementTab: 'li.pf-v6-c-tabs__item:contains("Enforcement")',
        deploymentTab: 'li.pf-v6-c-tabs__item:contains("Deployment")',
        policyTab: 'li.pf-v6-c-tabs__item:contains("Policy")',
        networkPoliciesTab: 'li.pf-v6-c-tabs__item:contains("Network policies")',
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
