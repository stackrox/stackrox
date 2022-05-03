// Cytoscape
export const NS_FONT_SIZE = 8;
export const TEXT_MAX_WIDTH = 30;
export const NODE_WIDTH = 8;
export const NODE_PADDING = 20;
export const SIDE_NODE_PADDING = 10;
export const EXTERNAL_NODE_PADDING = 40;
export const NODE_SOLID_BORDER_WIDTH = 2;
export const OUTER_PADDING = 12;
export const OUTER_SPACING_FACTOR = 0.1;

export const GRAPH_PADDING = 50;

// Cytoscape Zoom Constants
export const MAX_ZOOM = 3;
export const MIN_ZOOM = 0.25;
export const ZOOM_STEP = 0.75;

// Colors
export const INGRESS_EGRESS_ICON_COLOR = '#416383';
export const INTERNET_ACCESS_NODE_COLOR = '#64a6f0';
export const INTERNET_ACCESS_NODE_BORDER_COLOR = '#c4cdfa';

export const COLORS = {
    nonIsolated: 'hsla(2, 78%, 71%, 1)',
    active: 'hsla(214, 74%, 68%, 1)',
    externallyConnectedNode: 'hsla(242, 99%, 72%, 1)',
    externallyConnectedBorder: 'hsla(230, 90%, 85%, 1)',
    hoveredActive: 'hsla(214, 74%, 58%, 1)',
    selectedActive: 'hsla(214, 74%, 48%, 1)',
    label: 'hsla(231, 22%, 49%, 1)',
    NSEdge: 'hsla(231, 40%, 74%, 1.00)',
    inactive: 'hsla(229, 24%, 59%, 1)',
    inactiveNS: 'hsla(229, 24%, 80%, 1)',
    hovered: 'hsla(229, 24%, 70%, 1)',
    selected: 'hsla(229, 24%, 60%, 1)',
    hoveredEdge: '#3C58CC',
    edge: '#788CDF',
    simulatedStatus: {
        ADDED: '#47b238',
        REMOVED: '#fc655a',
        UNMODIFIED: '#788CDF',
        MODIFIED: '#e3c987',
    },
    hoveredSimulatedStatus: {
        ADDED: '#2c8820',
        REMOVED: '#c25047',
        UNMODIFIED: '#3C58CC',
        MODIFIED: '#b39956',
    },
};

export const PROTOCOLS = {
    L4_PROTOCOL_TCP: 'L4_PROTOCOL_TCP',
    L4_PROTOCOL_UDP: 'L4_PROTOCOL_UDP',
    L4_PROTOCOL_ANY: 'L4_PROTOCOL_ANY',
};

export const networkTraffic = {
    INGRESS: 'ingress',
    EGRESS: 'egress',
    BIDIRECTIONAL: 'bidirectional',
};

export const networkConnections = {
    ACTIVE: 'active',
    ALLOWED: 'allowed',
    ACTIVE_AND_ALLOWED: 'active/allowed',
};

export const nodeConnectionKeys = {
    INGRESS_ACTIVE: 'ingressActive',
    INGRESS_ALLOWED: 'ingressAllowed',
    EGRESS_ACTIVE: 'egressActive',
    EGRESS_ALLOWED: 'egressAllowed',
};

export const nodeTypes = {
    EXTERNAL_ENTITIES: 'INTERNET',
    CIDR_BLOCK: 'EXTERNAL_SOURCE',
};

export const networkFlowStatus = {
    ANOMALOUS: 'ANOMALOUS',
    BASELINE: 'BASELINE',
    BLOCKED: 'BLOCKED',
};
