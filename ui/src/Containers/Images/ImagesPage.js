import React, { useEffect, useState } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';

import ImagesPageHeader from './ImagesPageHeader';
import ImagesTablePanel from './ImagesTablePanel';
import ImageSidePanel from './ImageSidePanel';

function ImagesPage({
    history,
    location: { search },
    match: {
        params: { imageId }
    }
}) {
    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    // Handle changes in the currently selected image.
    const [selectedImageId, setSelectedImageId] = useState(imageId);

    // Handle changes in the current table page.
    const [currentPage, setCurrentPage] = useState(0);

    // Handle changes in the currently displayed images.
    const [currentImages, setCurrentImages] = useState([]);
    const [sortOption, setSortOption] = useState({ field: 'Image', reversed: false });
    const [imagesCount, setImagesCount] = useState(0);

    // When the selected image changes, update the URL.
    useEffect(
        () => {
            const urlSuffix = selectedImageId ? `/${selectedImageId}` : '';
            history.push({
                pathname: `/main/images${urlSuffix}`,
                search
            });
        },
        [selectedImageId, history, search]
    );

    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
                <ImagesPageHeader
                    currentPage={currentPage}
                    sortOption={sortOption}
                    setCurrentImages={setCurrentImages}
                    setImagesCount={setImagesCount}
                    setSelectedImageId={setSelectedImageId}
                    isViewFiltered={isViewFiltered}
                    setIsViewFiltered={setIsViewFiltered}
                />
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                        <ImagesTablePanel
                            currentPage={currentPage}
                            setCurrentPage={setCurrentPage}
                            currentImages={currentImages}
                            selectedImageId={selectedImageId}
                            setSelectedImageId={setSelectedImageId}
                            imagesCount={imagesCount}
                            isViewFiltered={isViewFiltered}
                            setSortOption={setSortOption}
                        />
                    </div>
                    <ImageSidePanel
                        selectedImageId={selectedImageId}
                        setSelectedImageId={setSelectedImageId}
                    />
                </div>
            </div>
        </section>
    );
}

ImagesPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired
};

export default ImagesPage;
