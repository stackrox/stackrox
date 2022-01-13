// TODO adapt the rest proto/storage/image.proto

export type ImageName = {
    registry: string;
    remote: string;
    tag: string;
    fullName: string;
};

export type Image = {
    name: string;
    priority: string;
    lastUpdated: string;
    id: string;
    fixableCves: number;
    cves: number;
    created: string;
    components: number;
};
