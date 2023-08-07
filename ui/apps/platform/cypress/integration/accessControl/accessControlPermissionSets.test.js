import withAuth from '../../helpers/basicAuth';
import { assertCannotFindThePage } from '../../helpers/visit';

import {
    assertAccessControlEntityDoesNotExist,
    clickEntityNameInTable,
    permissionSetsKey as entitiesKey,
    visitAccessControlEntities,
    visitAccessControlEntitiesWithStaticResponseForPermissions,
    visitAccessControlEntity,
} from './accessControl.helpers';
import { selectors } from './accessControl.selectors';

const defaultNames = ['Admin', 'Analyst', 'Continuous Integration', 'None', 'Sensor Creator'];

describe('Access Control Permission sets', () => {
    withAuth();

    it('cannot find the page if no permission', () => {
        const staticResponseForPermissions = {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        };
        visitAccessControlEntitiesWithStaticResponseForPermissions(
            entitiesKey,
            staticResponseForPermissions
        );

        assertCannotFindThePage();
    });

    it('list has heading, button, and table head cells', () => {
        visitAccessControlEntities(entitiesKey);

        cy.contains('h2', /^\d+ results? found$/);

        cy.get('button:contains("Create permission set")');

        cy.get('th:contains("Name")');
        cy.get('th:contains("Origin")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("Roles")');
        cy.get('th[aria-label="Row actions"]');
    });

    it('list has default names', () => {
        visitAccessControlEntities(entitiesKey);

        defaultNames.forEach((defaultName) => {
            cy.get(`td[data-label="Name"] a:contains("${defaultName}")`);
        });
    });

    it('list link for default Admin goes to form which has label instead of button and disabled input values', () => {
        visitAccessControlEntities(entitiesKey);

        const entityName = 'Admin';
        clickEntityNameInTable(entitiesKey, entityName);

        cy.get('h2').should('have.text', entityName);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("${entityName}")`);

        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');

        const { getAccessLevelSelectForResource } = selectors.form.permissionSet;

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            $tds.get().forEach((td) => {
                const resource = td.textContent;
                cy.get(getAccessLevelSelectForResource(resource)).should('be.disabled');
            });
        });
    });

    it('direct link to default Admin has all read and write access', () => {
        const targetID = 'ffffffff-ffff-fff4-f5ff-ffffffffffff';
        visitAccessControlEntity(entitiesKey, targetID);

        cy.get(selectors.form.inputName).should('have.value', 'Admin');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', resourceCount);

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                cy.get(getReadAccessIconForResource(resource)).should(
                    'have.attr',
                    'aria-label',
                    'permitted'
                );
                cy.get(getWriteAccessIconForResource(resource)).should(
                    'have.attr',
                    'aria-label',
                    'permitted'
                );
                cy.get(getAccessLevelSelectForResource(resource)).should(
                    'contain',
                    'Read and Write Access'
                );
            });
        });
    });

    it('direct link to default Analyst has all (but Administration) read and no write access', () => {
        const targetID = 'ffffffff-ffff-fff4-f5ff-fffffffffffe';
        visitAccessControlEntity(entitiesKey, targetID);

        cy.get(selectors.form.inputName).should('have.value', 'Analyst');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', resourceCount - 1);
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', '0');

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                if (resource === 'Administration') {
                    cy.get(getReadAccessIconForResource(resource)).should(
                        'have.attr',
                        'aria-label',
                        'forbidden'
                    );
                    cy.get(getAccessLevelSelectForResource(resource)).should(
                        'contain',
                        'No Access'
                    );
                } else {
                    cy.get(getReadAccessIconForResource(resource)).should(
                        'have.attr',
                        'aria-label',
                        'permitted'
                    );
                    cy.get(getAccessLevelSelectForResource(resource)).should(
                        'contain',
                        'Read Access'
                    );
                }
                cy.get(getWriteAccessIconForResource(resource)).should(
                    'have.attr',
                    'aria-label',
                    'forbidden'
                );
            });
        });
    });

    it('direct link to default Continuous Integration has limited read and write accesss', () => {
        const targetID = 'ffffffff-ffff-fff4-f5ff-fffffffffffd';
        visitAccessControlEntity(entitiesKey, targetID);

        cy.get(selectors.form.inputName).should('have.value', 'Continuous Integration');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        const resourcesLimited = ['Detection', 'Image'];

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', '2');
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', '1');

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                if (!resourcesLimited.some((v) => resource === v)) {
                    cy.get(getReadAccessIconForResource(resource)).should(
                        'have.attr',
                        'aria-label',
                        'forbidden'
                    );
                    cy.get(getWriteAccessIconForResource(resource)).should(
                        'have.attr',
                        'aria-label',
                        'forbidden'
                    );
                    cy.get(getAccessLevelSelectForResource(resource)).should(
                        'contain',
                        'No Access'
                    );
                }
            });
        });

        cy.get(getReadAccessIconForResource('Detection')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getWriteAccessIconForResource('Detection')).should(
            'have.attr',
            'aria-label',
            'forbidden'
        );
        cy.get(getAccessLevelSelectForResource('Detection')).should('contain', 'Read Access');

        // Zero-based index for Image instead of ImageComponent, ImageIntegration, WatchedImage.
        cy.get(getReadAccessIconForResource('Image', 0)).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getWriteAccessIconForResource('Image', 0)).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getAccessLevelSelectForResource('Image', 0)).should(
            'contain',
            'Read and Write Access'
        );
    });

    it('direct link to default None has no read nor write access', () => {
        const targetID = 'ffffffff-ffff-fff4-f5ff-fffffffffffc';
        visitAccessControlEntity(entitiesKey, targetID);

        cy.get(selectors.form.inputName).should('have.value', 'None');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', '0');
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', '0');

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                cy.get(getReadAccessIconForResource(resource)).should(
                    'have.attr',
                    'aria-label',
                    'forbidden'
                );
                cy.get(getWriteAccessIconForResource(resource)).should(
                    'have.attr',
                    'aria-label',
                    'forbidden'
                );
                cy.get(getAccessLevelSelectForResource(resource)).should('contain', 'No Access');
            });
        });
    });

    it('direct link to default Sensor Creator has limited read and write access', () => {
        const targetID = 'ffffffff-ffff-fff4-f5ff-fffffffffffa';
        visitAccessControlEntity(entitiesKey, targetID);

        cy.get(selectors.form.inputName).should('have.value', 'Sensor Creator');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        const resourcesLimited = ['Cluster', 'Administration'];

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', '2');
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', '2');

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                if (!resourcesLimited.some((v) => resource === v)) {
                    cy.get(getReadAccessIconForResource(resource)).should(
                        'have.attr',
                        'aria-label',
                        'forbidden'
                    );
                    cy.get(getWriteAccessIconForResource(resource)).should(
                        'have.attr',
                        'aria-label',
                        'forbidden'
                    );
                    cy.get(getAccessLevelSelectForResource(resource)).should(
                        'contain',
                        'No Access'
                    );
                }
            });
        });

        cy.get(getReadAccessIconForResource('Cluster')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getWriteAccessIconForResource('Cluster')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getAccessLevelSelectForResource('Cluster')).should(
            'contain',
            'Read and Write Access'
        );

        cy.get(getReadAccessIconForResource('Administration')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getWriteAccessIconForResource('Administration')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getAccessLevelSelectForResource('Administration')).should(
            'contain',
            'Read and Write Access'
        );
    });

    it('displays message instead of form if entity id does not exist', () => {
        const entityId = 'bogus';
        visitAccessControlEntity(entitiesKey, entityId);

        assertAccessControlEntityDoesNotExist(entitiesKey);
    });
});
