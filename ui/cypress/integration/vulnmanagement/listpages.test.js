import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

const hasExpectedHeaderColumns = colNames => {
    colNames.forEach(col => {
        cy.get(`${selectors.tableColumn}:contains('${col}')`);
    });
};

function validateDataInEntityListPage(entityCountAndName, entityURL) {
    cy.get(selectors.entityRowHeader)
        .invoke('text')
        .then(entityCountFromHeader => {
            if (entityCountAndName.includes('CVE')) {
                const numEntitiesListPage = parseInt(entityCountFromHeader, 10);
                const numEntitiesParentPage = parseInt(entityCountAndName, 10);
                expect(numEntitiesListPage - numEntitiesParentPage).to.be.lessThan(6);
            } else {
                expect(entityCountFromHeader).contains(
                    parseInt(entityCountAndName, 10),
                    `expected entity count ${entityCountAndName} found in the related entity list page`
                );
            }
        });
    cy.visit(entityURL);
}

function validateAllCVELinks(prevUrl) {
    cy.get(`${selectors.allCVEColumnLink}`)
        .eq(0)
        .invoke('text')
        .then(value => {
            cy.get(`${selectors.allCVEColumnLink}`)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            validateDataInEntityListPage(value.toUpperCase(), prevUrl);
        });
}

function validateFixableCVELinks(urlBack) {
    cy.get(`${selectors.fixableCVELink}`)
        .eq(0)
        .invoke('text')
        .then(value => {
            cy.get(`${selectors.fixableCVELink}`)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            if (parseInt(value, 10) === 1)
                validateDataInEntityListPage(`${parseInt(value, 10)} CVE`, urlBack);
            if (parseInt(value, 10) > 1)
                validateDataInEntityListPage(`${parseInt(value, 10)} CVES`, urlBack);
        });
}

function validateSort(selector) {
    let current;
    let prev;
    prev = -1000;
    cy.get(selector).each($el => {
        current = parseInt($el.text(), 10);
        const sortOrderStatus = current >= prev;
        expect(sortOrderStatus).to.equals(true, 'sort order is as expected');
        prev = current;
    });
}

function validateSortForCVE(selector) {
    let current;
    let prev;
    let sortOrderStatus = false;
    prev = 1000;
    cy.get(selector).each($el => {
        current = parseFloat($el.text(), 10.0);
        // eslint-disable-next-line no-restricted-globals
        if (!isNaN(prev) && !isNaN(current)) {
            sortOrderStatus = current <= prev;
            expect(sortOrderStatus).to.equals(true, 'sort order is as expected');
            prev = current;
        }
    });
}

function validateTileLinksInSidePanel(colSelector, col, parentUrl) {
    cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
        .invoke('text')
        .then(value => {
            cy.get(colSelector)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            cy.get(selectors.getTileLink(col.toUpperCase()))
                .find(selectors.tileLinkText)
                .contains(parseInt(value, 10));
            cy.get(selectors.getTileLink(col.toUpperCase()))
                .find(selectors.tileLinkValue)
                .contains(col.toUpperCase());
            cy.visit(parentUrl);
        });
}

function validateCVETileLinksInSidePanel(parentUrl) {
    cy.get(selectors.tableBodyColumn).each($el => {
        const value = $el.text();
        let cveCount = 0;
        if (value.toLowerCase().includes('cve')) cveCount = parseInt(value.split(' ')[0], 10);
        if (cveCount > 0) {
            cy.get(selectors.tableBodyColumn)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            cy.get(selectors.getTileLink('CVE'))
                .find(selectors.tileLinkValue)
                .contains('CVE');
            cy.get(selectors.tileLinkText).contains(cveCount);
            cy.visit(parentUrl);
        }
    });
}

function validateTabsInEntityPage(parentUrl, colSelector, col) {
    cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
        .invoke('text')
        .then(value => {
            cy.get(colSelector)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            cy.get(selectors.sidePanelExpandButton).click({ force: true });
            cy.get(selectors.getSidePanelTabLink(col.toLowerCase())).click({ force: true });
            expect(cy.get(selectors.tabHeader).contains(parseInt(value, 10)));
            cy.wait(3000);
            cy.visit(parentUrl);
        });
}

function validateFixableTabLinksInEntityPage(parentUrl) {
    cy.get(selectors.tableBodyColumn).each($el => {
        const value = $el.text();
        let fixableCount = 0;
        if (value.toLowerCase().includes('fixable')) {
            fixableCount = parseInt(value.split(' ')[2], 10);
        }
        if (fixableCount > 0) {
            cy.get(selectors.tableBodyColumn)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            if (!parentUrl.includes('components')) {
                cy.get(selectors.tabButton)
                    .contains('Fixable CVEs')
                    .click();
            }
            cy.get(selectors.getSidePanelTabHeader('fixable')).contains(fixableCount);
            cy.visit(parentUrl);
        }
    });
}

function validateCVETabsInSidePanel(parentUrl, colSelector, col) {
    cy.get(selectors.tableBodyColumn).each($el => {
        const value = $el.text();
        let cveCount = 0;
        if (value.toLowerCase().includes('cve')) cveCount = parseInt(value.split(' ')[0], 10);
        if (cveCount > 0) {
            cy.get(selectors.tableBodyColumn)
                .eq(0)
                .click({ force: true });
            cy.wait(2000);
            cy.get(selectors.sidePanelExpandButton).click({ force: true });
            cy.get(selectors.getSidePanelTabLink(col.toUpperCase())).click({ force: true });
            expect(cy.get(selectors.tabHeader).contains(cveCount));
            cy.wait(2000);
            cy.visit(parentUrl);
        }
    });
}

function validateLinksInListPage(col, parentUrl) {
    cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
        .invoke('text')
        .then(value => {
            cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`).click({
                force: true
            });
            validateDataInEntityListPage(value, parentUrl);
        });
}

function allChecksForEntities(parentUrl, entity) {
    validateLinksInListPage(entity, parentUrl);
    validateTileLinksInSidePanel(selectors.tableBodyColumn, entity, parentUrl);
    validateTabsInEntityPage(parentUrl, selectors.tableBodyColumn, entity);
}

function allCVECheck(parentUrl) {
    validateCVETileLinksInSidePanel(parentUrl);
    validateCVETabsInSidePanel(parentUrl, selectors.tableBodyColumn, 'CVEs');
    validateAllCVELinks(parentUrl);
}

function allFixableCheck(parentUrl) {
    validateFixableCVELinks(parentUrl);
    validateFixableTabLinksInEntityPage(parentUrl);
}

describe('Entities list Page', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();

    it('should display all the columns and links expected in clusters list page', () => {
        cy.visit(url.list.clusters);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'K8S Version',
            'Namespaces',
            'Deployments',
            'Policies',
            'Policy Status',
            'Latest Violation',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (!columnValue.includes('no') && columnValue.includes('polic'))
                allChecksForEntities(url.list.clusters, 'policies');
            if (columnValue !== 'no namespaces' && columnValue.includes('namespace'))
                allChecksForEntities(url.list.clusters, 'namespaces');

            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.clusters, 'deployments');
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.clusters);
            if (columnValue.includes('fixable')) allFixableCheck(url.list.clusters);
        });
    });

    it('should display all the columns and links expected in namespaces list page', () => {
        cy.visit(url.list.namespaces);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'Images',
            'Namespace',
            'Deployments',
            'Policies',
            'Policy Status',
            'Latest Violation',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (!columnValue.includes('no') && columnValue.includes('polic'))
                allChecksForEntities(url.list.namespaces, 'polic');
            if (columnValue !== 'no images' && columnValue.includes('image'))
                allChecksForEntities(url.list.namespaces, 'image');
            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.namespaces, 'deployment');
            if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.namespaces);
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.namespaces);
        });
        validateSort(selectors.riskScoreCol);
    });

    it('should display all the columns and links expected in deployments list page', () => {
        cy.visit(url.list.deployments);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'Images',
            'Namespace',
            'Deployment',
            'Policies',
            'Policy Status',
            'Latest Violation',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no failing policies' && columnValue.includes('polic'))
                allChecksForEntities(url.list.deployments, 'Polic');
            if (columnValue !== 'no images' && columnValue.includes('image'))
                allChecksForEntities(url.list.deployments, 'image');
            if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.deployments);
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.deployments);
        });
        validateSort(selectors.riskScoreCol);
    });

    it('should display all the columns and links expected in images list page', () => {
        cy.visit(url.list.images);
        hasExpectedHeaderColumns([
            'Image',
            'CVEs',
            'Top CVSS',
            'Created',
            'Scan Time',
            'Image Status',
            'Deployments',
            'Components',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no deployments' && columnValue.includes('Deployment'))
                allChecksForEntities(url.list.images, 'deployment');
            if (columnValue !== 'no components' && columnValue.includes('Component'))
                allChecksForEntities(url.list.images, 'component');
            if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.images);
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.images);
        });
        validateSort(selectors.riskScoreCol);
    });

    it('should display all the columns expected in components list page', () => {
        cy.visit(url.list.components);
        hasExpectedHeaderColumns([
            'Component',
            'CVEs',
            'Top CVSS',
            'Images',
            'Deployments',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.components, 'Deployment');
            if (columnValue !== 'no images' && columnValue.includes('image'))
                allChecksForEntities(url.list.components, 'Image');
            if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.components);
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.components);
        });
        validateSort(selectors.componentsRiskScoreCol);
    });

    it('should display all the columns and links expected in cves list page', () => {
        cy.visit(url.list.cves);
        hasExpectedHeaderColumns([
            'CVE',
            'Fixable',
            'CVSS Score',
            'Env. Impact',
            'Impact Score',
            'Scanned',
            'Published',
            'Deployments'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.cves, 'Deployment');
            if (columnValue !== 'no images' && columnValue.includes('image'))
                allChecksForEntities(url.list.cves, 'image');
            if (columnValue !== 'no components' && columnValue.includes('component'))
                allChecksForEntities(url.list.cves, 'component');
        });

        validateSortForCVE(selectors.cvesCvssScoreCol);
    });
});
