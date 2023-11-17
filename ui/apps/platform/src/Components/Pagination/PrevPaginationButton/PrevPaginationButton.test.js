import React, { useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';

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

const options = {
    name: 'Go to previous page', // aria-label attribute
};

test('can not press the previous button when on the first page', async () => {
    render(<MockPagination defaultPage={1} />);
    const button = screen.getByRole('button', options);

    // button should be disabled
    expect(button).toBeDisabled();
});

test('can press the previous button when on the last page', async () => {
    render(<MockPagination defaultPage={5} />);
    const button = screen.getByRole('button', options);

    // button should not be disabled
    expect(button).toBeEnabled();
});

test('pressing the button decreases the page count', async () => {
    const currentPage = 3;
    render(<MockPagination defaultPage={currentPage} />);
    const button = screen.getByRole('button', options);
    const input = screen.getByTestId('pagination-input');

    fireEvent.click(button);

    expect(input.value).toEqual(`${currentPage - 1}`);
});
