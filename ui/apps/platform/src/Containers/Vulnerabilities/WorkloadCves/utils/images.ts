/**
 *  Get the display name of an image based on the presence of a tag
 */
export function getImageBaseNameDisplay(
    id: string,
    imageName: {
        remote: string;
        tag: string;
    }
) {
    const { remote, tag } = imageName;
    return tag ? `${remote}:${tag}` : `${remote}@${id}`;
}
