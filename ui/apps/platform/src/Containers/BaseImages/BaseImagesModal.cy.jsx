import { useState } from 'react';
import { Button } from '@patternfly/react-core';

import { PublicConfigContext } from 'hooks/usePublicConfig';
import BaseImagesModal from './BaseImagesModal';

const mockPublicConfigValue = {
    publicConfig: { telemetry: { enabled: false } },
    isLoadingPublicConfig: false,
    error: undefined,
    refetchPublicConfig: () => {},
};

const baseImage = {
    id: 'base-image-id',
    baseImageRepoPath: 'docker.io/library/ubuntu',
    baseImageTagPattern: '22.04',
    user: { id: 'user-id', name: 'admin' },
};

function TestWrapper({ children }) {
    return (
        <PublicConfigContext.Provider value={mockPublicConfigValue}>
            {children}
        </PublicConfigContext.Provider>
    );
}

// Drives the modal like BaseImagesPage does: the modal stays mounted while it is closed, and the
// parent closes it from onSuccess without going through the modal's own close handler.
function BaseImagesModalHost({ baseImageToEdit = null, onSuccess }) {
    const [isOpen, setIsOpen] = useState(true);

    return (
        <TestWrapper>
            <Button onClick={() => setIsOpen(true)}>Open modal</Button>
            <BaseImagesModal
                isOpen={isOpen}
                onClose={() => setIsOpen(false)}
                onSuccess={(successMessage) => {
                    onSuccess(successMessage);
                    setIsOpen(false);
                }}
                baseImageToEdit={baseImageToEdit}
            />
        </TestWrapper>
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
            cy.contains('example-registry.io/path/to/image:tag').should('be.visible');
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
        it('should report success to the parent without an inline success alert', () => {
            const onClose = cy.stub();
            const onSuccess = cy.stub().as('onSuccess');

            cy.mount(
                <TestWrapper>
                    <BaseImagesModal isOpen onClose={onClose} onSuccess={onSuccess} />
                </TestWrapper>
            );

            cy.get('input#baseImagePath').type('ubuntu:22.04');
            cy.findByRole('button', { name: 'Save' }).click();

            cy.wait('@addBaseImage');
            cy.get('@onSuccess').should(
                'have.been.calledOnceWithExactly',
                'Base image successfully added'
            );
            cy.contains('Base image successfully added').should('not.exist');
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

    describe('reopening after a successful save', () => {
        it('should open the add form without a success alert', () => {
            const onSuccess = cy.stub().as('onSuccess');

            cy.mount(<BaseImagesModalHost onSuccess={onSuccess} />);

            cy.get('input#baseImagePath').type('ubuntu:22.04');
            cy.findByRole('button', { name: 'Save' }).click();

            cy.wait('@addBaseImage');
            cy.get('@onSuccess').should('have.been.calledOnce');
            cy.findByRole('dialog').should('not.exist');

            cy.findByRole('button', { name: 'Open modal' }).click();
            cy.findByRole('dialog').should('be.visible');
            cy.get('input#baseImagePath').should('have.value', '');
            cy.contains('Base image successfully added').should('not.exist');
        });

        it('should open the edit form without a success alert', () => {
            cy.intercept('PUT', '/v2/baseimages/*', { statusCode: 200, body: {} }).as(
                'updateBaseImage'
            );
            const onSuccess = cy.stub().as('onSuccess');

            cy.mount(<BaseImagesModalHost baseImageToEdit={baseImage} onSuccess={onSuccess} />);

            cy.get('input#baseImageTagPattern').clear();
            cy.get('input#baseImageTagPattern').type('24.*');
            cy.findByRole('button', { name: 'Save' }).click();

            cy.wait('@updateBaseImage');
            cy.get('@onSuccess').should(
                'have.been.calledOnceWithExactly',
                'Base image tag pattern successfully updated'
            );
            cy.findByRole('dialog').should('not.exist');

            cy.findByRole('button', { name: 'Open modal' }).click();
            cy.findByRole('dialog').should('be.visible');
            cy.contains('Base image tag pattern successfully updated').should('not.exist');
        });
    });
});
