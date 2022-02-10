export type ContainerRuntimeInfo = {
    type: ContainerRuntimeType;
    version: string;
};

export type ContainerRuntimeType =
    | 'UNKNOWN_CONTAINER_RUNTIME'
    | 'DOCKER_CONTAINER_RUNTIME'
    | 'CRIO_CONTAINER_RUNTIME';
