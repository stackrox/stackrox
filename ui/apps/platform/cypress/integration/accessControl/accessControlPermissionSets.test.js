import { permissionSetsUrl, selectors } from '../../constants/AccessControlPage';
import {
    permissions as permissionsApi,
    permissionSets as permissionSetsApi,
} from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

const h1 = 'Access Control';
const h2 = 'Permission sets';

const defaultNames = ['Admin', 'Analyst', 'Continuous Integration', 'None', 'Sensor Creator'];

describe('Access Control Permission sets', () => {
    withAuth();

    function visitPermissionSets() {
        cy.intercept('GET', permissionSetsApi.list).as('GetPermissionSets');
        cy.visit(permissionSetsUrl);
        cy.wait('@GetPermissionSets');
    }

    function visitPermissionSet(id) {
        cy.intercept('GET', permissionSetsApi.list).as('GetPermissionSets');
        cy.visit(`${permissionSetsUrl}/${id}`);
        cy.wait('@GetPermissionSets');
    }

    it('displays alert if no permission', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(permissionSetsUrl);
        cy.wait('@GetMyPermissions');

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLink).should('not.exist');

        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.alertTitle).should(
            'contain', // not have.text because it contains "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has headings, link, button, and table head cells, and no breadcrumbs', () => {
        visitPermissionSets();

        cy.get(selectors.breadcrumbNav).should('not.exist');

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.contains(selectors.h2, /^\d+ results? found$/).should('exist');
        cy.get(selectors.list.createButton).should('have.text', 'Create permission set');

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Description")`);
        cy.get(`${selectors.list.th}:contains("Roles")`);
    });

    it('list has default names', () => {
        visitPermissionSets();

        defaultNames.forEach((name) => {
            cy.get(`${selectors.list.tdNameLink}:contains("${name}")`);
        });
    });

    it('list link for default Admin goes to form which has label instead of button and disabled input values', () => {
        visitPermissionSets();

        const name = 'Admin';
        cy.get(`${selectors.list.tdNameLink}:contains("${name}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${name}")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', name);
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
        visitPermissionSet('io.stackrox.authz.permissionset.admin');

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

    it('direct link to default Analyst has all (but DebugLogs) read and no write access', () => {
        visitPermissionSet('io.stackrox.authz.permissionset.analyst');

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
                if (resource === 'DebugLogs') {
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
        visitPermissionSet('io.stackrox.authz.permissionset.continuousintegration');

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
                if (!resourcesLimited.includes(resource)) {
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
        visitPermissionSet('io.stackrox.authz.permissionset.none');

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
        visitPermissionSet('io.stackrox.authz.permissionset.sensorcreator');

        cy.get(selectors.form.inputName).should('have.value', 'Sensor Creator');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        const resourcesLimited = ['Cluster', 'ServiceIdentity'];

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', '2');
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', '2');

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                if (!resourcesLimited.includes(resource)) {
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

        cy.get(getReadAccessIconForResource('ServiceIdentity')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getWriteAccessIconForResource('ServiceIdentity')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        cy.get(getAccessLevelSelectForResource('ServiceIdentity')).should(
            'contain',
            'Read and Write Access'
        );
    });

    it('displays message instead of form if entity id does not exist', () => {
        cy.intercept('GET', permissionSetsApi.list).as('GetAuthProviders');
        cy.visit(`${permissionSetsUrl}/bogus`);
        cy.wait('@GetAuthProviders');

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2)`).should('not.exist');

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');
        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.notFound.title).should('have.text', 'Permission set does not exist');
        cy.get(selectors.notFound.a)
            .should('have.text', h2)
            .should('have.attr', 'href', permissionSetsUrl);
    });
});
