export const networkGraphSelectors = {
    graph: '.pf-ri__topology-section .pf-topology-content [data-id="stackrox-graph"]',
    groups: '.pf-ri__topology-section .pf-topology-content [data-id="stackrox-graph"] [data-layer-id="groups"]',
    nodes: '.pf-ri__topology-section .pf-topology-content [data-id="stackrox-graph"] [data-layer-id="default"]',
    toolbar: '.pf-ri__topology-section .pf-topology-content .pf-topology-control-bar',
    toolbarItem:
        '.pf-ri__topology-section .pf-topology-content .pf-topology-control-bar .pf-c-toolbar__item',
    drawer: '.pf-c-drawer__panel',
    drawerTitle: '.pf-c-drawer__panel [data-testid="drawer-title"]',
    drawerSubtitle: '.pf-c-drawer__panel [data-testid="drawer-subtitle"]',
    drawerTabs: '.pf-c-drawer__panel .pf-c-tabs__list',
};
