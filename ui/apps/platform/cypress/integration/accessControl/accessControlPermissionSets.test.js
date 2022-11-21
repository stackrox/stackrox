import { permissionSetsUrl, selectors } from '../../constants/AccessControlPage';
import {
    permissions as permissionsApi,
    permissionSets as permissionSetsApi,
} from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import { hasFeatureFlag } from '../../helpers/features';

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

        cy.get(selectors.alertTitle).should(
            'contain', // not have.text because it contains "Info alert:" for screen reader
            'You do not have permission to view permission sets.'
        );
    });

    it('list has headings, link, button, and table head cells, and no breadcrumbs', () => {
        visitPermissionSets();

        // Table has plural noun in title.
        cy.title().should('match', getRegExpForTitleWithBranding(`${h1} - ${h2}`));

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

        // Form has singular noun in title.
        cy.title().should('match', getRegExpForTitleWithBranding(`${h1} - Permission set`));

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
        /*
         * TODO: ROX-13585 - remove the pre-postgres constants once the migration to postgres
         * is completed and the support for BoltDB, RocksDB and Bleve is dropped.
         */
        const targetID = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'ffffffff-ffff-fff4-f5ff-ffffffffffff'
            : 'io.stackrox.authz.permissionset.admin';
        visitPermissionSet(targetID);

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

    // TODO: ROX-12750 Rename DebugLogs to Administration
    it('direct link to default Analyst has all (but DebugLogs) read and no write access', () => {
        /*
         * TODO: ROX-13585 - remove the pre-postgres constants once the migration to postgres
         * is completed and the support for BoltDB, RocksDB and Bleve is dropped.
         */
        const targetID = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'ffffffff-ffff-fff4-f5ff-fffffffffffe'
            : 'io.stackrox.authz.permissionset.analyst';
        visitPermissionSet(targetID);

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
                // TODO: ROX-12750 Rename DebugLogs to Administration
                if (resource.includes('DebugLogs')) {
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
        /*
         * TODO: ROX-13585 - remove the pre-postgres constants once the migration to postgres
         * is completed and the support for BoltDB, RocksDB and Bleve is dropped.
         */
        const targetID = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'ffffffff-ffff-fff4-f5ff-fffffffffffd'
            : 'io.stackrox.authz.permissionset.continuousintegration';
        visitPermissionSet(targetID);

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
                if (!resourcesLimited.some((v) => resource.includes(v))) {
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
        /*
         * TODO: ROX-13585 - remove the pre-postgres constants once the migration to postgres
         * is completed and the support for BoltDB, RocksDB and Bleve is dropped.
         */
        const targetID = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'ffffffff-ffff-fff4-f5ff-fffffffffffc'
            : 'io.stackrox.authz.permissionset.none';
        visitPermissionSet(targetID);

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
        /*
         * TODO: ROX-13585 - remove the pre-postgres constants once the migration to postgres
         * is completed and the support for BoltDB, RocksDB and Bleve is dropped.
         */
        const targetID = hasFeatureFlag('ROX_POSTGRES_DATASTORE')
            ? 'ffffffff-ffff-fff4-f5ff-fffffffffffa'
            : 'io.stackrox.authz.permissionset.sensorcreator';
        visitPermissionSet(targetID);

        cy.get(selectors.form.inputName).should('have.value', 'Sensor Creator');

        const {
            getReadAccessIconForResource,
            getWriteAccessIconForResource,
            getAccessLevelSelectForResource,
        } = selectors.form.permissionSet;

        // TODO: ROX-12750 Rename ServiceIdentity to Administration
        const resourcesLimited = ['Cluster', 'ServiceIdentity'];

        cy.get(selectors.form.permissionSet.tdResource).then(($tds) => {
            const resourceCount = String($tds.length);

            cy.get(selectors.form.permissionSet.resourceCount).should('have.text', resourceCount);
            cy.get(selectors.form.permissionSet.readCount).should('have.text', '2');
            cy.get(selectors.form.permissionSet.writeCount).should('have.text', '2');

            $tds.get().forEach((td) => {
                const resource = td.textContent;
                if (!resourcesLimited.some((v) => resource.includes(v))) {
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

        // TODO: ROX-12750 Rename ServiceIdentity to Administration
        cy.get(getReadAccessIconForResource('ServiceIdentity')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        // TODO: ROX-12750 Rename ServiceIdentity to Administration
        cy.get(getWriteAccessIconForResource('ServiceIdentity')).should(
            'have.attr',
            'aria-label',
            'permitted'
        );
        // TODO: ROX-12750 Rename ServiceIdentity to Administration
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
