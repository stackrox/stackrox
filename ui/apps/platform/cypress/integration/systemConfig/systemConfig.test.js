import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    logOut,
    saveSystemConfiguration,
    visitSystemConfiguration,
    visitSystemConfigurationFromLeftNav,
    visitSystemConfigurationWithStaticResponseForPermissions,
} from './systemConfig.helpers';
import { selectors, text } from './systemConfig.selectors';

function editBaseConfig(type) {
    cy.get('button:contains("Edit")').click();

    cy.get(selectors[type].config.toggle).should('exist');
    cy.get(selectors[type].config.toggle).check({ force: true }); // force for PatternFly Switch element
    cy.get(selectors[type].config.textInput).type(text.banner);
}

function editBannerConfig(type) {
    cy.get(selectors[type].config.colorPickerButton).click();
    cy.get(selectors[type].config.colorInput).clear();
    cy.get(selectors[type].config.colorInput).type(text.color);
    cy.get(selectors[type].widget).click();
    cy.get(selectors[type].config.size.input).click();
    cy.get(selectors[type].config.size.options).first().click();
    cy.get(selectors[type].config.backgroundColorPickerButton).click();
    cy.get(selectors[type].config.colorInput).clear();
    cy.get(selectors[type].config.colorInput).type(text.backgroundColor);
    cy.get(selectors[type].widget).click();
}

function disableConfig(type) {
    cy.get('button:contains("Edit")').click();
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
        visitSystemConfigurationFromLeftNav();

        cy.get(selectors.dataRetention.widget).should('exist');
        cy.get(selectors.header.widget).should('exist');
        cy.get(selectors.footer.widget).should('exist');
        cy.get(selectors.loginNotice.widget).should('exist');
    });

    it('should have title', () => {
        visitSystemConfiguration();

        cy.title().should('match', getRegExpForTitleWithBranding('System Configuration'));
    });

    it('should not render Edit button if READ_ACCESS to resource', () => {
        cy.fixture('auth/mypermissionsMinimalAccess.json').then(({ resourceToAccess }) => {
            const staticResponseForPermissions = {
                body: {
                    resourceToAccess: { ...resourceToAccess, Administration: 'READ_ACCESS' },
                },
            };

            visitSystemConfigurationWithStaticResponseForPermissions(staticResponseForPermissions);

            cy.get('button:contains("Edit")').should('not.exist');
        });
    });

    it('should allow the user to set data retention to "never delete"', () => {
        visitSystemConfiguration();

        const neverDeletedText = 'Never deleted';

        cy.get('button:contains("Edit")').click();

        // If you reran the test without setting these random values first, it won’t save.
        // The save button is disabled when the form is pristine (ie. already 0)
        cy.get(getNumericInputByLabel('All runtime violations')).clear();
        cy.get(getNumericInputByLabel('All runtime violations')).type(getRandomNumber());
        cy.get(getNumericInputByLabel('Runtime violations for deleted deployments')).clear();
        cy.get(getNumericInputByLabel('Runtime violations for deleted deployments')).type(
            getRandomNumber()
        );
        cy.get(getNumericInputByLabel('Resolved deploy-phase violations')).clear();
        cy.get(getNumericInputByLabel('Resolved deploy-phase violations')).type(getRandomNumber());
        cy.get(getNumericInputByLabel('Images no longer deployed')).clear();
        cy.get(getNumericInputByLabel('Images no longer deployed')).type(getRandomNumber());

        saveSystemConfiguration();

        // Change input values to 0 to set it to "never delete"
        cy.get('button:contains("Edit")').click();

        cy.get(getNumericInputByLabel('All runtime violations')).clear();
        cy.get(getNumericInputByLabel('All runtime violations')).type(0);
        cy.get(getNumericInputByLabel('Runtime violations for deleted deployments')).clear();
        cy.get(getNumericInputByLabel('Runtime violations for deleted deployments')).type(0);
        cy.get(getNumericInputByLabel('Resolved deploy-phase violations')).clear();
        cy.get(getNumericInputByLabel('Resolved deploy-phase violations')).type(0);
        cy.get(getNumericInputByLabel('Images no longer deployed')).clear();
        cy.get(getNumericInputByLabel('Images no longer deployed')).type(0);

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

        logOut();

        cy.get(selectors.loginNotice.banner).should('exist');
    });
});
