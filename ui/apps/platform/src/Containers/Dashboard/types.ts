// TODO These would be better defined as part of a global graphql request type
export type Namespace = {
    metadata: {
        id: string;
        name: string;
    };
};

export type Cluster = {
    id: string;
    name: string;
    namespaces: Namespace[];
};
