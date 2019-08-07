import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { fetchImage } from 'services/ImagesService';

import ImageDetails from 'Containers/Images/ImageDetails';

// The id in the image objects, and in the URL is not the ID the backend uses to get images.
function getBackendId(imageId) {
    if (imageId) {
        return imageId.split(':')[1];
    }
    return undefined;
}

// Load an image from the backend.
function loadImage(selectedImageId, setSelectedImage, setIsFetchingSelectedImage) {
    // Try to translate the image id to it's backend ID.
    const backendId = getBackendId(selectedImageId);
    if (!backendId) {
        return;
    }

    // isFetching will be reset to false when fetch finishes.
    setIsFetchingSelectedImage(true);
    fetchImage(backendId).then(
        image => {
            setSelectedImage(image);
            setIsFetchingSelectedImage(false);
        },
        () => {
            setSelectedImage(undefined);
            setIsFetchingSelectedImage(false);
        }
    );
}

function ImageSidePanel({ selectedImageId, setSelectedImageId }) {
    const [selectedImage, setSelectedImage] = useState(undefined);
    const [isFetchingSelectedImage, setIsFetchingSelectedImage] = useState(false);

    useEffect(
        () => {
            if (!selectedImageId) {
                setSelectedImage(undefined);
                return;
            }
            loadImage(selectedImageId, setSelectedImage, setIsFetchingSelectedImage);
        },
        [selectedImageId, setSelectedImageId, setSelectedImage, setIsFetchingSelectedImage]
    );

    // Only render if we have image data to render.
    if (!selectedImageId) return null;
    return (
        <ImageDetails
            image={selectedImage}
            setSelectedImageId={setSelectedImageId}
            loading={isFetchingSelectedImage}
        />
    );
}

ImageSidePanel.propTypes = {
    selectedImageId: PropTypes.string,
    setSelectedImageId: PropTypes.func.isRequired
};

ImageSidePanel.defaultProps = {
    selectedImageId: undefined
};

export default ImageSidePanel;
