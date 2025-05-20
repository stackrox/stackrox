import React, { useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';

import PaginationInput from './PaginationInput';

const MockPaginationInput = ({ defaultPage = 1 }) => {
    const [currentPage, setPage] = useState(defaultPage);
    const totalSize = 5;
    const pageSize = 1;
    return (
        <PaginationInput
            currentPage={currentPage}
            totalSize={totalSize}
            pageSize={pageSize}
            onChange={setPage}
        />
    );
};

test('cannot set page to a higher value than the total pages count', async () => {
    render(<MockPaginationInput />);
    const input = screen.getByTestId('pagination-input');

    fireEvent.change(input, { target: { value: '10' } });

    expect(input.value).toBe('5');
});

test('cannot set page to a lower value than 1', async () => {
    render(<MockPaginationInput />);
    const input = screen.getByTestId('pagination-input');

    fireEvent.change(input, { target: { value: '0' } });

    expect(input.value).toBe('1');
});
