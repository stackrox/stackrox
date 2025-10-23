export type CveType =
    | 'IMAGE_CVE'
    | 'IMAGE'
    | 'IMAGE_COMPONENT'
    | 'DEPLOYMENT'
    | 'NAMESPACE'
    | 'K8S_CVE'
    | 'ISTIO_CVE'
    | 'NODE_CVE'
    | 'NODE'
    | 'NODE_COMPONENT'
    | 'OPENSHIFT_CVE';
export type CveTypeProps = {
    types: CveType[];
};

const cveTypeMap = {
    IMAGE_CVE: 'Image CVE',
    IMAGE: 'Image CVE',
    IMAGE_COMPONENT: 'Image CVE',
    DEPLOYMENT: 'Image CVE',
    NAMESPACE: 'Image CVE',
    K8S_CVE: 'Kubernetes CVE',
    ISTIO_CVE: 'Istio CVE',
    NODE_CVE: 'Node CVE',
    NODE: 'Node CVE',
    NODE_COMPONENT: 'Node CVE',
    OPENSHIFT_CVE: 'OpenShift CVE',
};

const CveType = ({ types = [] }: CveTypeProps) => {
    const sortedTypes = types.map((x) => cveTypeMap[x] || 'Unknown').sort();

    return (
        <span>
            <div className="flex flex-col">
                {sortedTypes.map((cveType) => (
                    <div className="flex justify-center" key={cveType}>
                        {cveType}
                    </div>
                ))}
            </div>
        </span>
    );
};

export default CveType;
