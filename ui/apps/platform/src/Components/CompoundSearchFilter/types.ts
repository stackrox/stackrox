import { ValueOf } from 'utils/type.utils';

export type SearchFilterConfig = {
    displayName: string;
    searchCategory: string;
    attributes: Record<string, SearchFilterAttribute>;
};

export type SearchFilterAttribute = {
    displayName: string;
    filterChipLabel: string;
    searchTerm: string;
    inputType: string;
};

// Image search filter

export const imageSearchFilterConfig = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Image Name',
            searchTerm: 'Image',
            inputType: 'autocomplete',
        },
        OperatingSystem: {
            displayName: 'Operating System',
            filterChipLabel: 'Image Operating System',
            searchTerm: 'Image OS',
            inputType: 'text',
        },
        Tag: {
            displayName: 'Tag',
            filterChipLabel: 'Image Tag',
            searchTerm: 'Image Tag',
            inputType: 'text',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'Image CVSS',
            searchTerm: 'Image Top CVSS',
            inputType: 'dropdown-slider',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Image Label',
            searchTerm: 'Image Label',
            inputType: 'autocomplete',
        },
        CreatedTime: {
            displayName: 'Created Time',
            filterChipLabel: 'Image Created Time',
            searchTerm: 'Image Created Time',
            inputType: 'date-picker',
        },
        ScanTime: {
            displayName: 'Scan Time',
            filterChipLabel: 'Image Scan Time',
            searchTerm: 'Image Scan Time',
            inputType: 'date-picker',
        },
        Registry: {
            displayName: 'Registry',
            filterChipLabel: 'Image Registry',
            searchTerm: 'Image Registry',
            inputType: 'text',
        },
    },
} as const;

export type ImageSearchFilterConfig = {
    displayName: (typeof imageSearchFilterConfig)['displayName'];
    searchCategory: (typeof imageSearchFilterConfig)['searchCategory'];
    attributes: (typeof imageSearchFilterConfig)['attributes'];
};

export type ImageAttribute = keyof ImageSearchFilterConfig['attributes'];

// Deployment search filter

export const deploymentSearchFilterConfig = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Deployment Name',
            searchTerm: 'Deployment',
            inputType: 'autocomplete',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Deployment Label',
            searchTerm: 'Deployment Label',
            inputType: 'autocomplete',
        },
        Annotation: {
            displayName: 'Annotation',
            filterChipLabel: 'Deployment Annotation',
            searchTerm: 'Deployment Annotation',
            inputType: 'autocomplete',
        },
    },
} as const;

export type DeploymentSearchFilterConfig = {
    displayName: (typeof deploymentSearchFilterConfig)['displayName'];
    searchCategory: (typeof deploymentSearchFilterConfig)['searchCategory'];
    attributes: (typeof deploymentSearchFilterConfig)['attributes'];
};

export type DeploymentAttribute = keyof DeploymentSearchFilterConfig['attributes'];

// Namespace search filter

export const namespaceSearchFilterConfig = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Namespace Name',
            searchTerm: 'Namespace',
            inputType: 'autocomplete',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Namespace Label',
            searchTerm: 'Namespace Label',
            inputType: 'autocomplete',
        },
        Annotation: {
            displayName: 'Annotation',
            filterChipLabel: 'Namespace Annotation',
            searchTerm: 'Namespace Annotation',
            inputType: 'autocomplete',
        },
    },
} as const;

export type NamespaceSearchFilterConfig = {
    displayName: (typeof namespaceSearchFilterConfig)['displayName'];
    searchCategory: (typeof namespaceSearchFilterConfig)['searchCategory'];
    attributes: (typeof namespaceSearchFilterConfig)['attributes'];
};

export type NamespaceAttribute = keyof NamespaceSearchFilterConfig['attributes'];

// Cluster search filter

export const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Cluster Name',
            searchTerm: 'Cluster',
            inputType: 'autocomplete',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Cluster Label',
            searchTerm: 'Cluster Label',
            inputType: 'autocomplete',
        },
        Type: {
            displayName: 'Type',
            filterChipLabel: 'Cluster Type',
            searchTerm: 'Cluster Type',
            inputType: 'autocomplete',
        },
    },
} as const;

export type ClusterSearchFilterConfig = {
    displayName: (typeof clusterSearchFilterConfig)['displayName'];
    searchCategory: (typeof clusterSearchFilterConfig)['searchCategory'];
    attributes: (typeof clusterSearchFilterConfig)['attributes'];
};

export type ClusterAttribute = keyof ClusterSearchFilterConfig['attributes'];

// Node search filter

export const nodeSearchFilterConfig = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Node Name',
            searchTerm: 'Node',
            inputType: 'autocomplete',
        },
        OperatingSystem: {
            displayName: 'Operating System',
            filterChipLabel: 'Node Operating System',
            searchTerm: 'Operating System',
            inputType: 'text',
        },
        TopCVSS: {
            displayName: 'Top CVSS',
            filterChipLabel: 'Node Top CVSS',
            searchTerm: 'Node Top CVSS',
            inputType: 'dropdown-slider',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Node Label',
            searchTerm: 'Node Label',
            inputType: 'autocomplete',
        },
        Annotation: {
            displayName: 'Annotation',
            filterChipLabel: 'Node Annotation',
            searchTerm: 'Node Annotation',
            inputType: 'autocomplete',
        },
        JoinTime: {
            displayName: 'Join Time',
            filterChipLabel: 'Node Join Time',
            searchTerm: 'Node Join Time',
            inputType: 'date-picker',
        },
        ScanTime: {
            displayName: 'Scan Time',
            filterChipLabel: 'Node Scan Time',
            searchTerm: 'Node Scan Time',
            inputType: 'date-picker',
        },
    },
} as const;

export type NodeSearchFilterConfig = {
    displayName: (typeof nodeSearchFilterConfig)['displayName'];
    searchCategory: (typeof nodeSearchFilterConfig)['searchCategory'];
    attributes: (typeof nodeSearchFilterConfig)['attributes'];
};

export type NodeAttribute = keyof NodeSearchFilterConfig['attributes'];

// Image CVE search filter

export const imageCVESearchFilterConfig = {
    displayName: 'Image CVE',
    searchCategory: 'IMAGE_VULNERABILITIES',
    attributes: {
        ID: {
            displayName: 'ID',
            filterChipLabel: 'Image CVE ID',
            searchTerm: 'CVE ID',
            inputType: 'autocomplete',
        },
        DiscoveredTime: {
            displayName: 'Discovered Time',
            filterChipLabel: 'Image CVE Discovered Time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'Image CVE CVSS',
            searchTerm: 'CVSS',
            inputType: 'dropdown-slider',
        },
        Type: {
            displayName: 'Type',
            filterChipLabel: 'Image CVE Type',
            searchTerm: 'CVE Type',
            inputType: 'select',
        },
    },
} as const;

export type ImageCVESearchFilterConfig = {
    displayName: (typeof imageCVESearchFilterConfig)['displayName'];
    searchCategory: (typeof imageCVESearchFilterConfig)['searchCategory'];
    attributes: (typeof imageCVESearchFilterConfig)['attributes'];
};

export type ImageCVEAttribute = keyof ImageCVESearchFilterConfig['attributes'];

// Node CVE search filter

export const nodeCVESearchFilterConfig = {
    displayName: 'Node CVE',
    searchCategory: 'NODE_VULNERABILITIES',
    attributes: {
        ID: {
            displayName: 'ID',
            filterChipLabel: 'Node CVE ID',
            searchTerm: 'CVE ID',
            inputType: 'autocomplete',
        },
        DiscoveredTime: {
            displayName: 'Discovered Time',
            filterChipLabel: 'Node CVE Discovered Time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'Node CVE CVSS',
            searchTerm: 'CVSS',
            inputType: 'dropdown-slider',
        },
        // TODO: Add Top CVSS
        Snoozed: {
            displayName: 'Snoozed',
            filterChipLabel: 'Node CVE Snoozed',
            searchTerm: 'CVE Snoozed',
            inputType: 'select',
        },
    },
} as const;

export type NodeCVESearchFilterConfig = {
    displayName: (typeof nodeCVESearchFilterConfig)['displayName'];
    searchCategory: (typeof nodeCVESearchFilterConfig)['searchCategory'];
    attributes: (typeof nodeCVESearchFilterConfig)['attributes'];
};

export type NodeCVEAttribute = keyof NodeCVESearchFilterConfig['attributes'];

// Platform CVE search filter

export const platformCVESearchFilterConfig = {
    displayName: 'Platform CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES',
    attributes: {
        ID: {
            displayName: 'ID',
            filterChipLabel: 'Platform CVE ID',
            searchTerm: 'CVE ID',
            inputType: 'autocomplete',
        },
        DiscoveredTime: {
            displayName: 'Discovered Time',
            filterChipLabel: 'Platform CVE Discovered Time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'Platform CVE CVSS',
            searchTerm: 'CVSS',
            inputType: 'dropdown-slider',
        },
        Snoozed: {
            displayName: 'Snoozed',
            filterChipLabel: 'Platform CVE Snoozed',
            searchTerm: 'CVE Snoozed',
            inputType: 'select',
        },
        Type: {
            displayName: 'Type',
            filterChipLabel: 'Platform CVE Type',
            searchTerm: 'CVE Type',
            inputType: 'select',
        },
    },
} as const;

export type PlatformCVESearchFilterConfig = {
    displayName: (typeof platformCVESearchFilterConfig)['displayName'];
    searchCategory: (typeof platformCVESearchFilterConfig)['searchCategory'];
    attributes: (typeof platformCVESearchFilterConfig)['attributes'];
};

export type PlatformCVEAttribute = keyof PlatformCVESearchFilterConfig['attributes'];

// Image Component search filter

export const imageComponentSearchFilterConfig = {
    displayName: 'Image Component',
    searchCategory: 'IMAGE_COMPONENTS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Image Component Name',
            searchTerm: 'Component',
            inputType: 'autocomplete',
        },
        Source: {
            displayName: 'Source',
            filterChipLabel: 'Image Component Source',
            searchTerm: 'Component Source',
            inputType: 'select',
        },
        Version: {
            displayName: 'Version',
            filterChipLabel: 'Image Component Version',
            searchTerm: 'Component Version',
            inputType: 'text',
        },
    },
} as const;

export type ImageComponentSearchFilterConfig = {
    displayName: (typeof imageComponentSearchFilterConfig)['displayName'];
    searchCategory: (typeof imageComponentSearchFilterConfig)['searchCategory'];
    attributes: (typeof imageComponentSearchFilterConfig)['attributes'];
};

export type ImageComponentAttribute = keyof ImageComponentSearchFilterConfig['attributes'];

// Node Component search filter

export const nodeComponentSearchFilterConfig = {
    displayName: 'Node Component',
    searchCategory: 'NODE_COMPONENTS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Node Component Name',
            searchTerm: 'Component',
            inputType: 'autocomplete',
        },
        Source: {
            displayName: 'Source',
            filterChipLabel: 'Node Component Source',
            searchTerm: 'Component Source',
            inputType: 'select',
        },
        Version: {
            displayName: 'Version',
            filterChipLabel: 'Node Component Version',
            searchTerm: 'Component Version',
            inputType: 'text',
        },
    },
} as const;

export type NodeComponentSearchFilterConfig = {
    displayName: (typeof nodeComponentSearchFilterConfig)['displayName'];
    searchCategory: (typeof nodeComponentSearchFilterConfig)['searchCategory'];
    attributes: (typeof nodeComponentSearchFilterConfig)['attributes'];
};

export type NodeComponentAttribute = keyof NodeComponentSearchFilterConfig['attributes'];

// Compound search filter config

export type CompoundSearchFilterConfig = {
    Image: ImageSearchFilterConfig;
    Deployment: DeploymentSearchFilterConfig;
    Namespace: NamespaceSearchFilterConfig;
    Cluster: ClusterSearchFilterConfig;
    Node: NodeSearchFilterConfig;
    ImageCVE: ImageCVESearchFilterConfig;
    NodeCVE: NodeCVESearchFilterConfig;
    PlatformCVE: PlatformCVESearchFilterConfig;
    ImageComponent: ImageComponentSearchFilterConfig;
    NodeComponent: NodeComponentSearchFilterConfig;
};

export type SearchFilterEntityName = keyof CompoundSearchFilterConfig;

export type EntitySearchFilterConfig = ValueOf<Required<CompoundSearchFilterConfig>>;

export type SearchFilterAttributeName =
    | ImageAttribute
    | DeploymentAttribute
    | ClusterAttribute
    | NodeAttribute
    | ImageCVEAttribute
    | NodeCVEAttribute
    | PlatformCVEAttribute
    | ImageComponentAttribute;
