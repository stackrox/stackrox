import React from 'react';
import PropTypes from 'prop-types';

import Panel from 'Components/Panel';
import TableHeader from 'Components/TableHeader';
import TablePagination from 'Components/TablePaginationV2';
import { pageSize } from 'Components/Table';
import ImagesTable from './ImagesTable';

function ImagesTablePanel({
    currentPage,
    setCurrentPage,
    currentImages,
    selectedImageId,
    setSelectedImageId,
    imagesCount,
    isViewFiltered,
    setSortOption
}) {
    const pageCount = Math.ceil(imagesCount / pageSize);
    const paginationComponent = (
        <TablePagination pageCount={pageCount} page={currentPage} setPage={setCurrentPage} />
    );
    const headerComponent = (
        <TableHeader length={imagesCount} type="Image" isViewFiltered={isViewFiltered} />
    );

    return (
        <Panel headerTextComponent={headerComponent} headerComponents={paginationComponent}>
            <div className="h-full w-full">
                <ImagesTable
                    currentImages={currentImages}
                    setSelectedImageId={setSelectedImageId}
                    selectedImageId={selectedImageId}
                    setSortOption={setSortOption}
                />
            </div>
        </Panel>
    );
}

ImagesTablePanel.propTypes = {
    currentPage: PropTypes.number.isRequired,
    setCurrentPage: PropTypes.func.isRequired,
    currentImages: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSelectedImageId: PropTypes.func.isRequired,
    imagesCount: PropTypes.number.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    selectedImageId: PropTypes.string,
    setSortOption: PropTypes.func.isRequired
};

ImagesTablePanel.defaultProps = {
    selectedImageId: ''
};

export default ImagesTablePanel;
