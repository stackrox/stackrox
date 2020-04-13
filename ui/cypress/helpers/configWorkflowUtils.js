import capitalize from 'lodash/capitalize';
import { url, selectors as configManagementSelectors } from '../constants/ConfigManagementPage';

// specifying an "entityName" will try to select that row in the table
export const renderListAndSidePanel = (entity, entityName = null) => {
    cy.visit(url.list[entity]);
    cy.wait(1000);
    cy.get(`${configManagementSelectors.tableRows}${entityName ? `:contains(${entityName})` : ''}`)
        .not(configManagementSelectors.disabledTableRows)
        .find(configManagementSelectors.tableCells)
        .eq(1)
        .click({ force: true });
    cy.wait(500);
    cy.get(configManagementSelectors.widgets, { timeout: 7000 });
};

export const navigateToSingleEntityPage = entity => {
    cy.get(configManagementSelectors.externalLink).click();
    cy.url().should('contain', url.single[entity]);
};

export const hasCountWidgetsFor = entities => {
    entities.forEach(entity => {
        cy.get(`${configManagementSelectors.countWidgetTitle}:contains('${entity}')`);
    });
};

export const clickOnCountWidget = (entity, type) => {
    cy.get(`${configManagementSelectors.countWidgets}:contains('${capitalize(entity)}')`)
        .find(configManagementSelectors.countWidgetValue)

        .click({ force: true });

    if (type === 'side-panel') {
        cy.get('[data-testid="side-panel"]')
            .find('[data-testid="breadcrumb-link-text"]')
            .contains(entity);
    }

    if (type === 'entityList') {
        cy.get(`${configManagementSelectors.groupedTabs}:contains('${entity}')`);
        cy.get('li.bg-base-100').contains(entity);
    }
};

export const clickOnEntityWidget = (entity, type) => {
    cy.get(`${configManagementSelectors.relatedEntityWidgets}:contains('${capitalize(entity)}')`)
        .find(configManagementSelectors.relatedEntityWidgetValue)
        .invoke('text')
        .then(value => {
            cy.get(
                `${configManagementSelectors.relatedEntityWidgets}:contains('${capitalize(
                    entity
                )}')`
            ).click();
            cy.wait(500); // it takes time to load the page
            if (type === 'side-panel') {
                cy.get('[data-testid="side-panel"]')
                    .find('[data-testid="breadcrumb-link-text"]')
                    .contains(value);
            }
        });
};

export const clickOnRowEntity = (entity, subEntity, isNotCapitalized) => {
    cy.visit(url.list[entity]);
    cy.get(configManagementSelectors.tableRows)
        .find(
            `${configManagementSelectors.tableCells} a:contains('${
                isNotCapitalized ? subEntity : capitalize(subEntity)
            }')`
        )
        .eq(0)
        .click({ force: true });

    cy.get('[data-testid="side-panel"]')
        .find('[data-testid="breadcrumb-link-text"]')
        .contains(subEntity.toLowerCase());
};

export const clickOnSingleEntity = (entity, subEntity) => {
    cy.visit(url.list[entity]);
    cy.get(configManagementSelectors.tableRows)
        .find(`${configManagementSelectors.tableCells} a[href*='/${subEntity}']`)
        .eq(0)
        .invoke('text')
        .then(value => {
            cy.get(configManagementSelectors.tableRows)
                .find(`${configManagementSelectors.tableCells} a[href*='/${subEntity}']`)
                .eq(0)
                .click({ force: true });
            cy.wait(500); // it takes time to load the page
            cy.get('[data-testid="side-panel"]')
                .find('[data-testid="breadcrumb-link-text"]')
                .contains(value);
        });
};

export const hasTabsFor = entities => {
    entities.forEach(entity => {
        cy.get(configManagementSelectors.groupedTabs)
            .find('div')
            .contains(entity);
    });
};

export const hasRelatedEntityFor = entity => {
    cy.get(`${configManagementSelectors.relatedEntityWidgetTitle}:contains('${entity}')`);
};

const entityCountMatchesTableRows = (listEntity, context) => {
    const contextSelector = `[data-testid="${context === 'Page' ? 'panel' : 'side-panel'}"]`;
    cy.get(`${configManagementSelectors.countWidgets}:contains('${listEntity}')`)
        .find(configManagementSelectors.countWidgetValue)
        .invoke('text')
        .then(count => {
            if (count === '0') return;
            cy.get(`${configManagementSelectors.countWidgets}:contains('${listEntity}')`)
                .find('button')
                .invoke('attr', 'disabled', false)
                .click();
            cy.wait(2000);
            cy.get(`${contextSelector} .rt-tr-group`);
            cy.get(`${contextSelector} [data-testid="panel-header"]`)
                .invoke('text')
                .then(panelHeaderText => {
                    expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(count, 10));
                });
        });
};

export const pageEntityCountMatchesTableRows = listEntity => {
    entityCountMatchesTableRows(listEntity, 'Page');
};

export const sidePanelEntityCountMatchesTableRows = listEntity => {
    entityCountMatchesTableRows(listEntity, 'Side Panel');
};

export const entityListCountMatchesTableLinkCount = entities => {
    cy.get(configManagementSelectors.tableLinks)
        .contains(entities)
        .invoke('text')
        .then(value => {
            const numEntities = parseInt(value, 10);
            cy.get(configManagementSelectors.tableLinks)
                .contains(entities)
                .click();
            cy.get('[data-testid="side-panel"] [data-testid="panel-header"]')
                .invoke('text')
                .then(panelHeaderText => {
                    expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numEntities, 10));
                });
        });
};
