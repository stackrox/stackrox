import React, { useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';

import NextPaginationButton from './NextPaginationButton';
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
            <NextPaginationButton
                currentPage={currentPage}
                totalSize={totalSize}
                pageSize={pageSize}
                onChange={setPage}
            />
        </>
    );
};

test('can press the next button when on the first page', async () => {
    render(<MockPagination defaultPage={1} />);
    const button = screen.getByTestId('next-page-button');

    // button should not be disabled
    expect(button).not.toHaveAttribute('disabled');
});

test('can not press the next button when on the last page', async () => {
    render(<MockPagination defaultPage={5} />);
    const button = screen.getByTestId('next-page-button');

    // button should be disabled
    expect(button).toHaveAttribute('disabled');
});

test('pressing the button increases the page count', async () => {
    const currentPage = 1;
    render(<MockPagination defaultPage={currentPage} />);
    const button = screen.getByTestId('next-page-button');
    const input = screen.getByTestId('pagination-input');

    fireEvent.click(button);

    expect(input.value).toEqual(`${currentPage + 1}`);
});
