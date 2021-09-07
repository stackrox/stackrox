import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { editIntegration } from './integrationUtils';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateUniqueName,
    getMultiSelectButtonByLabel,
    getMultiSelectOption,
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/clairify/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Clairify Test'));
        getMultiSelectButtonByLabel('Type').click();
        getMultiSelectOption('Image Scanner').click();
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/docker/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Docker Test'));
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/anchore/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Docker Test'));
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/ecr/create');

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Registry id').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('Region').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Registry id').contains('A registry id is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
        getHelperElementByLabel('Region').contains('An AWS region is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid from and save
        getInputByLabel('Integration name').clear().type(generateUniqueName('ECR Test'));
        getInputByLabel('Registry id').clear().type('12345');
        getInputByLabel('Endpoint').clear().type('test.endpoint');
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/google/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('ECR Test'));
        getMultiSelectButtonByLabel('Type').click();
        getMultiSelectOption('Registry').click();
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/azure/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Azure Test'));
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/artifactory/create');

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
            .type(generateUniqueName('JFrog Artifactory Test'));
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/dtr/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('DTR Test'));
        getMultiSelectButtonByLabel('Type').click();
        getMultiSelectOption('Registry').click();
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/quay/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Quay Test'));
        getMultiSelectButtonByLabel('Type').click();
        getMultiSelectOption('Registry').click();
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/clair/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Clair Test'));
        getMultiSelectButtonByLabel('Type').click();
        getMultiSelectOption('Image Scanner').click();
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/nexus/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Nexus Test'));
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/tenable/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('Tenable Test'));
        getMultiSelectButtonByLabel('Type').click();
        getMultiSelectOption('Registry').click();
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/ibm/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('IBM Test'));
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

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/imageIntegrations/rhel/create');

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
        getInputByLabel('Integration name').clear().type(generateUniqueName('RHEL Test'));
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

    it('should show a hint about stored credentials for Docker Trusted Registry', () => {
        cy.get(selectors.dockerTrustedRegistryTile).click();
        editIntegration('DTR Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Quay', () => {
        cy.get(selectors.quayTile).click();
        editIntegration('Quay Test');
        cy.get('div:contains("OAuth Token"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Amazon ECR', () => {
        cy.get(selectors.amazonECRTile).click();
        editIntegration('Amazon ECR Test');
        cy.get('div:contains("Access Key ID"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Secret Access Key"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Tenable', () => {
        cy.get(selectors.tenableTile).click();
        editIntegration('Tenable Test');
        cy.get('div:contains("Access Key"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Secret Key"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Google Container Registry', () => {
        cy.get(selectors.googleContainerRegistryTile).click();
        editIntegration('Google Container Registry Test');
        cy.get('div:contains("Service Account Key"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Anchore Scanner', () => {
        cy.get(selectors.anchoreScannerTile).click();
        editIntegration('Anchore Scanner Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for IBM Cloud', () => {
        cy.get(selectors.ibmCloudTile).click();
        editIntegration('IBM Cloud Test');
        cy.get('div:contains("API Key"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Microsoft ACR', () => {
        cy.get(selectors.microsoftACRTile).click();
        editIntegration('Microsoft ACR Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for JFrog Artifactory', () => {
        cy.get(selectors.jFrogArtifactoryTile).click();
        editIntegration('JFrog Artifactory Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Sonatype Nexus', () => {
        cy.get(selectors.sonatypeNexusTile).click();
        editIntegration('Sonatype Nexus Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Red Hat', () => {
        cy.get(selectors.redHatTile).click();
        editIntegration('Red Hat Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });
});
