import React from 'react';
import { render, fireEvent, wait } from '@testing-library/react';

import Comment from './Comment';

test('should not save on an empty comment', async () => {
    const comment = {
        createdTime: new Date().toISOString(),
        message: ''
    };
    function doNothing() {}
    const { getByTestId, getByText } = render(
        <Comment
            comment={comment}
            onSave={doNothing}
            onClose={doNothing}
            onRemove={doNothing}
            defaultEdit
        />
    );
    const textarea = getByTestId('comment-textarea');
    const saveButton = getByText('Save');

    fireEvent.change(textarea, { target: { value: '   ' } });
    fireEvent.click(saveButton);

    await wait(() => expect(getByText('This field is required')).toBeInTheDocument());
});

test('should show links for urls with http(s) as a prefix', async () => {
    const comment = {
        createdTime: new Date().toISOString(),
        message: 'Here is a link: https://www.example.com'
    };
    function doNothing() {}
    const { getByTestId } = render(
        <Comment comment={comment} onSave={doNothing} onClose={doNothing} onRemove={doNothing} />
    );

    await wait(() =>
        expect(getByTestId('comment-link')).toHaveAttribute('href', 'https://www.example.com')
    );
});

test('should not show links for urls with non-http(s) as a prefix', async () => {
    const comment = {
        createdTime: new Date().toISOString(),
        message: 'These are not links: www.example3.com, example4.com'
    };
    function doNothing() {}
    const { queryByTestId } = render(
        <Comment comment={comment} onSave={doNothing} onClose={doNothing} onRemove={doNothing} />
    );

    expect(queryByTestId('comment-link')).toBeNull();
});
