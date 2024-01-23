import withAuth from '../../helpers/basicAuth';
import { hasOrchestratorFlavor } from '../../helpers/features';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getToggleGroupItem,
} from '../../helpers/formHelpers';

import {
    clickCreateNewIntegrationInTable,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    testIntegrationInFormWithStoredCredentials,
    testIntegrationInFormWithoutStoredCredentials,
    visitIntegrationsTable,
    visitIntegrationsWithStaticResponseForCapabilities,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'imageIntegrations';

const staticResponseForTest = { body: {} };

const staticResponseForPOST = {
    body: { id: 'abcdefgh' },
};

describe('Image Integrations', () => {
    withAuth();

    it('should create a new StackRox Scanner integration', () => {
        const integrationName = generateNameWithDate('StackRox Scanner Test');
        const integrationType = 'clairify';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithoutStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new Generic Docker Registry integration', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        const integrationName = generateNameWithDate('Generic Docker Registry Test');
        const integrationType = 'docker';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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
        getInputByLabel('Create integration without testing').click();

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        deleteIntegrationInTable(integrationSource, integrationType, integrationName);
    });

    it('should create a new Amazon ECR integration', () => {
        const integrationName = generateNameWithDate('Amazon ECR Test');
        const integrationType = 'ecr';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('12-digit AWS ID').type(' ');
        getInputByLabel('Region').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('12-digit AWS ID').contains('A 12-digit AWS ID is required');
        getHelperElementByLabel('Region').contains('An AWS region is required');
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check valid form and save
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('12-digit AWS ID').clear().type('12345789012');
        getInputByLabel('Region').clear().type('us-west-1');
        cy.get('label:contains("Use container IAM role")').click(); // turn on Use IAM Role

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should not render IAM Role on ECR form, when that capability is disabled', () => {
        visitIntegrationsWithStaticResponseForCapabilities(
            {
                body: { centralScanningCanUseContainerIamRoleForEcr: 'CapabilityDisabled' },
            },
            'imageIntegrations',
            'ecr',
            '',
            'create'
        );

        cy.get('label:contains("Use container IAM role")').should('not.exist');
    });

    it('should create a new Google Artifact Registry integration', () => {
        const integrationName = generateNameWithDate('Google Artifact Registry Test');
        const integrationType = 'artifactregistry';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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
            'Valid JSON is required for service account key'
        );
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check conditional fields

        // Step 2.1, enable workload identity, this should remove the service account field
        getInputByLabel('Use workload identity').click();
        getInputByLabel('Service account key (JSON)').should('be.disabled');
        // Step 2.2, disable workload identity, this should render the service account field again
        getInputByLabel('Use workload identity').click();
        getInputByLabel('Service account key (JSON)').should('be.enabled');

        // Step 3, check valid from and save
        getInputByLabel('Integration name').clear().type(integrationName);

        getInputByLabel('Registry endpoint').clear().type('test.endpoint');
        getInputByLabel('Project').clear().type('test');
        getInputByLabel('Service account key (JSON)').type('{"key":"value"}', {
            parseSpecialCharSequences: false,
        });

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new Google Container Registry integration', () => {
        const integrationName = generateNameWithDate('Google Container Registry Test');
        const integrationType = 'google';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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
            'Valid JSON is required for service account key'
        );
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 2, check conditional fields

        // Step 2.1, enable workload identity, this should remove the service account field
        getInputByLabel('Use workload identity').click();
        getInputByLabel('Service account key (JSON)').should('be.disabled');
        // Step 2.2, disable workload identity, this should render the service account field again
        getInputByLabel('Use workload identity').click();
        getInputByLabel('Service account key (JSON)').should('be.enabled');

        // Step 3, check valid from and save
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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new Microsoft Azure integration', () => {
        const integrationName = generateNameWithDate('Microsoft Azure Test');
        const integrationType = 'azure';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new JFrog Artifactory integration', () => {
        const integrationName = generateNameWithDate('JFrog Artifactory Test');
        const integrationType = 'artifactory';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new Quay integration', () => {
        const integrationName = generateNameWithDate('Quay Test');
        const integrationType = 'quay';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Step 0, should start out with disabled Save and Test buttons
        cy.get(selectors.buttons.test).should('be.disabled');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').type(' ');
        getInputByLabel('OAuth token').type(' ').blur();

        getHelperElementByLabel('Integration name').contains('An integration name is required');
        getHelperElementByLabel('Endpoint').contains('An endpoint is required');
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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new Clair integration', () => {
        const integrationName = generateNameWithDate('Clair Test');
        const integrationType = 'clair';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithoutStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new Nexus integration', () => {
        const integrationName = generateNameWithDate('Nexus Test');
        const integrationType = 'nexus';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new IBM integration', () => {
        const integrationName = generateNameWithDate('IBM Test');
        const integrationType = 'ibm';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });

    it('should create a new RHEL integration', () => {
        const integrationName = generateNameWithDate('RHEL Test');
        const integrationType = 'rhel';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        testIntegrationInFormWithStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

        // Test does not delete, because it did not create.
    });
});
