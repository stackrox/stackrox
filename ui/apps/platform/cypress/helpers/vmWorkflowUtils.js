import { selectors as vulnManagementSelectors } from '../constants/VulnManagementPage';

const idColumnOffset = 1;
export const hasExpectedHeaderColumns = (colNames, additionalColumnsToSkip = 0) => {
    colNames.forEach((col, idx) => {
        cy.get(
            `.rt-th.leading-normal:nth-child(${
                idx + idColumnOffset + additionalColumnsToSkip + 1
            }) > div:contains('${col}')`
        );
    });
};

function validateDataInEntityListPage(entityCountAndName, entityURL) {
    cy.get(vulnManagementSelectors.entityRowHeader)
        .invoke('text')
        .then((entityCountFromHeader) => {
            if (entityCountAndName.includes('CVE') && !entityCountAndName.includes('0')) {
                const numEntitiesListPage = parseInt(entityCountFromHeader, 10);
                const numEntitiesParentPage = parseInt(entityCountAndName, 10);
                expect(numEntitiesListPage).to.be.greaterThan(0);
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

function validateLinksInListPage(col, parentUrl) {
    cy.get(`${vulnManagementSelectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
        .invoke('text')
        .then((value) => {
            cy.get(`${vulnManagementSelectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
                .first()
                .click({
                    force: true,
                });
            validateDataInEntityListPage(value, parentUrl);
        });
}

function validateTileLinksInSidePanel(colSelector, col, parentUrl) {
    cy.get(`${vulnManagementSelectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
        .invoke('text')
        .then((value) => {
            cy.get(colSelector).eq(0).click({ force: true });
            let entitySelector;
            const col1 = col.toLowerCase();
            cy.log(`col1: ${col1}`);
            cy.log(`col1.includes('node'): ${col1.includes('node')}`);
            if (col1.includes('image')) {
                entitySelector = vulnManagementSelectors.imageTileLink;
            } else if (col1.includes('deployment')) {
                entitySelector = vulnManagementSelectors.deploymentTileLink;
            } else if (col1.includes('namespace')) {
                entitySelector = vulnManagementSelectors.namespaceTileLink;
            } else if (col1.includes('component')) {
                entitySelector = vulnManagementSelectors.componentTileLink;
            } else if (col1.includes('cve')) {
                entitySelector = vulnManagementSelectors.cveTileLink;
            } else if (col1.includes('node')) {
                entitySelector = vulnManagementSelectors.nodeTileLink;
            } else {
                entitySelector = vulnManagementSelectors.getTileLink(col.toUpperCase());
            }
            cy.log(`entitySelector: ${entitySelector}`);
            cy.get(entitySelector)
                .find(vulnManagementSelectors.tileLinkText)
                .contains(parseInt(value, 10));
            cy.get(entitySelector)
                .find(vulnManagementSelectors.tileLinkValue)
                .contains(col.toUpperCase());
            cy.visit(parentUrl);
        });
}

function validateTabsInEntityPage(parentUrl, colSelector, col) {
    cy.get(`${vulnManagementSelectors.tableColumnLinks}:contains('${col.toLowerCase()}')`)
        .invoke('text')
        .then((value) => {
            cy.get(colSelector).eq(0).click({ force: true });
            cy.get(vulnManagementSelectors.sidePanelExpandButton).click({ force: true });
            cy.get(vulnManagementSelectors.getSidePanelTabLink(col.toLowerCase())).click({
                force: true,
            });
            expect(cy.get(vulnManagementSelectors.tabHeader).contains(parseInt(value, 10)));
            cy.visit(parentUrl);
        });
}

function validateCVETileLinksInSidePanel(parentUrl) {
    cy.get(vulnManagementSelectors.tableBodyColumn).each(($el) => {
        const value = $el.text();
        let cveCount = 0;
        if (value.toLowerCase().includes('cve')) {
            cveCount = parseInt(value.split(' ')[0], 10);
        }
        if (cveCount > 0) {
            cy.get(vulnManagementSelectors.tableBodyColumn).eq(0).click({ force: true });
            // not reusing a predefined selector below, because we had made this function so DRY,
            //   we created a false positive that was labyrinthine to track down
            //   see: https://engblog.nextdoor.com/how-to-dry-up-your-tests-without-making-mummies-of-them-7de79a8e3df1
            cy.get('[data-testid="CVE-tile-link"]:contains("CVE")')
                .find(vulnManagementSelectors.tileLinkValue)
                .contains('CVE');
            cy.get(vulnManagementSelectors.tileLinkText).contains(cveCount);
            cy.visit(parentUrl);
        }
    });
}
function validateAllCVELinks(prevUrl) {
    cy.get(`${vulnManagementSelectors.allCVEColumnLink}`)
        .eq(0)
        .invoke('text')
        .then((value) => {
            cy.get(`${vulnManagementSelectors.allCVEColumnLink}`).eq(0).click({ force: true });
            validateDataInEntityListPage(value.toUpperCase(), prevUrl);
        });
}

function validateFixableCVELinks(urlBack) {
    cy.get(`${vulnManagementSelectors.fixableCVELink}`)
        .eq(0)
        .invoke('text')
        .then((value) => {
            cy.get(`${vulnManagementSelectors.fixableCVELink}`).eq(0).click({ force: true });
            if (parseInt(value, 10) === 1) {
                validateDataInEntityListPage(`${parseInt(value, 10)} CVE`, urlBack);
            }
            if (parseInt(value, 10) > 1) {
                validateDataInEntityListPage(`${parseInt(value, 10)} CVES`, urlBack);
            }
        });
}

function validateCVETabsInSidePanel(parentUrl, colSelector, col) {
    cy.get(vulnManagementSelectors.tableBodyColumn).each(($el) => {
        const value = $el.text();
        let cveCount = 0;
        if (value.toLowerCase().includes('cve')) {
            cveCount = parseInt(value.split(' ')[0], 10);
        }
        if (cveCount > 0) {
            cy.get(vulnManagementSelectors.tableBodyColumn).eq(0).click({ force: true });
            cy.get(vulnManagementSelectors.sidePanelExpandButton).click({
                force: true,
                timeout: 6000,
            });
            cy.get(vulnManagementSelectors.getSidePanelTabLink(col.toUpperCase())).click({
                force: true,
            });
            expect(cy.get(vulnManagementSelectors.tabHeader).contains(cveCount));
            cy.visit(parentUrl);
        }
    });
}

function validateFixableTabLinksInEntityPage(parentUrl) {
    cy.get(vulnManagementSelectors.tableBodyColumn).each(($el) => {
        const value = $el.text();
        let fixableCount = 0;
        if (value.toLowerCase().includes('fixable')) {
            fixableCount = parseInt(value.split(' ')[2], 10);
        }
        if (fixableCount > 0) {
            cy.get(vulnManagementSelectors.tableBodyColumn).eq(0).click({ force: true });
            if (!parentUrl.includes('components')) {
                cy.get(vulnManagementSelectors.tabButton, { timeout: 6000 })
                    .contains('Fixable CVEs')
                    .click();
            }
            cy.get(vulnManagementSelectors.getSidePanelTabHeader('fixable')).contains(fixableCount);
            cy.visit(parentUrl);
        }
    });
}
// below commented functions will be enabled once back end sorting starts working
/* export const =  validateSort = selector => {
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

// uncomment this function when we uncomment this part of the test in ui/cypress/integration/vulnmanagement/cvesListPages.test.js
export const validateSortForCVE = selector => {
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
};
*/

export const allChecksForEntities = (parentUrl, entity) => {
    validateLinksInListPage(entity, parentUrl);
    validateTileLinksInSidePanel(vulnManagementSelectors.tableBodyColumn, entity, parentUrl);
    validateTabsInEntityPage(parentUrl, vulnManagementSelectors.tableBodyColumn, entity);
};

export const allCVECheck = (parentUrl) => {
    validateCVETileLinksInSidePanel(parentUrl);
    validateCVETabsInSidePanel(parentUrl, vulnManagementSelectors.tableBodyColumn, 'CVEs');
    validateAllCVELinks(parentUrl);
};

export const allFixableCheck = (parentUrl) => {
    validateFixableCVELinks(parentUrl);
    validateFixableTabLinksInEntityPage(parentUrl);
};
