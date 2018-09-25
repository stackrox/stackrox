import { MOUSE } from 'three';

// Force layout constants
export const MAX_RADIUS = 12; // max radius of individual nodes
export const PADDING = 2; // separation between same namespace nodes
export const CLUSTER_PADDING = 100;
export const CLUSTER_INNER_PADDING = 25;
export const NAMESPACE_LABEL_OFFSET = 10;
export const SCALE_DURATION = 250;
export const SCALE_FACTOR = 1.5;
export const SCALE_EXTENT = [0.5, 2];
export const SERVICE_LABEL_OFFSET = 15;
export const NODE_LABEL_SIZE = 65;

export const FORCE_CONFIG = {
    FORCE_COLLISION_RADIUS_OFFSET: 20,
    FORCE_CLUSTER_STRENGTH: 0.9
};

export const MIN_ZOOM = 0.25;
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
