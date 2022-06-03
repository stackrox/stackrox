import selectors, { systemConfigUrl, text } from '../constants/SystemConfigPage';
import withAuth from '../helpers/basicAuth';
import { visitFromLeftNavExpandable } from '../helpers/nav';
import { system as configApi } from '../constants/apiEndpoints';

function visitSytemConfigurationFromLeftNav() {
    cy.intercept('GET', configApi.config).as('getSystemConfiguration');
    visitFromLeftNavExpandable('Platform Configuration', 'System Configuration');
    cy.wait('@getSystemConfiguration');
}

function visitSystemConfiguration() {
    cy.intercept('GET', configApi.config).as('getSystemConfiguration');
    cy.visit(systemConfigUrl);
    cy.wait('@getSystemConfiguration');
}

function editBaseConfig(type) {
    cy.get(selectors.pageHeader.editButton).click();

    cy.get(selectors[type].config.toggle).should('exist');
    cy.get(selectors[type].config.toggle).check({ force: true }); // force for PatternFly Switch element
    cy.get(selectors[type].config.textInput).type(text.banner);
}

function editBannerConfig(type) {
    cy.get(selectors[type].config.colorPickerBtn).first().click();
    cy.get(selectors[type].config.colorInput).clear().type(text.color);
    cy.get(selectors[type].widget).click();
    cy.get(selectors[type].config.size.input).click();
    cy.get(selectors[type].config.size.options).first().click();
    cy.get(selectors[type].config.colorPickerBtn).last().click();
    cy.get(selectors[type].config.colorInput).clear().type(text.backgroundColor);
    cy.get(selectors[type].widget).click();
}

function saveSystemConfiguration() {
    cy.intercept('PUT', configApi.config).as('putSystemConfiguration');
    cy.get(selectors.pageHeader.saveButton).click();
    cy.wait('@putSystemConfiguration');
}

function disableConfig(type) {
    cy.get(selectors.pageHeader.editButton).click();
    cy.get(selectors[type].config.toggle).uncheck({ force: true }); // force for PatternFly Switch element
    saveSystemConfiguration();
    cy.get(selectors[type].state).contains('Disabled');
}

function getNumericInputByLabel(labelName) {
    return `.pf-c-form__group:contains("${labelName}") input`;
}

function getRandomNumber() {
    return Math.floor(Math.random() * 100);
}

describe('System Configuration', () => {
    withAuth();

    it('should go to System Configuration from main navigation', () => {
        visitSytemConfigurationFromLeftNav();
        cy.location('pathname').should('eq', systemConfigUrl);
        cy.get('h1:contains("System Configuration")');
        cy.get(selectors.dataRetention.widget).should('exist');
        cy.get(selectors.header.widget).should('exist');
        cy.get(selectors.footer.widget).should('exist');
        cy.get(selectors.loginNotice.widget).should('exist');
    });

    it('should allow the user to set data retention to "never delete"', () => {
        visitSystemConfiguration();

        const neverDeletedText = 'Never deleted';

        cy.get(selectors.pageHeader.editButton).click();

        // If you reran the test without setting these random values first, it won’t save.
        // The save button is disabled when the form is pristine (ie. already 0)
        cy.get(getNumericInputByLabel('All runtime violations')).clear().type(getRandomNumber());
        cy.get(getNumericInputByLabel('Runtime violations for deleted deployments'))
            .clear()
            .type(getRandomNumber());
        cy.get(getNumericInputByLabel('Resolved deploy-phase violations'))
            .clear()
            .type(getRandomNumber());
        cy.get(getNumericInputByLabel('Images no longer deployed')).clear().type(getRandomNumber());

        saveSystemConfiguration();

        // Change input values to 0 to set it to "never delete"
        cy.get(selectors.pageHeader.editButton).click();

        cy.get(getNumericInputByLabel('All runtime violations')).clear().type(0);
        cy.get(getNumericInputByLabel('Runtime violations for deleted deployments'))
            .clear()
            .type(0);
        cy.get(getNumericInputByLabel('Resolved deploy-phase violations')).clear().type(0);
        cy.get(getNumericInputByLabel('Images no longer deployed')).clear().type(0);

        saveSystemConfiguration();

        cy.get(selectors.dataRetention.allRuntimeViolationsBox).should('contain', neverDeletedText);
        cy.get(selectors.dataRetention.resolvedDeployViolationsBox).should(
            'contain',
            neverDeletedText
        );
        cy.get(selectors.dataRetention.imagesBox).should('contain', neverDeletedText);
        cy.get(selectors.dataRetention.deletedRuntimeViolationsBox).should(
            'contain',
            neverDeletedText
        );
    });

    it('should be able to edit and enable header', () => {
        visitSystemConfiguration();

        editBaseConfig('header');
        editBannerConfig('header');
        saveSystemConfiguration();

        cy.get(selectors.header.state).contains('Enabled');
        cy.get(selectors.header.banner).should('exist');

        disableConfig('header');
        cy.get(selectors.header.banner).should('not.exist');
    });

    it('should be able to edit and enable footer', () => {
        visitSystemConfiguration();

        editBaseConfig('footer');
        editBannerConfig('footer');
        saveSystemConfiguration();

        cy.get(selectors.footer.state).contains('Enabled');
        cy.get(selectors.footer.banner).should('exist');

        disableConfig('footer');
        cy.get(selectors.footer.banner).should('not.exist');
    });

    it('should be able to edit and enable login notice', () => {
        visitSystemConfiguration();

        editBaseConfig('loginNotice');
        saveSystemConfiguration();

        cy.get(selectors.loginNotice.state).contains('Enabled');
        cy.get(selectors.navLinks.topNav).click();
        cy.get(selectors.navLinks.logout).click();
        cy.get(selectors.loginNotice.banner).should('exist');
    });
});
