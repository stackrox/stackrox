import pf6 from '../../selectors/pf6';

export const selectors = {
    actions: {
        btn: `td ${pf6.kebabToggle}[aria-label="Kebab toggle"]`, // via ActionsColumn element
        excludeDeploymentBtn: `${pf6.menuListButton}:contains("Exclude deployment from policy")`,
        resolveBtn: `${pf6.menuListButton}:contains("Mark as resolved")`,
        resolveAndAddToBaselineBtn: `${pf6.menuListButton}:contains("Resolve and add to process baseline")`,
    },
    details: {
        title: pf6.pageHeaderTitle,
        subtitle: pf6.pageHeaderSubtitle,
        tabs: pf6.tab,
        violationTab: `${pf6.tab}:contains("Violation")`,
        enforcementTab: `${pf6.tab}:contains("Enforcement")`,
        deploymentTab: `${pf6.tab}:contains("Deployment")`,
        policyTab: `${pf6.tab}:contains("Policy")`,
        networkPoliciesTab: `${pf6.tab}:contains("Network policies")`,
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
