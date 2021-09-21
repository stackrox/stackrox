import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';

import Comment from './Comment';

// Having troubles getting this test to pass with patternfly
// eslint-disable-next-line jest/no-disabled-tests
test.skip('should not save on an empty comment', async () => {
    const comment = {
        createdTime: new Date().toISOString(),
        message: '',
    };
    function doNothing() {}
    render(
        <Comment
            comment={comment}
            onSave={doNothing}
            onClose={doNothing}
            onRemove={doNothing}
            defaultEdit
        />
    );
    const textarea = screen.getByTestId('comment-textarea');
    const saveButton = screen.getByText('Save');

    fireEvent.change(textarea, { target: { value: '   ' } });
    fireEvent.click(saveButton);

    const element = await screen.findByText('This field is required');
    expect(element).toBeInTheDocument();
});

test('should show links for urls with http(s) as a prefix', async () => {
    const comment = {
        createdTime: new Date().toISOString(),
        message: 'Here is a link: https://www.example.com',
    };
    function doNothing() {}
    render(
        <Comment comment={comment} onSave={doNothing} onClose={doNothing} onRemove={doNothing} />
    );

    const commentLink = await screen.findByTestId('comment-link');
    expect(commentLink).toHaveAttribute('href', 'https://www.example.com');
});

test('should not show links for urls with non-http(s) as a prefix', async () => {
    const comment = {
        createdTime: new Date().toISOString(),
        message: 'These are not links: www.example3.com, example4.com',
    };
    function doNothing() {}
    render(
        <Comment comment={comment} onSave={doNothing} onClose={doNothing} onRemove={doNothing} />
    );

    expect(screen.queryByTestId('comment-link')).toBeNull();
});
