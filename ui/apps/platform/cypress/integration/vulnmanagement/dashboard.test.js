import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import {
    interactAndWaitForVulnerabilityManagementEntities,
    verifyVulnerabilityManagementDashboardCVEs,
    visitVulnerabilityManagementDashboard,
    visitVulnerabilityManagementDashboardFromLeftNav,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

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
    return `${selectors.getWidget(widgetHeading)} a:contains("View all")`;
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

    it('should navigate from menu item for Image CVEs to entities list', () => {
        verifyVulnerabilityManagementDashboardCVEs('image-cves', /^\d+ Image CVEs?$/);
    });

    it('should navigate from menu item for Node CVEs to entities list', () => {
        verifyVulnerabilityManagementDashboardCVEs('node-cves', /^\d+ Node CVEs?$/);
    });

    it('should navigate from menu item Cluster (Platform) CVEs to entities list', () => {
        verifyVulnerabilityManagementDashboardCVEs('cluster-cves', /^\d+ Platform CVEs?$/);
    });

    it('should navigate from images link to images list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get('[data-testid="page-header"] a')
                .contains('[data-testid="tile-link-value"]', /^\d+ images?/)
                .click();
        }, entitiesKey);

        cy.get('[data-testid="panel"]').contains('[data-testid="panel-header"]', /^\d+ images?/);
    });

    it('should navigate to the clusters list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const menuListItemText = 'clusters'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should navigate to the namespaces list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'namespaces';
        const menuListItemText = 'namespaces'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should navigate to the deployments list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'deployments';
        const menuListItemText = 'deployments'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should navigate to the node components list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'node-components';
        const menuListItemText = 'node components'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should navigate to the image components list', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-components';
        const menuListItemText = 'image components'; // lowercase because of Tailwind capitalize class

        verifyVulnerabilityManagementDashboardApplicationAndInfrastructure(
            entitiesKey,
            menuListItemText
        );
    });

    it('should go to images list from View all link of Top riskiest images', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const widgetHeading = 'Top riskiest images';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should(
            'eq',
            '?sort[0][id]=Image%20Risk%20Priority&sort[0][desc]=false'
        );
    });

    it('should go to image-cves list from View all link of Recently detected image vulnerabilities', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-cves';
        const widgetHeading = 'Recently detected image vulnerabilities';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should('eq', '?sort[0][id]=CVE%20Created%20Time&sort[0][desc]=true');
    });

    it('should to to image-cves list from View all link of Most common image vulnerabilities', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'image-cves';
        const widgetHeading = 'Most common image vulnerabilities';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should(
            'eq',
            '?sort[0][id]=Deployment%20Count&sort[0][desc]=true&sort[1][id]=CVSS&sort[1][desc]=true'
        );
    });

    it('should go to clusters list from View all link of Clusters with most orchestrator and Istio vulnerabilities', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'clusters';
        const widgetHeading = 'Clusters with most orchestrator and Istio vulnerabilities';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);

        cy.location('search').should('eq', '');
    });

    it('should to to deployments list from View all link of Top risky deployments by CVE count and CVSS score', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'deployments';
        const widgetHeading = 'Top risky deployments by CVE count and CVSS score';

        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });

    it('should go to namespaces list from View all link of Top risky namespaces by CVE count and CVSS score', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'namespaces';
        const widgetHeading = 'Top risky namespaces by CVE count and CVSS score';

        selectTopRiskyOption(widgetHeading);
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });

    it('should go to images list from View all link of Top risky images by CVE count and CVSS score', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'images';
        const widgetHeading = 'Top risky images by CVE count and CVSS score';

        selectTopRiskyOption(widgetHeading);
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });

    it('should go to nodes list from View all link of Top risky images by CVE count and CVSS score', () => {
        visitVulnerabilityManagementDashboard();

        const entitiesKey = 'nodes';
        const widgetHeading = 'Top risky nodes by CVE count and CVSS score';

        selectTopRiskyOption(widgetHeading);
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(getViewAllSelectorForWidget(widgetHeading)).click();
        }, entitiesKey);
    });
});
