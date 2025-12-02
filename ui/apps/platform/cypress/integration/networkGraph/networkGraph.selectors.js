const manageCidrBlocksModal = '*[role="dialog"]:has(h1:contains("Manage CIDR blocks"))';

export const networkGraphSelectors = {
    graph: '.pf-ri__topology-section .pf-topology-content [data-id="stackrox-graph"]',
    groups: '.pf-ri__topology-section .pf-topology-content [data-id="stackrox-graph"] [data-layer-id="groups"]',
    nodes: '.pf-ri__topology-section .pf-topology-content [data-id="stackrox-graph"] [data-layer-id="default"]',
    toolbar: '.pf-ri__topology-section .pf-topology-content .pf-topology-control-bar',
    toolbarItem:
        '.pf-ri__topology-section .pf-topology-content .pf-topology-control-bar .pf-v6-c-toolbar__item',
    drawer: '.pf-v6-c-drawer__panel',
    drawerTitle: '.pf-v6-c-drawer__panel [data-testid="drawer-title"]',
    drawerSubtitle: '.pf-v6-c-drawer__panel [data-testid="drawer-subtitle"]',
    drawerTabs: '.pf-v6-c-drawer__panel .pf-v6-c-tabs__list',
    deploymentNode: (deploymentName) =>
        `${networkGraphSelectors.nodes} [data-type="node"] .pf-topology__node__label:contains("${deploymentName}")`,
    // filteredNamespaceGroupNode
    // react-topology@5.2.1: in so-called nodes (default) and has the filtered-namespace class
    // react-topology@5.4.1: moved from so-called nodes to groups and is grand-niece instead of grand-daughter.
    filteredNamespaceGroupNode: (namespace) =>
        `${networkGraphSelectors.groups} g[data-type="group"][data-id="${namespace}"] text:contains("${namespace}")`,
    relatedNamespaceGroupNode: (namespace) =>
        `${networkGraphSelectors.nodes} [data-type="group"] .related-namespace text:contains("${namespace}")`,
    manageCidrBlocksButton: 'button:contains("Manage CIDR blocks")',
    manageCidrBlocksModal,
    manageCidrBlocksModalClose: `${manageCidrBlocksModal} button[aria-label="Close"]`,
    cidrBlockEntryNameInputAt: (index) =>
        `${manageCidrBlocksModal} input[name="entities.${index}.entity.name"]`,
    cidrBlockEntryCidrInputAt: (index) =>
        `${manageCidrBlocksModal} input[name="entities.${index}.entity.cidr"]`,
    cidrBlockEntryDeleteButtonAt: (index) =>
        `${manageCidrBlocksModal} button[name="entities.${index}.entity.delete"]`,
    updateCidrBlocksButton: `${manageCidrBlocksModal} button:contains("Update configuration")`,
    cidrModalAlertWithMessage: (message) =>
        `${manageCidrBlocksModal} .pf-v6-c-alert:contains("${message}")`,
};
