import React, { useState } from 'react';
import { render, fireEvent } from '@testing-library/react';

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

test('can not set page to a higher value than the total pages count', async () => {
    const { getByTestId } = render(<MockPaginationInput />);
    const input = getByTestId('pagination-input');

    fireEvent.change(input, { target: { value: '10' } });

    expect(input.value).toBe('5');
});

test('can not set page to a lower value than 1', async () => {
    const { getByTestId } = render(<MockPaginationInput />);
    const input = getByTestId('pagination-input');

    fireEvent.change(input, { target: { value: '0' } });

    expect(input.value).toBe('1');
});
