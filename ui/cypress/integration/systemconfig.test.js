import selectors, { systemConfigUrl, text } from './constants/SystemConfigPage';
import withAuth from './helpers/basicAuth';

describe('System Config', () => {
    withAuth();

    const openNav = () => {
        cy.get(selectors.navLinks.topNav)
            .last()
            .click();
    };

    const editBaseConfig = type => {
        cy.get(selectors.pageHeader.editButton).click();

        cy.get(selectors[type].config.toggle).should('exist');
        cy.get(selectors[type].config.toggle).click();
        cy.get(selectors[type].config.textInput).type(text.banner);
    };

    const editBannerConfig = type => {
        cy.get(selectors[type].config.colorPickerBtn)
            .first()
            .click();
        cy.get(selectors[type].config.colorInput)
            .clear()
            .type(text.color);
        cy.get(selectors[type].widget).click();
        cy.get(selectors[type].config.size.input).click();
        cy.get(selectors[type].config.size.options)
            .first()
            .click();
        cy.get(selectors[type].config.colorPickerBtn)
            .last()
            .click();
        cy.get(selectors[type].config.colorInput)
            .clear()
            .type(text.backgroundColor);
        cy.get(selectors[type].widget).click();
    };

    const saveConfig = type => {
        cy.get(selectors.pageHeader.saveButton).click();
        cy.get(selectors[type].state).contains('enabled');
    };

    const disableConfig = type => {
        cy.get(selectors.pageHeader.editButton).click();
        cy.get(selectors[type].config.toggle).click();
        cy.get(selectors.pageHeader.saveButton).click();
        cy.get(selectors[type].state).contains('disabled');
    };

    it('should have link from top navigation', () => {
        cy.visit('/');
        openNav();
        cy.get(selectors.navLinks.menu).contains('System Config');
    });

    it('should go to System Config page', () => {
        cy.visit('/');
        openNav();
        cy.get(selectors.navLinks.systemConfig).click();
        cy.url().should('contain', systemConfigUrl);
        cy.get(selectors.header.widget).should('exist');
        cy.get(selectors.footer.widget).should('exist');
        cy.get(selectors.loginNotice.widget).should('exist');
    });

    it('should be able to edit and enable header', () => {
        cy.visit(systemConfigUrl);
        editBaseConfig('header');
        editBannerConfig('header');
        saveConfig('header');

        cy.get(selectors.header.banner).should('exist');
        disableConfig('header');
        cy.get(selectors.header.banner).should('not.exist');
    });

    it('should be able to edit and enable footer', () => {
        cy.visit(systemConfigUrl);
        editBaseConfig('footer');
        editBannerConfig('footer');
        saveConfig('footer');
        cy.get(selectors.footer.banner).should('exist');
        disableConfig('footer');
        cy.get(selectors.footer.banner).should('not.exist');
    });

    it('should be able to edit and enable login notice', () => {
        cy.visit(systemConfigUrl);
        editBaseConfig('loginNotice');
        saveConfig('loginNotice');
        openNav();
        cy.get(selectors.navLinks.logout).click();
        cy.get(selectors.loginNotice.banner).should('exist');
    });
});
