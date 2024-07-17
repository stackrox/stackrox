/* eslint-disable @typescript-eslint/no-duplicate-type-constituents */
import { sourceTypeLabels, sourceTypes } from 'types/image.proto';
import { DeepPartialByKey, ValueOf } from 'utils/type.utils';

export type SearchFilterConfig = {
    displayName: string;
    searchCategory: string;
    attributes: Record<string, SearchFilterAttribute>;
};

export type BaseSearchFilterAttribute = {
    displayName: string;
    filterChipLabel: string;
    searchTerm: string;
    inputType: SearchFilterAttributeInputType;
};

export interface SelectSearchFilterAttribute extends BaseSearchFilterAttribute {
    inputType: 'select';
    inputProps: {
        options: { label: string; value: string }[];
    };
}

export type SearchFilterAttribute = BaseSearchFilterAttribute | SelectSearchFilterAttribute;

// Image search filter

export const imageSearchFilterConfig = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Image name',
            searchTerm: 'Image',
            inputType: 'autocomplete',
        },
        'Operating system': {
            displayName: 'Operating system',
            filterChipLabel: 'Image operating system',
            searchTerm: 'Image OS',
            inputType: 'autocomplete',
        },
        Tag: {
            displayName: 'Tag',
            filterChipLabel: 'Image tag',
            searchTerm: 'Image Tag',
            inputType: 'text',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Image label',
            searchTerm: 'Image Label',
            inputType: 'autocomplete',
        },
        Registry: {
            displayName: 'Registry',
            filterChipLabel: 'Image registry',
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

export type ImageAttributeInputType = ValueOf<ImageSearchFilterConfig['attributes']>['inputType'];

// Deployment search filter

export const deploymentSearchFilterConfig = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Deployment name',
            searchTerm: 'Deployment',
            inputType: 'autocomplete',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Deployment label',
            searchTerm: 'Deployment Label',
            inputType: 'autocomplete',
        },
        Annotation: {
            displayName: 'Annotation',
            filterChipLabel: 'Deployment annotation',
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

export type DeploymentAttributeInputType = ValueOf<
    DeploymentSearchFilterConfig['attributes']
>['inputType'];

// Namespace search filter

export const namespaceSearchFilterConfig = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Namespace name',
            searchTerm: 'Namespace',
            inputType: 'autocomplete',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Namespace label',
            searchTerm: 'Namespace Label',
            inputType: 'autocomplete',
        },
        Annotation: {
            displayName: 'Annotation',
            filterChipLabel: 'Namespace annotation',
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

export type NamespaceAttributeInputType = ValueOf<
    NamespaceSearchFilterConfig['attributes']
>['inputType'];

// Cluster search filter

export const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Cluster name',
            searchTerm: 'Cluster',
            inputType: 'autocomplete',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Cluster label',
            searchTerm: 'Cluster Label',
            inputType: 'autocomplete',
        },
        Type: {
            displayName: 'Type',
            filterChipLabel: 'Cluster type',
            searchTerm: 'Cluster Type',
            inputType: 'autocomplete',
        },
        'Platform Type': {
            displayName: 'Platform Type',
            filterChipLabel: 'Platform type',
            searchTerm: 'Cluster Platform Type',
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

export type ClusterAttributeInputType = ValueOf<
    ClusterSearchFilterConfig['attributes']
>['inputType'];

// Node search filter

export const nodeSearchFilterConfig = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Node name',
            searchTerm: 'Node',
            inputType: 'autocomplete',
        },
        'Operating system': {
            displayName: 'Operating system',
            filterChipLabel: 'Node operating system',
            searchTerm: 'Operating System',
            inputType: 'text',
        },
        Label: {
            displayName: 'Label',
            filterChipLabel: 'Node label',
            searchTerm: 'Node Label',
            inputType: 'autocomplete',
        },
        Annotation: {
            displayName: 'Annotation',
            filterChipLabel: 'Node annotation',
            searchTerm: 'Node Annotation',
            inputType: 'autocomplete',
        },
        'Scan time': {
            displayName: 'Scan time',
            filterChipLabel: 'Node scan time',
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

export type NodeAttributeInputType = ValueOf<NodeSearchFilterConfig['attributes']>['inputType'];

// Image CVE search filter

export const imageCVESearchFilterConfig = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Image CVE',
            searchTerm: 'CVE',
            inputType: 'autocomplete',
        },
        'Discovered time': {
            displayName: 'Discovered time',
            filterChipLabel: 'Image CVE discovered time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'CVSS',
            searchTerm: 'CVSS',
            inputType: 'condition-number',
        },
    },
} as const;

export type ImageCVESearchFilterConfig = {
    displayName: (typeof imageCVESearchFilterConfig)['displayName'];
    searchCategory: (typeof imageCVESearchFilterConfig)['searchCategory'];
    attributes: (typeof imageCVESearchFilterConfig)['attributes'];
};

export type ImageCVEAttribute = keyof ImageCVESearchFilterConfig['attributes'];

export type ImageCVEAttributeInputType = ValueOf<
    ImageCVESearchFilterConfig['attributes']
>['inputType'];

// Node CVE search filter

export const nodeCVESearchFilterConfig = {
    displayName: 'CVE',
    searchCategory: 'NODE_VULNERABILITIES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'CVE',
            searchTerm: 'CVE',
            inputType: 'autocomplete',
        },
        'Discovered time': {
            displayName: 'Discovered time',
            filterChipLabel: 'CVE discovered time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'CVE CVSS',
            searchTerm: 'CVSS',
            inputType: 'condition-number',
        },
        // TODO: Add Top CVSS
    },
} as const;

export type NodeCVESearchFilterConfig = {
    displayName: (typeof nodeCVESearchFilterConfig)['displayName'];
    searchCategory: (typeof nodeCVESearchFilterConfig)['searchCategory'];
    attributes: (typeof nodeCVESearchFilterConfig)['attributes'];
};

export type NodeCVEAttribute = keyof NodeCVESearchFilterConfig['attributes'];

export type NodeCVEAttributeInputType = ValueOf<
    NodeCVESearchFilterConfig['attributes']
>['inputType'];

// Platform CVE search filter

export const platformCVESearchFilterConfig = {
    displayName: 'CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'CVE',
            searchTerm: 'CVE',
            inputType: 'autocomplete',
        },
        'Discovered time': {
            displayName: 'Discovered time',
            filterChipLabel: 'CVE discovered time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        CVSS: {
            displayName: 'CVSS',
            filterChipLabel: 'CVE CVSS',
            searchTerm: 'CVSS',
            inputType: 'condition-number',
        },
        Type: {
            displayName: 'Type',
            filterChipLabel: 'CVE type',
            searchTerm: 'CVE Type',
            inputType: 'select',
            inputProps: {
                options: [
                    { label: 'K8s CVE', value: 'K8S_CVE' },
                    { label: 'Istio CVE', value: 'ISTIO_CVE' },
                    { label: 'Openshift CVE', value: 'OPENSHIFT_CVE' },
                ],
            },
        },
    },
} as const;

export type PlatformCVESearchFilterConfig = {
    displayName: (typeof platformCVESearchFilterConfig)['displayName'];
    searchCategory: (typeof platformCVESearchFilterConfig)['searchCategory'];
    attributes: (typeof platformCVESearchFilterConfig)['attributes'];
};

export type PlatformCVEAttribute = keyof PlatformCVESearchFilterConfig['attributes'];

export type PlatformCVEAttributeInputType = ValueOf<
    PlatformCVESearchFilterConfig['attributes']
>['inputType'];

// Image Component search filter

export const imageComponentSearchFilterConfig = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Image component name',
            searchTerm: 'Component',
            inputType: 'autocomplete',
        },
        Source: {
            displayName: 'Source',
            filterChipLabel: 'Image component source',
            searchTerm: 'Component Source',
            inputType: 'select',
            inputProps: {
                options: sourceTypes.map((sourceType) => {
                    return { label: sourceTypeLabels[sourceType], value: sourceType };
                }),
            },
        },
        Version: {
            displayName: 'Version',
            filterChipLabel: 'Image component version',
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

export type ImageComponentAttributeInputType = ValueOf<
    ImageComponentSearchFilterConfig['attributes']
>['inputType'];

// Node Component search filter

export const nodeComponentSearchFilterConfig = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS',
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Node component name',
            searchTerm: 'Component',
            inputType: 'autocomplete',
        },
        Version: {
            displayName: 'Version',
            filterChipLabel: 'Node component version',
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

export type NodeComponentAttributeInputType = ValueOf<
    NodeComponentSearchFilterConfig['attributes']
>['inputType'];

// Profile Check search filter

export const profileCheckSearchFilterConfig = {
    displayName: 'Profile check',
    searchCategory: 'COMPLIANCE', //@TODO: Update this once we know what to use
    attributes: {
        Name: {
            displayName: 'Name',
            filterChipLabel: 'Profile check name',
            searchTerm: 'Compliance Check Name',
            inputType: 'text',
        },
    },
} as const;

export type ProfileCheckSearchFilterConfig = {
    displayName: (typeof profileCheckSearchFilterConfig)['displayName'];
    searchCategory: (typeof profileCheckSearchFilterConfig)['searchCategory'];
    attributes: (typeof profileCheckSearchFilterConfig)['attributes'];
};

export type ProfileCheckAttribute = keyof ProfileCheckSearchFilterConfig['attributes'];

export type ProfileCheckAttributeInputType = ValueOf<
    ProfileCheckSearchFilterConfig['attributes']
>['inputType'];

// Compliance Rule search filter

export const complianceScanSearchFilterConfig = {
    displayName: 'Compliance scan',
    searchCategory: 'COMPLIANCE', //@TODO: Update this once we know what to use
    attributes: {
        'Config ID': {
            displayName: 'Config ID',
            filterChipLabel: 'Compliance scan config ID',
            searchTerm: 'Compliance Scan Config Id',
            inputType: 'text',
        },
    },
} as const;

export type ComplianceScanSearchFilterConfig = {
    displayName: (typeof complianceScanSearchFilterConfig)['displayName'];
    searchCategory: (typeof complianceScanSearchFilterConfig)['searchCategory'];
    attributes: (typeof complianceScanSearchFilterConfig)['attributes'];
};

export type ComplianceScanAttribute = keyof ComplianceScanSearchFilterConfig['attributes'];

export type ComplianceScanAttributeInputType = ValueOf<
    ComplianceScanSearchFilterConfig['attributes']
>['inputType'];

// Compound search filter config

export const compoundSearchFilter: CompoundSearchFilterConfig = {
    Image: imageSearchFilterConfig,
    Deployment: deploymentSearchFilterConfig,
    Namespace: namespaceSearchFilterConfig,
    Cluster: clusterSearchFilterConfig,
    Node: nodeSearchFilterConfig,
    'Image CVE': imageCVESearchFilterConfig,
    'Node CVE': nodeCVESearchFilterConfig,
    'Platform CVE': platformCVESearchFilterConfig,
    'Image component': imageComponentSearchFilterConfig,
    'Node component': nodeComponentSearchFilterConfig,
    'Profile check': profileCheckSearchFilterConfig,
    'Compliance scan': complianceScanSearchFilterConfig,
};

export type CompoundSearchFilterConfig = {
    Image: ImageSearchFilterConfig;
    Deployment: DeploymentSearchFilterConfig;
    Namespace: NamespaceSearchFilterConfig;
    Cluster: ClusterSearchFilterConfig;
    Node: NodeSearchFilterConfig;
    'Image CVE': ImageCVESearchFilterConfig;
    'Node CVE': NodeCVESearchFilterConfig;
    'Platform CVE': PlatformCVESearchFilterConfig;
    'Image component': ImageComponentSearchFilterConfig;
    'Node component': NodeComponentSearchFilterConfig;
    'Profile check': ProfileCheckSearchFilterConfig;
    'Compliance scan': ComplianceScanSearchFilterConfig;
};

// @TODO: Consider Dave's suggestion about reorganizing and readjusting types (https://github.com/stackrox/stackrox/pull/11349#discussion_r1628428375)
export type PartialCompoundSearchFilterConfig = Partial<
    DeepPartialByKey<CompoundSearchFilterConfig, 'attributes'>
>;

export const compoundSearchEntityNames = Object.keys(compoundSearchFilter);

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
    | ImageComponentAttribute
    | ProfileCheckAttribute
    | ComplianceScanAttribute;

export type SearchFilterAttributeInputType =
    | ImageAttributeInputType
    | DeploymentAttributeInputType
    | ClusterAttributeInputType
    | NodeAttributeInputType
    | ImageCVEAttributeInputType
    | NodeCVEAttributeInputType
    | PlatformCVEAttributeInputType
    | ImageComponentAttributeInputType
    | ProfileCheckAttributeInputType
    | ComplianceScanAttributeInputType;

// Misc

export type OnSearchPayload = {
    action: 'ADD' | 'REMOVE';
    category: string;
    value: string;
};
