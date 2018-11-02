import { MOUSE } from 'three';

// Force layout constants
export const NODE_RADIUS = 16; // radius of nodes
export const MAX_RADIUS = 16; // max radius of individual nodes
export const PADDING = 15; // separation between same namespace nodes
export const CLUSTER_PADDING = 100; // separation between nodes in different namespaces
export const CLUSTER_INNER_PADDING = 60; // minimum separation between node and edge of the namespace
export const CLUSTER_BORDER_PADDING = CLUSTER_INNER_PADDING + 5;
export const SCALE_DURATION = 250;
export const SCALE_FACTOR = 1.5;
export const SCALE_EXTENT = [0.5, 2];

export const NODE_CANVAS_SIZE = 128;
export const NODE_LABEL_FONT_SIZE = 24;
export const NODE_LABEL_OFFSET = 15;
export const NODE_LABEL_CANVAS_SIZE = 256;
export const NAMESPACE_LABEL_FONT_SIZE = 48;
export const NAMESPACE_LABEL_CANVAS_SIZE = 512;
export const NAMESPACE_LABEL_OFFSET = 30;
export const NAMESPACE_BORDER_DASH_WIDTH = [1, 1];
export const NAMESPACE_BORDER_CANVAS_WIDTH = 32;
export const NAMESPACE_BORDER_CANVAS_HEIGHT = 32;
export const INTERNET_ACCESS_ICON_WIDTH = 32;
export const INTERNET_ACCESS_ICON_HEIGHT = 16;
export const INTERNET_ACCESS_ICON_X_OFFSET = 14;
export const INTERNET_ACCESS_ICON_Y_OFFSET = 22;

export const FORCE_CONFIG = {
    FORCE_COLLISION_RADIUS_OFFSET: 20,
    FORCE_CLUSTER_STRENGTH: 0.9
};

export const MIN_ZOOM = 0.25;
export const ZOOM_LEVEL_TO_SHOW_LINKS = 1.5;
export const MAX_ZOOM = 2;
export const ORBIT_CONTROLS_CONFIG = {
    maxZoom: MAX_ZOOM,
    minZoom: MIN_ZOOM,
    enablePan: true,
    enableRotate: false,
    enableDamping: true,
    dampingFactor: 0.12,
    mouseButtons: {
        PAN: MOUSE.LEFT,
        ZOOM: MOUSE.MIDDLE
    }
};

export const RENDERER_CONFIG = {
    antialias: true,
    precision: 'highp',
    alpha: true
};

// Colors
const PRIMARY_COLOR_STRING = '#525966';
export const INTERNET_ACCESS_COLOR = 0xa3deff;
export const NAMESPACE_BORDER_COLOR = '#cacfe5';
export const NAMESPACE_BORDER_RECT_COLOR = '#a2c3e8';
export const NAMESPACE_INTERNET_ACCESS_BORDER_COLOR = '#e4eefc';
export const INGRESS_EGRESS_ICON_BG_COLOR = '#d2e7ff';
export const INGRESS_EGRESS_ICON_COLOR = '#416383';
export const CANVAS_BG_COLOR = '#ffffff';
export const NODE_COLOR = '#7c86b3';
export const INTERNET_ACCESS_NODE_COLOR = '#64a6f0';
export const INTERNET_ACCESS_NODE_BORDER_COLOR = '#c4cdfa';
export const LINK_COLOR = '#C5CEF7';
export const NAMESPACE_LINK_COLOR = 0xbdc2d9;
export const TEXT_COLOR = PRIMARY_COLOR_STRING;
export const NAMESPACE_TEXT_COLOR = '#60689B';

// Network graph object types
export const NETWORK_GRAPH_TYPES = Object.freeze({
    NODE: 'SERVICE',
    LINK: 'SERVICE_LINK',
    NAMESPACE: 'NAMESPACE',
    NAMESPACE_LINK: 'NAMESPACE_LINK'
});

export const ACTIVE_STATE = 0;
export const ALLOWED_STATE = 1;
export const ALL_STATE = 2;

export const TRANSPARENT = 0.05;
export const VISIBLE = 1;

export const NAMESPACE_LINK_WIDTH = 0.005;
export const NODE_LINK_WIDTH = 0.004;
export const NODE_DASH_ARRAY = 0.05;
export const NODE_DASH_OFFSET = 0;
export const NODE_DASH_RATIO = 0.65;
