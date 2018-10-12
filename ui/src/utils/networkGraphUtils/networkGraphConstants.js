import { MOUSE } from 'three';

// Force layout constants
export const MAX_RADIUS = 12; // max radius of individual nodes
export const PADDING = 2; // separation between same namespace nodes
export const CLUSTER_PADDING = 150;
export const CLUSTER_INNER_PADDING = 60;
export const CLUSTER_BORDER_PADDING = CLUSTER_INNER_PADDING + 5;
export const SCALE_DURATION = 250;
export const SCALE_FACTOR = 1.5;
export const SCALE_EXTENT = [0.5, 2];

export const NODE_LABEL_SIZE = 65;
export const NODE_LABEL_OFFSET = 15;
export const NAMESPACE_LABEL_SIZE = 200;
export const NAMESPACE_LABEL_OFFSET = 30;
export const NAMESPACE_BORDER_DASH_WIDTH = [1, 1];
export const NAMESPACE_BORDER_CANVAS_WIDTH = 32;
export const NAMESPACE_BORDER_CANVAS_HEIGHT = 32;

export const FORCE_CONFIG = {
    FORCE_COLLISION_RADIUS_OFFSET: 20,
    FORCE_CLUSTER_STRENGTH: 0.999
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
const PRIMARY_COLOR_HEX = 0x5a6fd9;
const PRIMARY_COLOR_STRING = '#525966';
export const INTERNET_ACCESS_COLOR = 0xa3deff;
export const NAMESPACE_BORDER_COLOR = 0xced3ed;
export const NAMESPACE_BORDER_RECT_COLOR = '#a2c3e8';
export const NAMESPACE_INTERNET_ACCESS_BORDER_COLOR = '#e4eefc';
export const NODE_COLOR = '#7c86b3';
export const INTERNET_ACCESS_NODE_COLOR = PRIMARY_COLOR_HEX;
export const LINK_COLOR = PRIMARY_COLOR_HEX;
export const NAMESPACE_LINK_COLOR = 0xbdc2d9;
export const TEXT_COLOR = PRIMARY_COLOR_STRING;

// Network graph object types
export const NETWORK_GRAPH_TYPES = Object.freeze({
    NODE: 'SERVICE',
    LINK: 'SERVICE_LINK',
    NAMESPACE: 'NAMESPACE',
    NAMESPACE_LINK: 'NAMESPACE_LINK'
});

export const TRANSPARENT = 0.05;
export const VISIBLE = 1;

export const NAMESPACE_LINK_WIDTH = 0.005;
