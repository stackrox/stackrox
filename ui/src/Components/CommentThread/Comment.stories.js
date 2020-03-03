import React from 'react';

import Comment from 'Components/CommentThread/Comment';

export default {
    title: 'Comment',
    component: Comment
};

function onSave() {}

function onRemove() {}

export const withEdited = () => {
    const comment = {
        id: '1',
        message: 'This comment was edited!',
        user: {
            id: 'user-id-1',
            name: 'Bob Dylan',
            email: 'bob@gmail.com'
        },
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-31T21:21:31.218853651Z',
        modifiable: false
    };
    return <Comment comment={comment} onSave={onSave} onRemove={onRemove} />;
};

export const withoutEdited = () => {
    const comment = {
        id: '1',
        message: 'This comment was created!',
        user: {
            id: 'user-id-1',
            name: 'Bob Dylan',
            email: 'bob@gmail.com'
        },
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-29T21:21:31.218853651Z',
        modifiable: false
    };
    return <Comment comment={comment} onSave={onSave} onRemove={onRemove} />;
};

export const withAbilityToModify = () => {
    const comment = {
        id: '1',
        message: 'This comment can be modified!',
        user: {
            id: 'user-id-1',
            name: 'Bob Dylan',
            email: 'bob@gmail.com'
        },
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-29T21:21:31.218853651Z',
        modifiable: true
    };
    return <Comment comment={comment} onSave={onSave} onRemove={onRemove} />;
};

export const withURLs = () => {
    const comment = {
        id: '1',
        message: 'This comment has a link: https://www.google.com',
        user: {
            id: 'user-id-1',
            name: 'Bob Dylan',
            email: 'bob@gmail.com'
        },
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-29T21:21:31.218853651Z',
        modifiable: false
    };
    return <Comment comment={comment} onSave={onSave} onRemove={onRemove} />;
};
