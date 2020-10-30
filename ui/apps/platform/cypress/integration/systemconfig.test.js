import selectors, { systemConfigUrl, text } from '../constants/SystemConfigPage';
import withAuth from '../helpers/basicAuth';
import { system as configApi } from '../constants/apiEndpoints';

function openConfigNav() {
    cy.get(selectors.navLinks.config).click();
}

function openTopNav() {
    cy.get(selectors.navLinks.topNav).last().click();
}

function editBaseConfig(type) {
    cy.get(selectors.pageHeader.editButton, { timeout: 10000 }).click();

    cy.get(selectors[type].config.toggle).should('exist');
    cy.get(selectors[type].config.toggle).click();
    cy.get(selectors[type].config.textInput, { timeout: 10000 }).type(text.banner);
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

function saveConfig(type) {
    cy.get(selectors.pageHeader.saveButton).click();
    cy.get(selectors[type].state).contains('enabled');
}

function disableConfig(type) {
    cy.get(selectors.pageHeader.editButton).click();
    cy.get(selectors[type].config.toggle).click();
    cy.get(selectors.pageHeader.saveButton).click();
    cy.get(selectors[type].state).contains('disabled');
}

function getNumericInputByLabel(labelName) {
    return `label:contains('${labelName}') + .react-numeric-input input`;
}

function getRandomNumber() {
    return Math.floor(Math.random() * 100);
}

describe('System Configuration', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', configApi.config).as('getSystemConfig');
    });

    it('should allow the user to set data retention to "never delete"', () => {
        const neverDeletedText = 'Never deleted';

        cy.visit(systemConfigUrl);
        cy.wait('@getSystemConfig');
        cy.get(selectors.pageHeader.editButton).click();

        // If you reran the test without setting these random values first, it won’t save.
        // The save button is disabled when the form is pristine (ie. already 0)
        cy.get(getNumericInputByLabel('All Runtime Violations')).clear().type(getRandomNumber());
        cy.get(getNumericInputByLabel('Runtime Violations For Deleted Deployments'))
            .clear()
            .type(getRandomNumber());
        cy.get(getNumericInputByLabel('Resolved Deploy-Phase Violations'))
            .clear()
            .type(getRandomNumber());
        cy.get(getNumericInputByLabel('Images No Longer Deployed')).clear().type(getRandomNumber());
        cy.get(selectors.pageHeader.saveButton).click();
        cy.wait('@getSystemConfig');
        // Change input values to 0 to set it to "never delete"
        cy.get(selectors.pageHeader.editButton).click();
        cy.get(getNumericInputByLabel('All Runtime Violations')).clear().type(0);
        cy.get(getNumericInputByLabel('Runtime Violations For Deleted Deployments'))
            .clear()
            .type(0);
        cy.get(getNumericInputByLabel('Resolved Deploy-Phase Violations')).clear().type(0);
        cy.get(getNumericInputByLabel('Images No Longer Deployed')).clear().type(0);
        cy.get(selectors.pageHeader.saveButton).click();

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

    it('should have link from Platform Configuration side nav sub-menu', () => {
        cy.visit('/');
        openConfigNav();
        cy.get(selectors.navLinks.systemConfig).should('have.text', 'System Configuration');
    });

    it('should go to page', () => {
        cy.visit('/');
        openConfigNav();
        cy.get(selectors.navLinks.systemConfig).click();
        cy.url().should('contain', systemConfigUrl);
        cy.wait('@getSystemConfig');
        cy.get(selectors.dataRetention.widget).should('exist');
        cy.get(selectors.header.widget).should('exist');
        cy.get(selectors.footer.widget).should('exist');
        cy.get(selectors.loginNotice.widget).should('exist');
        cy.get(selectors.downloadTelemetry.widget).should('exist');
    });

    it('should be able to edit and enable header', () => {
        cy.visit(systemConfigUrl);
        cy.wait('@getSystemConfig');
        editBaseConfig('header');
        editBannerConfig('header');
        saveConfig('header');

        cy.get(selectors.header.banner).should('exist');
        disableConfig('header');
        cy.get(selectors.header.banner).should('not.exist');
    });

    it('should be able to edit and enable footer', () => {
        cy.visit(systemConfigUrl);
        cy.wait('@getSystemConfig');
        editBaseConfig('footer');
        editBannerConfig('footer');
        saveConfig('footer');
        cy.get(selectors.footer.banner).should('exist');
        disableConfig('footer');
        cy.get(selectors.footer.banner).should('not.exist');
    });

    it('should be able to edit and enable login notice', () => {
        cy.visit(systemConfigUrl);
        cy.wait('@getSystemConfig');
        editBaseConfig('loginNotice');
        saveConfig('loginNotice');
        openTopNav();
        cy.get(selectors.navLinks.logout).click();
        cy.get(selectors.loginNotice.banner).should('exist');
    });
});
