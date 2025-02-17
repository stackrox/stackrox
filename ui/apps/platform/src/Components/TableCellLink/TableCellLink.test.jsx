import React from 'react';
import { screen } from '@testing-library/react';

import renderWithRouter from 'test-utils/renderWithRouter';
import TableCellLink from './TableCellLink';

describe('TableCellLink', () => {
    it('should render its text in a router link', () => {
        const text = 'rempote';
        const link = '/main/configmanagement/cluster/88d17fde-3b80-48dc-a4f3-1c8068e95f28';

        renderWithRouter(
            <TableCellLink pdf={false} url={link}>
                {text}
            </TableCellLink>
        );

        const el = screen.getByText(text);
        expect(el.href).toContain(link);
    });

    it('should render plain text when PDF flag is set', () => {
        const text = 'rempote';
        const link = '/main/configmanagement/cluster/88d17fde-3b80-48dc-a4f3-1c8068e95f28';

        renderWithRouter(
            <TableCellLink pdf url={link}>
                {text}
            </TableCellLink>
        );

        const el = screen.getByText(text);
        expect(el.href).toBeFalsy();
    });
});
