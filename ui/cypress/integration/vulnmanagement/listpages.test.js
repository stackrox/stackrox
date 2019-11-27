import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';

const hasExpectedHeaderColumns = colNames => {
    colNames.forEach(col => {
        cy.get(`${selectors.tableColumn}:contains('${col}')`);
    });
};

function validateDataInEntityListPage(entityCountAndName, entityURL) {
    cy.get(selectors.entityRowHeader)
        .invoke('text')
        .then(entityCountFromHeader => {
            expect(entityCountFromHeader).contains(
                entityCountAndName,
                `expected entity count ${entityCountAndName} found in the related entity list page`
            );
        });
    cy.visit(entityURL);
}
function validateClickableLinks(colLinks, parentUrl) {
    colLinks.forEach(col => {
        if (col !== 'Policies') {
            cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
                .invoke('text')
                .then(value => {
                    cy.get(
                        `${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`
                    ).click();
                    validateDataInEntityListPage(value, parentUrl);
                });
        }
        if (col === 'Policies') {
            cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
                .invoke('text')
                .then(value => {
                    expect(value).contains('policies' || 'policy', 'expected text displayed');
                    cy.get(`${selectors.tableColumnLinks}:contains('${value}')`).click();
                    validateDataInEntityListPage(value, parentUrl);
                });
        }
    });
}

function validateClickableLinksEntityListPage(colLinks, parentUrl) {
    colLinks.forEach(col => {
        if (col !== 'Policies') {
            cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
                .eq(0)
                .invoke('text')
                .then(value => {
                    cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
                        .eq(0)
                        .click({ force: true });
                    validateDataInEntityListPage(value, parentUrl);
                });
        }
        if (col === 'Policies') {
            cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
                .eq(0)
                .invoke('text')
                .then(value => {
                    expect(value).contains('policies' || 'policy', 'expected text displayed');
                    cy.get(`${selectors.tableColumnLinks}:contains('${value}')`)
                        .eq(0)
                        .click({ force: true });
                    validateDataInEntityListPage(`${parseInt(value, 10)} policies`, parentUrl);
                });
        }
    });
}

function validateAllCVELinks(prevUrl) {
    cy.get(`${selectors.allCVEColumnLink}`)
        .eq(0)
        .invoke('text')
        .then(value => {
            cy.get(`${selectors.allCVEColumnLink}`)
                .eq(0)
                .click({ force: true });
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

describe('Entities list Page', () => {
    withAuth();
    it.skip('should display all the columns and links expected in clusters list page', () => {
        cy.visit(url.list.clusters);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'K8S version',
            'Namespaces',
            'Deployments',
            'Policies',
            'Policy status',
            'Latest violation',
            'Risk Priority'
        ]);
        validateClickableLinks(['Namespace', 'Deployment', 'Policies'], url.list.clusters);
        validateAllCVELinks(url.list.clusters);
        validateFixableCVELinks(url.list.clusters);
        validateSort(selectors.riskScoreCol);
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
            'Policy status',
            'Latest violation',
            'Risk Priority'
        ]);
        validateClickableLinksEntityListPage(['image', 'deployment'], url.list.namespaces);
        validateAllCVELinks(url.list.namespaces);
        validateFixableCVELinks(url.list.namespaces);
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
            'Latest violation',
            'Risk Priority'
        ]);
        validateClickableLinksEntityListPage(['image', 'Policies'], url.list.deployments);
        validateAllCVELinks(url.list.deployments);
        validateFixableCVELinks(url.list.deployments);
        validateSort(selectors.riskScoreCol);
    });

    it('should display all the columns and links expected in images list page', () => {
        cy.visit(url.list.images);
        hasExpectedHeaderColumns([
            'Image',
            'CVEs',
            'Top CVSS',
            'Created',
            'Scan time',
            'Image Status',
            'Deployments',
            'Components',
            'Risk Priority'
        ]);
        validateClickableLinksEntityListPage(['deployment', 'component'], url.list.images);
        validateAllCVELinks(url.list.images);
        validateFixableCVELinks(url.list.images);
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
        validateClickableLinksEntityListPage(['deployment', 'image'], url.list.components);
        validateAllCVELinks(url.list.components);
        validateFixableCVELinks(url.list.components);
        validateSort(selectors.componentsRiskScoreCol);
    });

    it('should display all the columns and links  expected in cves list page', () => {
        cy.visit(url.list.cves);
        hasExpectedHeaderColumns([
            'CVE',
            'Fixable',
            'CVSS score',
            'Env. Impact',
            'Impact score',
            'Scanned',
            'Published',
            'Deployments'
        ]);
        validateClickableLinksEntityListPage(['image', 'deployment', 'component'], url.list.cves);
        validateSortForCVE(selectors.cvesCvssScoreCol);
    });
});
