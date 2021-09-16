import capitalize from 'lodash/capitalize';
import pluralize from 'pluralize';

import * as api from '../constants/apiEndpoints';
import { url, selectors as configManagementSelectors } from '../constants/ConfigManagementPage';

// specifying an "entityName" will try to select that row in the table
export const renderListAndSidePanel = (entity, entityName = null) => {
    cy.intercept('POST', api.graphqlPluralEntity(entity)).as('entities');
    cy.intercept('POST', api.graphqlSingularEntity(pluralize.singular(entity))).as('getEntity');
    cy.visit(url.list[entity]);
    cy.wait('@entities');
    cy.get(`${configManagementSelectors.tableRows}${entityName ? `:contains(${entityName})` : ''}`)
        .not(configManagementSelectors.disabledTableRows)
        .find(configManagementSelectors.tableCells)
        .eq(1)
        .click({ force: true });
    cy.wait('@getEntity');
    cy.get(configManagementSelectors.widgets);
};

export const navigateToSingleEntityPage = (entity) => {
    cy.intercept('POST', api.graphqlSingularEntity(entity)).as('getEntity');
    cy.get(configManagementSelectors.externalLink).click();
    cy.wait('@getEntity');
    cy.location('pathname').should('contain', url.single[entity]);
};

export const hasCountWidgetsFor = (entities) => {
    entities.forEach((entity) => {
        cy.get(`${configManagementSelectors.countWidgetTitle}:contains('${entity}')`);
    });
};

export const clickOnCountWidget = (entity, type) => {
    // TODO add another argument to intercept getEntity_SUBENTITY query
    cy.get(`${configManagementSelectors.countWidgets}:contains('${capitalize(entity)}')`)
        .find(configManagementSelectors.countWidgetValue)

        .click({ force: true });

    if (type === 'side-panel') {
        cy.get(
            `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${entity}")`
        );
    }

    if (type === 'entityList') {
        cy.get(`${configManagementSelectors.groupedTabs}:contains('${entity}')`);
        cy.get(`li.bg-base-100:contains("${entity}")`);
    }
};

export const clickOnEntityWidget = (entity, type) => {
    cy.intercept('POST', api.graphqlSingularEntity(entity)).as('getEntity');
    cy.get(`${configManagementSelectors.relatedEntityWidgets}:contains('${capitalize(entity)}')`)
        .find(configManagementSelectors.relatedEntityWidgetValue)
        .invoke('text')
        .then((value) => {
            cy.get(
                `${configManagementSelectors.relatedEntityWidgets}:contains('${capitalize(
                    entity
                )}')`
            ).click();
            cy.wait('@getEntity');
            if (type === 'side-panel') {
                cy.get(
                    `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${value}")`
                );
            }
        });
};

export const clickOnRowEntity = (entity, subEntity, isNotCapitalized) => {
    cy.intercept('POST', api.graphqlPluralEntity(entity)).as('entities');
    cy.visit(url.list[entity]);
    cy.wait('@entities');
    cy.get(configManagementSelectors.tableRows)
        .find(
            `${configManagementSelectors.tableCells} a:contains('${
                isNotCapitalized ? subEntity : capitalize(subEntity)
            }')`
        )
        .eq(0)
        .click({ force: true });
    // TODO wait on entity_SUBENTITY request

    cy.get(
        `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${subEntity.toLowerCase()}")`
    );
};

export const clickOnSingleEntity = (entity, subEntity) => {
    cy.intercept('POST', api.graphqlPluralEntity(entity)).as('entities');
    cy.intercept('POST', api.graphqlSingularEntity(subEntity)).as('getEntity');
    cy.visit(url.list[entity]);
    cy.wait('@entities');
    cy.get(configManagementSelectors.tableRows)
        .find(`${configManagementSelectors.tableCells} a[href*='/${subEntity}']`)
        .eq(0)
        .invoke('text')
        .then((value) => {
            cy.get(configManagementSelectors.tableRows)
                .find(`${configManagementSelectors.tableCells} a[href*='/${subEntity}']`)
                .eq(0)
                .click({ force: true });
            cy.wait('@getEntity');
            cy.get(
                `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${value}")`
            );
        });
};

export const hasTabsFor = (entities) => {
    entities.forEach((entity) => {
        cy.get(`${configManagementSelectors.groupedTabs} div:contains("${entity}")`);
    });
};

export const hasRelatedEntityFor = (entity) => {
    cy.get(`${configManagementSelectors.relatedEntityWidgetTitle}:contains('${entity}')`);
};

const entityCountMatchesTableRows = (listEntity, context) => {
    // TODO add another argument to intercept getEntity_SUBENTITY query
    const contextSelector = `[data-testid="${context === 'Page' ? 'panel' : 'side-panel'}"]`;
    cy.get(`${configManagementSelectors.countWidgets}:contains('${listEntity}')`)
        .find(configManagementSelectors.countWidgetValue)
        .invoke('text')
        .then((count) => {
            if (count === '0') {
                return;
            }
            cy.get(`${configManagementSelectors.countWidgets}:contains('${listEntity}')`)
                .find('button')
                .invoke('attr', 'disabled', false)
                .click();
            cy.wait(2000);
            cy.get(`${contextSelector} .rt-tr-group`);
            cy.get(`${contextSelector} [data-testid="panel-header"]`)
                .invoke('text')
                .then((panelHeaderText) => {
                    expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(count, 10));
                });
        });
};

export const pageEntityCountMatchesTableRows = (listEntity) => {
    entityCountMatchesTableRows(listEntity, 'Page');
};

export const sidePanelEntityCountMatchesTableRows = (listEntity) => {
    entityCountMatchesTableRows(listEntity, 'Side Panel');
};

export const entityListCountMatchesTableLinkCount = (entities1, entities2) => {
    cy.intercept('POST', api.graphqlPluralEntity(entities1)).as('entities');
    cy.visit(url.list[entities1]);
    cy.wait('@entities');
    cy.get(configManagementSelectors.tableLinks)
        .contains(entities2) // return first match
        .then(($a) => {
            const linkText = $a.text();
            cy.wrap($a).click();
            // TODO wait on entity_SUBENTITY request
            cy.get('[data-testid="side-panel"] [data-testid="panel-header"]')
                .invoke('text')
                .should((panelHeaderText) => {
                    // Expect leading numeric digits to match, however entire text might not match;
                    // for example, '10 Users & Groups' versus '10 users and groups'
                    expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(linkText, 10));
                });
        });
};
