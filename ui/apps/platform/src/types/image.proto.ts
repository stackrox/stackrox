// TODO adapt the rest proto/storage/image.proto

export type ImageName = {
    registry: string;
    remote: string;
    tag: string;
    fullName: string;
};
