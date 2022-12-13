import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

import {
    interactAndWaitForVulnerabilityManagementEntity,
    visitVulnerabilityManagementDashboard,
} from './vulnerabilityManagement.helpers';
import { selectors } from './vulnerabilityManagement.selectors';

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

function getItemTextSelectorForWidget(widgetHeading, itemTextSelector) {
    return `[data-testid="widget"]:contains("${widgetHeading}") ${itemTextSelector}:eq(0)`;
}

const itemTextSelectorForNumberedList = '[data-testid="numbered-list-item-name"]';
const itemTextSelectorForLabelText = '.rv-xy-plot__series--label text';
const itemTextSelectorForClusters = 'li > div > div > div';

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
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
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
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has link from Top Riskiest Node Components widget to node component page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'node-components';
        const widgetHeading = 'Top Riskiest Node Components';

        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
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
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
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
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has link from Frequently Violated Policies widget to policy page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'policies';
        const widgetHeading = 'Frequently Violated Policies';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForLabelText),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    it.skip('has link from Recently Detected Image Vulnerabilities widget to vulnerability page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'cves'; // TODO enable test when we decide whether request will be getCve or getImageCve
        const widgetHeading = 'Recently Detected Image Vulnerabilities';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    // Vulnerability graphQL resolver is not support on postgres. Use Image/Node/ClusterVulnerability resolver.
    it.skip('has link from Most Common Image Vulnerabilities widget to vulnerability page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-cves'; // TODO enable test when we decide whether request will be getCve or getImageCve
        const widgetHeading = 'Most Common Image Vulnerabilities';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForLabelText),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    it('has link from Deployments With Most Severe Policy Violations widget to deployment page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'deployments';
        const widgetHeading = 'Deployments With Most Severe Policy Violations';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has link from Clusters With Most Orchestrator & Istio Vulnerabilities to cluster page', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const widgetHeading = 'Clusters With Most Orchestrator & Istio Vulnerabilities';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForClusters),
            (itemText) => itemText
        );
    });
});
