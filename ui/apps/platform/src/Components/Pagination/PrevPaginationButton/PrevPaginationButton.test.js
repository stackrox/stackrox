import React, { useState } from 'react';
import { render, fireEvent } from '@testing-library/react';

import PrevPaginationButton from '.';
import PaginationInput from '../PaginationInput';

const MockPagination = ({ defaultPage = 1 }) => {
    const [currentPage, setPage] = useState(defaultPage);
    const totalSize = 5;
    const pageSize = 1;
    return (
        <>
            <PaginationInput
                currentPage={currentPage}
                totalSize={totalSize}
                pageSize={pageSize}
                onChange={setPage}
            />
            <PrevPaginationButton
                currentPage={currentPage}
                totalSize={totalSize}
                pageSize={pageSize}
                onChange={setPage}
            />
        </>
    );
};

test('can not press the previous button when on the first page', async () => {
    const { getByTestId } = render(<MockPagination defaultPage={1} />);
    const button = getByTestId('prev-page-button');

    // button should be disabled
    expect(button).toHaveAttribute('disabled');
});

test('can press the previous button when on the last page', async () => {
    const { getByTestId } = render(<MockPagination defaultPage={5} />);
    const button = getByTestId('prev-page-button');

    // button should not be disabled
    expect(button).not.toHaveAttribute('disabled');
});

test('pressing the button decreases the page count', async () => {
    const currentPage = 3;
    const { getByTestId } = render(<MockPagination defaultPage={currentPage} />);
    const button = getByTestId('prev-page-button');
    const input = getByTestId('pagination-input');

    fireEvent.click(button);

    expect(input.value).toEqual(`${currentPage - 1}`);
});
