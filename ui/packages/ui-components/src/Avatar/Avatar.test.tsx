import React from 'react';
import { render, screen } from '@testing-library/react';

import Avatar from './Avatar';

describe('Avatar', () => {
    test('shows initials when no image provided', () => {
        render(<Avatar name="John Smith" extraClassName="my-class" />);
        expect(screen.getByText('JS')).toHaveClass('my-class');
    });

    test('shows default content when neither name nor image provided', () => {
        render(<Avatar extraClassName="my-class" />);
        expect(screen.getByText('--')).toHaveClass('my-class');
    });

    test('shows image when image provided', () => {
        render(<Avatar name="John Smith" imageSrc="url" extraClassName="my-class" />);
        expect(screen.getByAltText('JS')).toHaveAttribute('src', 'url');
        expect(screen.getByAltText('JS')).toHaveClass('my-class');
    });
});
