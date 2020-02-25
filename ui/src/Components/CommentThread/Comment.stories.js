import React from 'react';

import Comment from 'Components/CommentThread/Comment';

export default {
    title: 'Comment',
    component: Comment
};

function onSave() {}

function onDelete() {}

export const withEdited = () => {
    const comment = {
        id: '1',
        message: 'This is a link: https://www.google.com',
        email: 'sc@stackrox.com',
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-31T21:21:31.218853651Z',
        canModify: false
    };
    return <Comment comment={comment} onSave={onSave} onDelete={onDelete} />;
};

export const withoutEdited = () => {
    const comment = {
        id: '1',
        message: 'This is a link: https://www.google.com',
        email: 'sc@stackrox.com',
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-29T21:21:31.218853651Z',
        canModify: false
    };
    return <Comment comment={comment} onSave={onSave} onDelete={onDelete} />;
};

export const withAbilityToModify = () => {
    const comment = {
        id: '1',
        message: 'This is a link: https://www.google.com',
        email: 'sc@stackrox.com',
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-29T21:21:31.218853651Z',
        canModify: true
    };
    return <Comment comment={comment} onSave={onSave} onDelete={onDelete} />;
};

export const withoutAbilityToModify = () => {
    const comment = {
        id: '1',
        message: 'This is a link: https://www.google.com',
        email: 'sc@stackrox.com',
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-29T21:21:31.218853651Z',
        canModify: false
    };
    return <Comment comment={comment} onSave={onSave} onDelete={onDelete} />;
};
