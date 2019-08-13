import { url, selectors } from '../constants/ConfigManagementPage';
import withAuth from '../helpers/basicAuth';

const renderListAndSidePanel = entity => {
    cy.visit(url.list[entity]);
    cy.get(selectors.tableRows)
        .eq(0)
        .find(selectors.tableCells)
        .eq(1)
        .click({ force: true });
    cy.get(selectors.widgets);
};

const navigateToSingleEntityPage = entity => {
    cy.get(selectors.externalLink).click();
    cy.url().should('contain', url.single[entity]);
};

const hasCountWidgetsFor = entities => {
    entities.forEach(entity => {
        cy.get(`${selectors.countWidgetTitle}:contains('${entity}')`);
    });
};

const hasTabsFor = entities => {
    entities.forEach(entity => {
        cy.get(selectors.groupedTabs)
            .find('div')
            .contains(entity);
    });
};

const hasRelatedEntityFor = entity => {
    cy.get(`${selectors.relatedEntityWidgetTitle}:contains('${entity}')`);
};

const entityCountMatchesTableRows = (listEntity, context) => {
    cy.get(`${selectors.countWidgets}:contains('${listEntity}')`)
        .find(selectors.countWidgetValue)
        .invoke('text')
        .then(count => {
            cy.get(`${selectors.countWidgets}:contains('${listEntity}')`).click();
            cy.get(
                `[data-test-id="${
                    context === 'Page' ? 'panel' : 'side-panel'
                }"] [data-test-id="panel-header"]`
            )
                .invoke('text')
                .then(panelHeaderText => {
                    expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(count, 10));
                });
        });
};

const pageEntityCountMatchesTableRows = listEntity => {
    entityCountMatchesTableRows(listEntity, 'Page');
};

const sidePanelEntityCountMatchesTableRows = listEntity => {
    entityCountMatchesTableRows(listEntity, 'Side Panel');
};

const entityListCountMatchesTableLinkCount = entities => {
    cy.get(selectors.tableLinks)
        .contains(entities)
        .invoke('text')
        .then(value => {
            const numEntities = parseInt(value, 10);
            cy.get(selectors.tableLinks)
                .contains(entities)
                .click();
            cy.get('[data-test-id="side-panel"] [data-test-id="panel-header"]')
                .invoke('text')
                .then(panelHeaderText => {
                    expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numEntities, 10));
                });
        });
};

describe('Config Management Entities', () => {
    withAuth();

    context('Policy', () => {
        it('should render the policies list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('policies');
        });

        it('should take you to a policy single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policy');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policy');
            hasCountWidgetsFor(['Deployments']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policy');
            hasTabsFor(['deployments']);
        });

        it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
            context('Page', () => {
                renderListAndSidePanel('policies');
                navigateToSingleEntityPage('policy');
                pageEntityCountMatchesTableRows('Deployments');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('policies');
                sidePanelEntityCountMatchesTableRows('Deployments');
            });
        });
    });

    context('CIS Control', () => {
        it('should render the controls list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('controls');
        });

        it('should take you to a control single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
            hasCountWidgetsFor(['Nodes']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
            hasTabsFor(['nodes']);
        });

        it('should have the same number of Nodes in the count widget as in the Nodes table', () => {
            context('Page', () => {
                renderListAndSidePanel('controls');
                navigateToSingleEntityPage('control');
                pageEntityCountMatchesTableRows('Nodes');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('controls');
                sidePanelEntityCountMatchesTableRows('Nodes');
            });
        });
    });

    context('Cluster', () => {
        it('should render the clusters list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('clusters');
        });

        it('should take you to a cluster single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
        });

        // @TODO: Fix this test
        xit('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            hasCountWidgetsFor([
                'Nodes',
                'Namespaces',
                'Deployments',
                'Images',
                'Secrets',
                'Users & Groups',
                'Service Accounts',
                'Roles',
                'Policies',
                'Controls'
            ]);
        });

        // @TODO: Fix this test
        xit('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            hasTabsFor([
                'nodes',
                'namespaces',
                'deployments',
                'images',
                'secrets',
                'users and groups',
                'service accounts',
                'roles',
                'policies',
                'controls'
            ]);
        });

        it('should have the same number of Nodes in the count widget as in the Nodes table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Nodes');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Nodes');
            });
        });

        it('should have the same number of Namespaces in the count widget as in the Namespaces table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Namespaces');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Namespaces');
            });
        });

        it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Deployments');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Deployments');
            });
        });

        it('should have the same number of Images in the count widget as in the Images table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Images');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Images');
            });
        });

        it('should have the same number of Users & Groups in the count widget as in the Users & Groups table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Users & Groups');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Users & Groups');
            });
        });

        it('should have the same number of Service Accounts in the count widget as in the Service Accounts table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Service Accounts');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Service Accounts');
            });
        });

        it('should have the same number of Roles in the count widget as in the Roles table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Roles');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Roles');
            });
        });

        it('should have the same number of Policies in the count widget as in the Policies table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Policies');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Policies');
            });
        });

        // @TODO: Fix this test
        xit('should have the same number of Controls in the count widget as in the Controls table', () => {
            context('Page', () => {
                renderListAndSidePanel('clusters');
                navigateToSingleEntityPage('cluster');
                pageEntityCountMatchesTableRows('Controls');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('clusters');
                sidePanelEntityCountMatchesTableRows('Controls');
            });
        });

        it('should open the side panel to show the same number of Users & Groups when the Users & Groups link is clicked', () => {
            cy.visit(url.list.clusters);
            entityListCountMatchesTableLinkCount('Users & Groups');
        });

        it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
            cy.visit(url.list.clusters);
            entityListCountMatchesTableLinkCount('Service Accounts');
        });

        it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
            cy.visit(url.list.clusters);
            entityListCountMatchesTableLinkCount('Roles');
        });
    });

    context('Namespace', () => {
        it('should render the namespaces list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('namespaces');
        });

        it('should take you to a namespace single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
        });

        it('should show the related cluster widget', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            hasRelatedEntityFor('Cluster');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            hasCountWidgetsFor(['Deployments', 'Secrets', 'Images', 'Policies']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            hasTabsFor(['deployments', 'secrets', 'images', 'policies']);
        });

        it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
            context('Page', () => {
                renderListAndSidePanel('namespaces');
                navigateToSingleEntityPage('namespace');
                pageEntityCountMatchesTableRows('Deployments');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('namespaces');
                sidePanelEntityCountMatchesTableRows('Deployments');
            });
        });

        it('should have the same number of Secrets in the count widget as in the Secrets table', () => {
            context('Page', () => {
                renderListAndSidePanel('namespaces');
                navigateToSingleEntityPage('namespace');
                pageEntityCountMatchesTableRows('Secrets');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('namespaces');
                sidePanelEntityCountMatchesTableRows('Secrets');
            });
        });

        it('should have the same number of Policies in the count widget as in the Policies table', () => {
            context('Page', () => {
                renderListAndSidePanel('namespaces');
                navigateToSingleEntityPage('namespace');
                pageEntityCountMatchesTableRows('Policies');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('namespaces');
                sidePanelEntityCountMatchesTableRows('Policies');
            });
        });

        it('should have the same number of Images in the count widget as in the Images table', () => {
            context('Page', () => {
                renderListAndSidePanel('namespaces');
                navigateToSingleEntityPage('namespace');
                pageEntityCountMatchesTableRows('Images');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('namespaces');
                sidePanelEntityCountMatchesTableRows('Images');
            });
        });

        it('should open the side panel to show the same number of Users & Groups when the Users & Groups link is clicked', () => {
            cy.visit(url.list.clusters);
            entityListCountMatchesTableLinkCount('Users & Groups');
        });

        it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
            cy.visit(url.list.clusters);
            entityListCountMatchesTableLinkCount('Service Accounts');
        });

        it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
            cy.visit(url.list.clusters);
            entityListCountMatchesTableLinkCount('Roles');
        });
    });

    context('Node', () => {
        it('should render the nodes list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('nodes');
        });

        it('should take you to a nodes single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
        });

        // @TODO: Fix this test
        xit('should show the related cluster widget', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            hasRelatedEntityFor('cluster');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            hasCountWidgetsFor(['Controls']);
        });

        // @TODO: Fix this test
        xit('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            hasTabsFor(['controls']);
        });

        it('should have the same number of Controls in the count widget as in the Controls table', () => {
            context('Page', () => {
                renderListAndSidePanel('nodes');
                navigateToSingleEntityPage('node');
                pageEntityCountMatchesTableRows('Controls');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('nodes');
                sidePanelEntityCountMatchesTableRows('Controls');
            });
        });
    });

    context('Deployment', () => {
        it('should render the deployments list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('deployments');
        });

        it('should take you to a deployments single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
        });

        it('should show the related cluster, namespace, and service account widgets', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            hasRelatedEntityFor('Cluster');
            hasRelatedEntityFor('Namespace');
            hasRelatedEntityFor('Service Account');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            hasCountWidgetsFor(['Images', 'Policies']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            hasTabsFor(['images', 'policies']);
        });

        // @TODO: Fix this test
        xit('should have the same number of Images in the count widget as in the Images table', () => {
            context('Page', () => {
                renderListAndSidePanel('deployments');
                navigateToSingleEntityPage('deployment');
                pageEntityCountMatchesTableRows('Images');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('deployments');
                sidePanelEntityCountMatchesTableRows('Images');
            });
        });

        it('should have the same number of Policies in the count widget as in the Policies table', () => {
            context('Page', () => {
                renderListAndSidePanel('deployments');
                navigateToSingleEntityPage('deployment');
                pageEntityCountMatchesTableRows('Policies');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('deployments');
                sidePanelEntityCountMatchesTableRows('Policies');
            });
        });
    });

    context('Image', () => {
        it('should render the images list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('images');
        });

        it('should take you to a images single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
            hasCountWidgetsFor(['Deployments']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
            hasTabsFor(['deployments']);
        });

        it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
            context('Page', () => {
                renderListAndSidePanel('images');
                navigateToSingleEntityPage('image');
                pageEntityCountMatchesTableRows('Deployments');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('images');
                sidePanelEntityCountMatchesTableRows('Deployments');
            });
        });
    });

    context('Secret', () => {
        it('should render the secrets list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('secrets');
        });

        it('should take you to a secrets single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
        });

        // @TODO: Fix this test
        xit('should show the related namespace widget', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
            hasRelatedEntityFor('namespace');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
            hasCountWidgetsFor(['Deployments']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('secrets');
            navigateToSingleEntityPage('secret');
            hasTabsFor(['deployments']);
        });

        it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
            context('Page', () => {
                renderListAndSidePanel('secrets');
                navigateToSingleEntityPage('secret');
                pageEntityCountMatchesTableRows('Deployments');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('secrets');
                sidePanelEntityCountMatchesTableRows('Deployments');
            });
        });
    });

    context('Role', () => {
        it('should render the roles list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('roles');
        });

        it('should take you to a roles single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            hasCountWidgetsFor(['Users & Groups', 'Service Accounts']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            hasTabsFor(['users and groups', 'service accounts']);
        });

        it('should have the same number of Users & Groups in the count widget as in the Users & Groups table', () => {
            context('Page', () => {
                renderListAndSidePanel('roles');
                navigateToSingleEntityPage('role');
                pageEntityCountMatchesTableRows('Users & Groups');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('roles');
                sidePanelEntityCountMatchesTableRows('Users & Groups');
            });
        });

        it('should have the same number of Service Accounts in the count widget as in the Service Accounts table', () => {
            context('Page', () => {
                renderListAndSidePanel('roles');
                navigateToSingleEntityPage('role');
                pageEntityCountMatchesTableRows('Service Accounts');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('roles');
                sidePanelEntityCountMatchesTableRows('Service Accounts');
            });
        });
    });

    context('Subject (Users & Groups)', () => {
        it('should render the users & groups list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('subjects');
        });

        it('should take you to a subject single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
        });

        it('should show the related cluster widget', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            hasRelatedEntityFor('Cluster');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            hasCountWidgetsFor(['Roles']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            hasTabsFor(['roles']);
        });

        it('should have the same number of Roles in the count widget as in the Roles table', () => {
            context('Page', () => {
                renderListAndSidePanel('subjects');
                navigateToSingleEntityPage('subject');
                pageEntityCountMatchesTableRows('Roles');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('subjects');
                sidePanelEntityCountMatchesTableRows('Roles');
            });
        });
    });

    context('Service Account', () => {
        it('should render the service accounts list and open the side panel when a row is clicked', () => {
            renderListAndSidePanel('serviceAccounts');
        });

        it('should take you to a service account single when the "navigate away" button is clicked', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
        });

        // @TODO: Fix this test
        xit('should show the related namespace widget', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            hasRelatedEntityFor('namespace');
        });

        it('should have the correct count widgets for a single entity view', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            hasCountWidgetsFor(['Deployments', 'Roles']);
        });

        it('should have the correct tabs for a single entity view', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            hasTabsFor(['deployments', 'roles']);
        });

        it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
            context('Page', () => {
                renderListAndSidePanel('serviceAccounts');
                navigateToSingleEntityPage('serviceAccount');
                pageEntityCountMatchesTableRows('Deployments');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('serviceAccounts');
                sidePanelEntityCountMatchesTableRows('Deployments');
            });
        });

        it('should have the same number of Roles in the count widget as in the Roles table', () => {
            context('Page', () => {
                renderListAndSidePanel('serviceAccounts');
                navigateToSingleEntityPage('serviceAccount');
                pageEntityCountMatchesTableRows('Roles');
            });

            context('Side Panel', () => {
                renderListAndSidePanel('serviceAccounts');
                sidePanelEntityCountMatchesTableRows('Roles');
            });
        });
    });
});
