import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';

describe('Image Integrations Test', () => {
    withAuth();

    beforeEach(() => {
        cy.intercept('GET', api.integrations.imageIntegrations, {
            fixture: 'integrations/imageIntegrations.json',
        }).as('getImageIntegrations');
        cy.intercept('POST', api.integrations.imageIntegrations, {}).as('postImageIntegrations');

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getImageIntegrations');
    });

    it('should create a new StackRox Scanner integration', () => {
        cy.get(selectors.clairifyTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Clairify Test'));
        getSelectButtonByLabel('Type').click();
        getSelectOption('Image Scanner').click();
        getInputByLabel('Endpoint').clear().type('https://scanner.stackrox:8080');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/clairify');
        });
    });

    it('should create a new Generic Docker Registry integration', () => {
        cy.get(selectors.dockerRegistryTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Docker Test'));
        getInputByLabel('Endpoint').clear().type('registry-1.docker.io');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/docker');
        });
    });

    it('should create a new Anchore integration', () => {
        cy.get(selectors.anchoreScannerTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('Username').type(' ');
        getInputByLabel('Password').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Username').contains('A username is required');
        getHelperElementByLabel('Password').contains('A password is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Docker Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').clear().type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/anchore');
        });
    });

    it('should create a new Amazon ECR integration', () => {
        cy.get(selectors.amazonECRTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Registry ID').type(' ');
        getInputByLabel('Region').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Registry ID').contains('A registry ID is required');
        getHelperElementByLabel('Region').contains('An AWS region is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid form and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('ECR Test'));
        getInputByLabel('Registry ID').clear().type('12345');
        getInputByLabel('Region').clear().type('us-west-1');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/ecr');
        });
    });

    it('should create a new Google Container Registry integration', () => {
        cy.get(selectors.googleContainerRegistryTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Registry endpoint').type(' ');
        getInputByLabel('Project').type(' ');
        getInputByLabel('Service account key (JSON)').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Registry endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Project').contains('A project is required');
        getHelperElementByLabel('Service account key (JSON)').contains(
            'A service account key is required'
        );
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('ECR Test'));
        getSelectButtonByLabel('Type').click();
        getSelectOption('Registry').click();
        getInputByLabel('Registry endpoint').clear().type('test.endpoint');
        getInputByLabel('Project').clear().type('test');
        getInputByLabel('Service account key (JSON)').type('{"key":"value"}', {
            parseSpecialCharSequences: false,
        });

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/google');
        });
    });

    it('should create a new Microsoft Azure integration', () => {
        cy.get(selectors.microsoftACRTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('Password').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Password').contains('A password is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Azure Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/azure');
        });
    });

    it('should create a new JFrog Artifactory integration', () => {
        cy.get(selectors.jFrogArtifactoryTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('Password').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Password').contains('A password is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name')
            .clear()
            .type(generateNameWithDate('JFrog Artifactory Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/artifactory');
        });
    });

    it('should create a new Docker Trusted Registry integration', () => {
        cy.get(selectors.dockerTrustedRegistryTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('Username').type(' ');
        getInputByLabel('Password').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Username').contains('A username is required');
        getHelperElementByLabel('Password').contains('A password is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('DTR Test'));
        getSelectButtonByLabel('Type').click();
        getSelectOption('Registry').click();
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/dtr');
        });
    });

    it('should create a new Quay integration', () => {
        cy.get(selectors.quayTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('OAuth token').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('OAuth token').contains('An OAuth token is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Quay Test'));
        getSelectButtonByLabel('Type').click();
        getSelectOption('Registry').click();
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('OAuth token').clear().type('12345');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/quay');
        });
    });

    it('should create a new Clair integration', () => {
        cy.get(selectors.clairTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Clair Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/clair');
        });
    });

    it('should create a new Nexus integration', () => {
        cy.get(selectors.sonatypeNexusTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('Password').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Password').contains('A password is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Nexus Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').clear().type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/nexus');
        });
    });

    it('should create a new Tenable integration', () => {
        cy.get(selectors.tenableTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Access key').type(' ');
        getInputByLabel('Secret key').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Access key').contains('An access key is required');
        getHelperElementByLabel('Secret key').contains('A secret key is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('Tenable Test'));
        getSelectButtonByLabel('Type').click();
        getSelectOption('Registry').click();
        getInputByLabel('Access key').clear().type('12345');
        getInputByLabel('Secret key').clear().type('12345');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/tenable');
        });
    });

    it('should create a new IBM integration', () => {
        cy.get(selectors.ibmCloudTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('API key').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('API key').contains('An API key is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('IBM Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('API key').clear().type('12345');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/ibm');
        });
    });

    it('should create a new RHEL integration', () => {
        cy.get(selectors.redHatTile).click();

        cy.get(selectors.buttons.new).click();

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateNameWithDate('RHEL Test'));
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').clear().type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        cy.get(selectors.buttons.save).should('be.enabled').click();
        cy.wait('@postImageIntegrations');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/imageIntegrations/rhel');
        });
    });
});
