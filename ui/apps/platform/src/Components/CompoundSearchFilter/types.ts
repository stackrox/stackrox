export type DeepRequired<T> = {
    [P in keyof T]-?: T[P] extends object ? DeepRequired<T[P]> : T[P];
};

export type ImageSearchFilterConfig = {
    displayName: 'Image';
    searchCategory: 'IMAGES';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Image Name';
            searchTerm: 'Image';
            inputType: 'autocomplete';
        };
        OperatingSystem?: {
            displayName: 'Operating System';
            filterChipLabel: 'Image Operating System';
            searchTerm: 'Image OS';
            inputType: 'text';
        };
        Tag?: {
            displayName: 'Tag';
            filterChipLabel: 'Image Tag';
            searchTerm: 'Image Tag';
            inputType: 'text';
        };
        CVSS?: {
            displayName: 'CVSS';
            filterChipLabel: 'Image CVSS';
            searchTerm: 'Image Top CVSS';
            inputType: 'dropdown-slider';
        };
        Label?: {
            displayName: 'Label';
            filterChipLabel: 'Image Label';
            searchTerm: 'Image Label';
            inputType: 'autocomplete';
        };
        CreatedTime?: {
            displayName: 'Created Time';
            filterChipLabel: 'Image Created Time';
            searchTerm: 'Image Created Time';
            inputType: 'date-picker';
        };
        ScanTime?: {
            displayName: 'Scan Time';
            filterChipLabel: 'Image Scan Time';
            searchTerm: 'Image Scan Time';
            inputType: 'date-picker';
        };
        Registry?: {
            displayName: 'Registry';
            filterChipLabel: 'Image Registry';
            searchTerm: 'Image Registry';
            inputType: 'text';
        };
    };
};

export type DeploymentSearchFilterConfig = {
    displayName: 'Deployment';
    searchCategory: 'DEPLOYMENTS';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Deployment Name';
            searchTerm: 'Deployment';
            inputType: 'autocomplete';
        };
        Label?: {
            displayName: 'Label';
            filterChipLabel: 'Deployment Label';
            searchTerm: 'Deployment Label';
            inputType: 'autocomplete';
        };
        Annotation?: {
            displayName: 'Annotation';
            filterChipLabel: 'Deployment Annotation';
            searchTerm: 'Deployment Annotation';
            inputType: 'autocomplete';
        };
    };
};

export type NamespaceSearchFilterConfig = {
    displayName: 'Namespace';
    searchCategory: 'NAMESPACES';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Namespace Name';
            searchTerm: 'Namespace';
            inputType: 'autocomplete';
        };
        Label?: {
            displayName: 'Label';
            filterChipLabel: 'Namespace Label';
            searchTerm: 'Namespace Label';
            inputType: 'autocomplete';
        };
        Annotation?: {
            displayName: 'Annotation';
            filterChipLabel: 'Namespace Annotation';
            searchTerm: 'Namespace Annotation';
            inputType: 'autocomplete';
        };
    };
};

export type ClusterSearchFilterConfig = {
    displayName: 'Cluster';
    searchCategory: 'CLUSTERS';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Cluster Name';
            searchTerm: 'Cluster';
            inputType: 'autocomplete';
        };
        Label?: {
            displayName: 'Label';
            filterChipLabel: 'Cluster Label';
            searchTerm: 'Cluster Label';
            inputType: 'autocomplete';
        };
        Type?: {
            displayName: 'Type';
            filterChipLabel: 'Cluster Type';
            searchTerm: 'Cluster Type';
            inputType: 'autocomplete';
        };
    };
};

export type NodeSearchFilterConfig = {
    displayName: 'Node';
    searchCategory: 'NODES';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Node Name';
            searchTerm: 'Node';
            inputType: 'autocomplete';
        };
        OperatingSystem?: {
            displayName: 'Operating System';
            filterChipLabel: 'Node Operating System';
            searchTerm: 'Operating System';
            inputType: 'text';
        };
        TopCVSS?: {
            displayName: 'Top CVSS';
            filterChipLabel: 'Node Top CVSS';
            searchTerm: 'Node Top CVSS';
            inputType: 'dropdown-slider';
        };
        Label?: {
            displayName: 'Label';
            filterChipLabel: 'Node Label';
            searchTerm: 'Node Label';
            inputType: 'autocomplete';
        };
        Annotation?: {
            displayName: 'Annotation';
            filterChipLabel: 'Node Annotation';
            searchTerm: 'Node Annotation';
            inputType: 'autocomplete';
        };
        JoinTime?: {
            displayName: 'Join Time';
            filterChipLabel: 'Node Join Time';
            searchTerm: 'Node Join Time';
            inputType: 'date-picker';
        };
        ScanTime?: {
            displayName: 'Scan Time';
            filterChipLabel: 'Node Scan Time';
            searchTerm: 'Node Scan Time';
            inputType: 'date-picker';
        };
    };
};

export type ImageCVESearchFilterConfig = {
    displayName: 'Image CVE';
    searchCategory: 'IMAGE_VULNERABILITIES';
    attributes: {
        ID?: {
            displayName: 'ID';
            filterChipLabel: 'Image CVE ID';
            searchTerm: 'CVE ID';
            inputType: 'autocomplete';
        };
        DiscoveredTime?: {
            displayName: 'Discovered Time';
            filterChipLabel: 'Image CVE Discovered Time';
            searchTerm: 'CVE Created Time';
            inputType: 'date-picker';
        };
        CVSS?: {
            displayName: 'CVSS';
            filterChipLabel: 'Image CVE CVSS';
            searchTerm: 'CVSS';
            inputType: 'dropdown-slider';
        };
        Type?: {
            displayName: 'Type';
            filterChipLabel: 'Image CVE Type';
            searchTerm: 'CVE Type';
            inputType: 'select';
        };
    };
};

export type NodeCVESearchFilterConfig = {
    displayName: 'Node CVE';
    searchCategory: 'NOD_VULNERABILITIES';
    attributes: {
        ID?: {
            displayName: 'ID';
            filterChipLabel: 'Node CVE ID';
            searchTerm: 'CVE ID';
            inputType: 'autocomplete';
        };
        DiscoveredTime?: {
            displayName: 'Discovered Time';
            filterChipLabel: 'Node CVE Discovered Time';
            searchTerm: 'CVE Created Time';
            inputType: 'date-picker';
        };
        CVSS?: {
            displayName: 'CVSS';
            filterChipLabel: 'Node CVE CVSS';
            searchTerm: 'CVSS';
            inputType: 'dropdown-slider';
        };
        // TODO: Add Top CVSS
        Snoozed?: {
            displayName: 'Snoozed';
            filterChipLabel: 'Node CVE Snoozed';
            searchTerm: 'CVE Snoozed';
            inputType: 'select';
        };
    };
};

export type PlatformCVESearchFilterConfig = {
    displayName: 'Platform CVE';
    searchCategory: 'CLUSTER_VULNERABILITIES';
    attributes: {
        ID?: {
            displayName: 'ID';
            filterChipLabel: 'Platform CVE ID';
            searchTerm: 'CVE ID';
            inputType: 'autocomplete';
        };
        DiscoveredTime?: {
            displayName: 'Discovered Time';
            filterChipLabel: 'Platform CVE Discovered Time';
            searchTerm: 'CVE Created Time';
            inputType: 'date-picker';
        };
        CVSS?: {
            displayName: 'CVSS';
            filterChipLabel: 'Platform CVE CVSS';
            searchTerm: 'CVSS';
            inputType: 'dropdown-slider';
        };
        Snoozed?: {
            displayName: 'Snoozed';
            filterChipLabel: 'Platform CVE Snoozed';
            searchTerm: 'CVE Snoozed';
            inputType: 'select';
        };
        Type?: {
            displayName: 'Type';
            filterChipLabel: 'Platform CVE Type';
            searchTerm: 'CVE Type';
            inputType: 'select';
        };
    };
};

export type ImageComponentSearchFilterConfig = {
    displayName: 'Image Component';
    searchCategory: 'IMAGE_COMPONENTS';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Image Component Name';
            searchTerm: 'Component';
            inputType: 'autocomplete';
        };
        Source?: {
            displayName: 'Source';
            filterChipLabel: 'Image Component Source';
            searchTerm: 'Component Source';
            inputType: 'select';
        };
        Version?: {
            displayName: 'Version';
            filterChipLabel: 'Image Component Version';
            searchTerm: 'Component Version';
            inputType: 'text';
        };
    };
};

export type NodeComponentSearchFilterConfig = {
    displayName: 'Node Component';
    searchCategory: 'NODE_COMPONENTS';
    attributes: {
        Name?: {
            displayName: 'Name';
            filterChipLabel: 'Node Component Name';
            searchTerm: 'Component';
            inputType: 'autocomplete';
        };
        Source?: {
            displayName: 'Source';
            filterChipLabel: 'Node Component Source';
            searchTerm: 'Component Source';
            inputType: 'select';
        };
        Version?: {
            displayName: 'Version';
            filterChipLabel: 'Node Component Version';
            searchTerm: 'Component Version';
            inputType: 'text';
        };
    };
};

export type CompoundSearchFilterConfig = {
    Image?: ImageSearchFilterConfig;
    Deployment?: DeploymentSearchFilterConfig;
    Namespace?: NamespaceSearchFilterConfig;
    Cluster?: ClusterSearchFilterConfig;
    Node?: NodeSearchFilterConfig;
    ImageCVE?: ImageCVESearchFilterConfig;
    NodeCVE?: NodeCVESearchFilterConfig;
    PlatformCVE?: PlatformCVESearchFilterConfig;
    ImageComponent?: ImageComponentSearchFilterConfig;
    NodeComponent?: NodeComponentSearchFilterConfig;
};
