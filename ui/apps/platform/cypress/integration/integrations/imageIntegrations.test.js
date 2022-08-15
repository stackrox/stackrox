import * as api from '../../constants/apiEndpoints';
import { labels, selectors, url } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getToggleGroupItem,
} from '../../helpers/formHelpers';
import { visitIntegrationsUrl } from '../../helpers/integrations';

function assertImageIntegrationTable(integrationType) {
    const label = labels.imageIntegrations[integrationType];
    cy.get(`${selectors.breadcrumbItem}:contains("${label}")`);
    cy.get(`${selectors.title2}:contains("${label}")`);
}

function getImageIntegrationTypeUrl(integrationType) {
    return `${url}/imageIntegrations/${integrationType}`;
}

function visitImageIntegrationType(integrationType) {
    visitIntegrationsUrl(getImageIntegrationTypeUrl(integrationType));
    assertImageIntegrationTable(integrationType);
}

function saveImageIntegrationType(integrationType) {
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    // Mock request.
    cy.intercept('POST', api.integrations.imageIntegrations, {}).as('postImageIntegration');
    cy.get(selectors.buttons.save).should('be.enabled').click();
    cy.wait(['@postImageIntegration', '@getImageIntegrations']);
    assertImageIntegrationTable(integrationType);
    cy.location('pathname').should('eq', getImageIntegrationTypeUrl(integrationType));
}

describe('Image Integrations Test', () => {
    withAuth();

    it('should create a new StackRox Scanner integration', () => {
        const integrationName = generateNameWithDate('StackRox Scanner Test');
        const integrationType = 'clairify';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);

        const selected = 'pf-m-selected';
        getToggleGroupItem('Type', 0, 'Image Scanner').should('have.class', selected);
        getToggleGroupItem('Type', 1, 'Node Scanner').should('not.have.class', selected);
        getToggleGroupItem('Type', 2, 'Image Scanner + Node Scanner').should(
            'not.have.class',
            selected
        );
        getToggleGroupItem('Type', 2, 'Image Scanner + Node Scanner')
            .click()
            .should('have.class', selected);

        getInputByLabel('Endpoint').clear().type('https://scanner.stackrox:8080');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Generic Docker Registry integration', () => {
        const integrationName = generateNameWithDate('Generic Docker Registry Test');
        const integrationType = 'docker';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('registry-1.docker.io');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Amazon ECR integration', () => {
        const integrationName = generateNameWithDate('Amazon ECR Test');
        const integrationType = 'ecr';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Registry ID').clear().type('12345');
        getInputByLabel('Region').clear().type('us-west-1');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Google Container Registry integration', () => {
        const integrationName = generateNameWithDate('Google Container Registry Test');
        const integrationType = 'google';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);

        const selected = 'pf-m-selected';
        getToggleGroupItem('Type', 0, 'Registry').should('have.class', selected);
        getToggleGroupItem('Type', 1, 'Scanner').should('not.have.class', selected);
        getToggleGroupItem('Type', 2, 'Registry + Scanner').should('not.have.class', selected);
        getToggleGroupItem('Type', 2, 'Registry + Scanner').click().should('have.class', selected);

        getInputByLabel('Registry endpoint').clear().type('test.endpoint');
        getInputByLabel('Project').clear().type('test');
        getInputByLabel('Service account key (JSON)').type('{"key":"value"}', {
            parseSpecialCharSequences: false,
        });

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Microsoft Azure integration', () => {
        const integrationName = generateNameWithDate('Microsoft Azure Test');
        const integrationType = 'azure';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new JFrog Artifactory integration', () => {
        const integrationName = generateNameWithDate('JFrog Artifactory Test');
        const integrationType = 'artifactory';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Quay integration', () => {
        const integrationName = generateNameWithDate('Quay Test');
        const integrationType = 'quay';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);

        const selected = 'pf-m-selected';
        getToggleGroupItem('Type', 0, 'Registry').should('have.class', selected);
        getToggleGroupItem('Type', 1, 'Scanner').should('not.have.class', selected);
        getToggleGroupItem('Type', 2, 'Registry + Scanner').should('not.have.class', selected);
        getToggleGroupItem('Type', 2, 'Registry + Scanner').click().should('have.class', selected);

        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('OAuth token').clear().type('12345');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Clair integration', () => {
        const integrationName = generateNameWithDate('Clair Test');
        const integrationType = 'clair';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('test.endpoint');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new Nexus integration', () => {
        const integrationName = generateNameWithDate('Nexus Test');
        const integrationType = 'nexus';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').clear().type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new IBM integration', () => {
        const integrationName = generateNameWithDate('IBM Test');
        const integrationType = 'ibm';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('API key').clear().type('12345');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });

    it('should create a new RHEL integration', () => {
        const integrationName = generateNameWithDate('RHEL Test');
        const integrationType = 'rhel';
        visitImageIntegrationType(integrationType);
        cy.get(selectors.buttons.newIntegration).click();

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
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('test.endpoint');
        getInputByLabel('Username').clear().type('admin');
        getInputByLabel('Password').clear().type('password');

        cy.get(selectors.buttons.test).should('be.enabled');
        saveImageIntegrationType(integrationType);
    });
});
