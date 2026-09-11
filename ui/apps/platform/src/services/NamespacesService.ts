import axios from './instance';

const namespacesUrl = '/v1/namespaces';

export type NamespaceMetadata = {
    id: string;
    name: string;
    clusterId: string;
    clusterName: string;
};

export type Namespace = {
    metadata: NamespaceMetadata;
};

export function fetchNamespaces(): Promise<Namespace[]> {
    return axios
        .get<{ namespaces: Namespace[] }>(namespacesUrl)
        .then((response) => response.data?.namespaces ?? []);
}
