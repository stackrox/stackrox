import withAuth from '../../helpers/basicAuth';
import {
    interactAndWaitForVulnerabilityManagementEntity,
    visitVulnerabilityManagementDashboard,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

function verifyItemLinkToEntityPage(entitiesKey, itemTextSelector, getHeaderTextFromItemText) {
    cy.get(itemTextSelector)
        .invoke('text')
        .then((itemText) => {
            const headerText = getHeaderTextFromItemText(itemText);

            interactAndWaitForVulnerabilityManagementEntity(() => {
                cy.get(itemTextSelector).click();
            }, entitiesKey);

            cy.get(`[data-testid="header-text"]:contains("${headerText}")`);
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
    const widgetSelector = selectors.getWidget('Top riskiest');
    cy.get(`${widgetSelector} .react-select__control`).click();
    cy.get(`${widgetSelector} .react-select__option:contains("${optionText}")`).click();
}

describe('Vulnerability Management Dashboard', () => {
    withAuth();

    // Some tests might fail in local deployment.

    it('has item link to image page from Top riskiest images', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const widgetHeading = 'Top riskiest images';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    it('has item link to node component page from Top riskiest node components', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'node-components';
        const widgetHeading = 'Top riskiest node components';

        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithColonSeparators
        );
    });

    it('has item link to image component page from Top riskiest image components', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-components';
        const widgetHeading = 'Top riskiest image components';

        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithColonSeparators
        );
    });

    it('has item link to node page from Top riskiest nodes', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'nodes';
        const widgetHeading = 'Top riskiest nodes';

        selectTopRiskiestOption(widgetHeading);
        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithoutSeparators
        );
    });

    // TODO test fails because of product problem that page does not render the CVE id.
    it.skip('has item link to image CVE page from Recently detected image vulnerabilities', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-cves';
        const widgetHeading = 'Recently detected image vulnerabilities';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForNumberedList),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    // TODO test fails because of product problem that page does not render the CVE id.
    it.skip('has item link to image CVE page from Most common image vulnerabilities widget', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-cves';
        const widgetHeading = 'Most common image vulnerabilities';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForLabelText),
            getHeaderTextFromItemTextWithSlashSeparators
        );
    });

    it('has item link to cluster single page from Clusters with most orchestrator and Istio vulnerabilities', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const widgetHeading = 'Clusters with most orchestrator and Istio vulnerabilities';

        verifyItemLinkToEntityPage(
            entitiesKey,
            getItemTextSelectorForWidget(widgetHeading, itemTextSelectorForClusters),
            (itemText) => itemText
        );
    });
});
