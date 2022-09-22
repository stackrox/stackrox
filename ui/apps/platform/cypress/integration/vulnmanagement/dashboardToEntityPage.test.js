import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    interactAndWaitForVulnerabilityManagementEntity,
    visitVulnerabilityManagementDashboard,
} from '../../helpers/vulnmanagement/entities';

function verifyItemLinkToEntityPage(entitiesKey, itemTextSelector, getHeaderTextFromItemText) {
    cy.get(itemTextSelector)
        .invoke('text')
        .then((itemText) => {
            const headerText = getHeaderTextFromItemText(itemText);

            interactAndWaitForVulnerabilityManagementEntity(() => {
                cy.get(itemTextSelector).click();
            }, entitiesKey);

            cy.get(`${selectors.entityPageHeader}:contains("${headerText}")`);
        });
}

function getItemTextSelectorForWidget(
    widgetHeading,
    itemTextSelector = '[data-testid="numbered-list-item-name"]'
) {
    return `[data-testid="widget"]:contains("${widgetHeading}") ${itemTextSelector}:eq(0)`;
}

function getHeaderTextFromItemTextWithColonSeparators(itemText) {
    const [, itemTextAfterNumberBeforeSlash] = /^\d+\.([^:]+):.*$/.exec(itemText);
    return itemTextAfterNumberBeforeSlash.trim();
}

function getHeaderTextFromItemTextWithSlashSeparators(itemText) {
    const [, itemTextAfterNumberBeforeSlash] = /^\d+\.([^/]+)\/.*$/.exec(itemText);
    return itemTextAfterNumberBeforeSlash.trim();
}

function getHeaderTextFromItemTextWithoutSeparators(itemText) {
    const [, itemTextAfterNumber] = /^\d+\.(.+)$/.exec(itemText);
    return itemTextAfterNumber.trim();
}

function selectTopRiskiestOption(optionText) {
    const widgetSelector = selectors.getWidget('Top Riskiest');
    cy.get(`${widgetSelector} .react-select__control`).click();
    cy.get(`${widgetSelector} .react-select__option:contains("${optionText}")`).click();
}

describe('Vulnerability Management Dashboard', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
            this.skip();
        }
    });

    // Some tests might fail in local deployment.

    it('has link from Top Riskiest Images widget to image page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const widgetHeading = 'Top Riskiest Images';
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has link from Top Riskiest Node Components widget to node component page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'components'; // page makes singular request for components instead of node-components
        const widgetHeading = 'Top Riskiest Node Components';
        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithColonSeparators
        );
    });

    it('has link from Top Riskiest Image Components widget to image component page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-components';
        const widgetHeading = 'Top Riskiest Image Components';
        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithColonSeparators
        );
    });

    it('has link from Top Riskiest Nodes widget to node page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'nodes';
        const widgetHeading = 'Top Riskiest Nodes';
        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has link from Frequently Violated Policies widget to policy page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'policies';
        const widgetHeading = 'Frequently Violated Policies';
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    it('has link from Recently Detected Image Vulnerabilities widget to vulnerability page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'cves'; // page makes singular request for cves instead of image-cves
        const widgetHeading = 'Recently Detected Image Vulnerabilities';
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    it('has link from Most Common Image Vulnerabilities widget to vulnerability page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'cves'; // page makes singular request for cves instead of image-cves
        const widgetHeading = 'Most Common Image Vulnerabilities';
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, '.rv-xy-plot__series--label text'),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    it('has link from Deployments With Most Severe Policy Violations widget to deployment page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'deployments';
        const widgetHeading = 'Deployments With Most Severe Policy Violations';
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has link from Clusters With Most Orchestrator & Istio Vulnerabilities to cluster page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const widgetHeading = 'Clusters With Most Orchestrator & Istio Vulnerabilities';
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, 'li > div > div > div'),
            (itemText) => itemText
        );
    });
});
