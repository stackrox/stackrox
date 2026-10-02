import { PublicConfigContext } from 'hooks/usePublicConfig';
import BaseImagesModal from './BaseImagesModal';

const mockPublicConfigValue = {
    publicConfig: { telemetry: { enabled: false } },
    isLoadingPublicConfig: false,
    error: undefined,
    refetchPublicConfig: () => {},
};

function TestWrapper({ children }) {
    return (
        <PublicConfigContext.Provider value={mockPublicConfigValue}>
            {children}
        </PublicConfigContext.Provider>
    );
}

describe('BaseImagesModal', () => {
    beforeEach(() => {
        cy.intercept('POST', '/v2/baseimages', { statusCode: 200, body: {} }).as('addBaseImage');
    });

    describe('form validation', () => {
        it('should show error when field is empty after blur', () => {
            const onClose = cy.stub();
            const onSuccess = cy.stub();

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').focus();
            cy.get('input#baseImagePath').blur();
            cy.contains('Base image path is required').should('be.visible');
        });

        it('should show error when path is missing tag separator', () => {
            const onClose = cy.stub();
            const onSuccess = cy.stub();

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').type('ubuntu');
            cy.get('input#baseImagePath').blur();
            cy.contains('Base image path must include a tag pattern after ":"').should(
                'be.visible'
            );
            cy.contains('For registries with a port, use registry:port/repo:tag').should(
                'be.visible'
            );
        });

        it('should show error when the pattern is placed on the path of a registry with a port', () => {
            const onClose = cy.stub();
            const onSuccess = cy.stub();

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            // Registry port colon must not be mistaken for the tag separator.
            cy.get('input#baseImagePath').type('registry:5000/repo*');
            cy.get('input#baseImagePath').blur();
            cy.contains('Base image path must include a tag pattern after ":"').should(
                'be.visible'
            );
        });

        it('should reject invalid tag pattern characters', () => {
            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={cy.stub()} onSuccess={cy.stub()} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').type('registry:5000/repo:1.@bad');
            cy.get('input#baseImagePath').blur();
            cy.contains('Tag pattern must not contain "/", ":", "@", or whitespace').should(
                'be.visible'
            );
            cy.findByRole('button', { name: 'Save' }).should('be.disabled');
        });

        it('should reject tag patterns longer than 128 bytes', () => {
            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={cy.stub()} onSuccess={cy.stub()} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').type(`ubuntu:${'a'.repeat(129)}`);
            cy.get('input#baseImagePath').blur();
            cy.contains('Tag pattern must be at most 128 bytes').should('be.visible');
            cy.findByRole('button', { name: 'Save' }).should('be.disabled');
        });

        it('should enable save button when form is valid', () => {
            const onClose = cy.stub();
            const onSuccess = cy.stub();

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            cy.findByRole('button', { name: 'Save' }).should('be.disabled');
            cy.get('input#baseImagePath').type('ubuntu:22.04');
            cy.findByRole('button', { name: 'Save' }).should('not.be.disabled');
        });
    });

    describe('alert rendering', () => {
        it('should show success alert after successful submission', () => {
            const onClose = cy.stub();
            const onSuccess = cy.stub();

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').type('ubuntu:22.04');
            cy.findByRole('button', { name: 'Save' }).click();

            cy.wait('@addBaseImage');
            cy.contains('Base image successfully added').should('be.visible');
        });

        it('should show error alert when submission fails', () => {
            cy.intercept('POST', '/v2/baseimages', {
                statusCode: 500,
                body: { message: 'Internal server error' },
            }).as('addBaseImageError');

            const onClose = cy.stub();
            const onSuccess = cy.stub();

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').type('ubuntu:22.04');
            cy.findByRole('button', { name: 'Save' }).click();

            cy.wait('@addBaseImageError');
            cy.contains('Error adding base image').should('be.visible');
        });
    });
});
