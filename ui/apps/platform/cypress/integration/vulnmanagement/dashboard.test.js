import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import {
    interactAndWaitForVulnerabilityManagementEntities,
    visitVulnerabilityManagementDashboard,
    visitVulnerabilityManagementDashboardFromLeftNav,
} from '../../helpers/vulnmanagement/entities';
import { hasFeatureFlag } from '../../helpers/features';

function verifyVulnerabilityManagementDashboardCVEs(entitiesKey, menuListItemRegExp) {
    visitVulnerabilityManagementDashboard();

    // Selector contains singular noun to match 1 CVE.
    const menuButtonSelector = `button[data-testid="menu-button"]:contains("CVE")`;
    const menuListItemSelector = `${menuButtonSelector} + div[data-testid="menu-list"]`;

    cy.get(menuButtonSelector).click(); // open menu list
    cy.get(menuListItemSelector)
        .contains('a', menuListItemRegExp)
        .then(($a) => {
            const linkText = $a.text();
            const panelHeaderText = linkText.replace(/s$/, 'S'); // TODO fix UI inconsistency

            interactAndWaitForVulnerabilityManagementEntities(() => {
                cy.wrap($a).click(); // visit entities list
            }, entitiesKey);

            cy.get(`[data-testid="panel-header"]:contains(${panelHeaderText})`);
        });
}

function verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
    entitiesKey,
    menuListItemText
) {
    const menuButtonSelector = `button[data-testid="menu-button"]:contains("Application & Infrastructure")`;
    const menuListItemSelector = `${menuButtonSelector} + div a:contains("${menuListItemText}")`;

    cy.get(menuButtonSelector).click(); // open menu list
    interactAndWaitForVulnerabilityManagementEntities(() => {
        cy.get(menuListItemSelector).click(); // visit entities list
    }, entitiesKey);
}

function getViewAllSelectorForWidget(widgetHeading) {
    return `${selectors.getWidget(widgetHeading)} ${selectors.viewAllButton}`;
}

function selectTopRiskyOption(optionText) {
    cy.get('[data-testid="widget"]:contains("Top risky") .react-select__control').click();
    cy.get(
        `[data-testid="widget"]:contains("Top risky") .react-select__option:contains("${optionText}")`
    ).click();
}

describe('Vulnerability Management Dashboard', () => {
    withAuth();

    it('should visit using the left nav', () => {
        visitVulnerabilityManagementDashboardFromLeftNav();
    });

    it('should have title', () => {
        visitVulnerabilityManagementDashboard();

        cy.title().should(
            'match',
            getRegExpForTitleWithBranding('Vulnerability Management - Dashboard')
        );
    });

    it('should show same number of policies between the tile and the policies list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'policies';
        const tileLinkSelector = `${selectors.tileLinks}:eq(0)`;
        cy.get(`${tileLinkSelector} ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                interactAndWaitForVulnerabilityManagementEntities(() => {
                    cy.get(tileLinkSelector).click();
                }, entitiesKey);

                cy.get(`[data-testid="panel"] [data-testid="panel-header"]:contains("${value}")`);
            });
    });

    it('should show same number of Image CVEs in menu item and entities list', function () {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }

        verifyVulnerabilityManagementDashboardCVEs('image-cves', /^\d+ Image CVEs?$/);
    });

    it('should show same number of Node CVEs in menu item and entities list', function () {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }

        verifyVulnerabilityManagementDashboardCVEs('node-cves', /^\d+ Node CVEs?$/);
    });

    it('should show same number of Cluster (Platform) CVEs in menu item and entities list', function () {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }

        verifyVulnerabilityManagementDashboardCVEs('cluster-cves', /^\d+ Platform CVEs?$/);
    });

    it('should show same number of images between the tile and the images list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const tileToCheck = hasFeatureFlag('ROX_POSTGRES_DATASTORE') ? 2 : 3;
        cy.get(`${selectors.tileLinks}:eq(${tileToCheck}) ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                interactAndWaitForVulnerabilityManagementEntities(() => {
                    cy.get(`${selectors.tileLinks}:eq(${tileToCheck})`).click();
                }, entitiesKey);

                cy.get(`[data-testid="panel"] [data-testid="panel-header"]:contains("${value}")`);
            });
    });

    it('should properly navigate to the clusters list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const menuListItemText = 'clusters'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should properly navigate to the namespaces list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'namespaces';
        const menuListItemText = 'namespaces'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should properly navigate to the deployments list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'deployments';
        const menuListItemText = 'deployments'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should navigate to the node components list', function () {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }

        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'node-components';
        const menuListItemText = 'node components'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should navigate to the image components list', function () {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }

        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-components';
        const menuListItemText = 'image components'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('clicking the "Top Riskiest Images" widget\'s "View All" button should take you to the images list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const widgetHeading = 'Top Riskiest Images';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should(
            'eq',
            '?sort[0][id]=Image%20Risk%20Priority&sort[0][desc]=false'
        );
    });

    it('clicking the "Frequently Violated Policies" widget\'s "View All" button should take you to the policies list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'policies';
        const widgetHeading = 'Frequently Violated Policies';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should('eq', '?sort[0][id]=Severity&sort[0][desc]=true');
    });

    it('clicking the "Recently Detected Image Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = hasFeatureFlag('ROX_POSTGRES_DATASTORE') ? 'image-cves' : 'cves';
        const widgetHeading = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'Recently Detected Image Vulnerabilities'
            : 'Recently Detected Vulnerabilities';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should('eq', '?sort[0][id]=CVE%20Created%20Time&sort[0][desc]=true');
    });

    it('clicking the "Most Common Image Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = hasFeatureFlag('ROX_POSTGRES_DATASTORE') ? 'image-cves' : 'cves';
        const widgetHeading = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'Most Common Image Vulnerabilities'
            : 'Most Common Vulnerabilities';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should(
            'eq',
            '?sort[0][id]=Deployment%20Count&sort[0][desc]=true&sort[1][id]=CVSS&sort[1][desc]=true'
        );
    });

    it('clicking the "Clusters With Most Orchestrator & Istio Vulnerabilities" widget\'s "View All" button should take you to the clusters list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const widgetHeading = 'Clusters With Most Orchestrator & Istio Vulnerabilities';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should('eq', '');
    });

    it('clicking the "Top risky deployments by CVE count & CVSS score" widget\'s "View All" button should take you to the deployments list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'deployments';
        const widgetHeading = 'Top risky deployments by CVE count & CVSS score';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });

    it('clicking the "Top risky namespaces by CVE count & CVSS score" widget\'s "View All" button should take you to the namespaces list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'namespaces';
        const widgetHeading = 'Top risky namespaces by CVE count & CVSS score';

        selectTopRiskyOption(widgetHeading);
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });

    it('clicking the "Top risky images by CVE count & CVSS score" widget\'s "View All" button should take you to the images list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const widgetHeading = 'Top risky images by CVE count & CVSS score';

        selectTopRiskyOption(widgetHeading);
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });

    it('clicking the "Top risky images by CVE count & CVSS score" widget\'s "View All" button should take you to the nodes list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'nodes';
        const widgetHeading = 'Top risky nodes by CVE count & CVSS score';

        selectTopRiskyOption(widgetHeading);
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });
});
