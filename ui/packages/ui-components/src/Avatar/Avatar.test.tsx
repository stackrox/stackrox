import React from 'react';
import { render } from '@testing-library/react';

import Avatar from './Avatar';

describe('Avatar', () => {
    test('shows initials when no image provided', () => {
        const { getByText } = render(<Avatar name="John Smith" extraClassName="my-class" />);
        expect(getByText('JS')).toHaveClass('my-class');
    });

    test('shows default content when neither name nor image provided', () => {
        const { getByText } = render(<Avatar extraClassName="my-class" />);
        expect(getByText('--')).toHaveClass('my-class');
    });

    test('shows image when image provided', () => {
        const { getByAltText } = render(
            <Avatar name="John Smith" imageSrc="url" extraClassName="my-class" />
        );
        expect(getByAltText('JS')).toHaveAttribute('src', 'url');
        expect(getByAltText('JS')).toHaveClass('my-class');
    });
});
