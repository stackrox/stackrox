import React, { useState } from 'react';
import PropTypes from 'prop-types';

import { getUserName } from 'services/AuthService';

import CommentThread from 'Components/CommentThread';

const defaultComments = [
    {
        id: '1',
        message: 'Completely unrelated, but check out this subreddit https://www.reddit.com/r/aww/',
        user: 'Saif Chaudhry',
        createdTime: '2019-12-29T21:21:31.218853651Z',
        updatedTime: '2019-12-30T21:21:31.218853651Z',
        canModify: true
    },
    {
        id: '2',
        message: 'Oh nice! This is the content I like',
        user: 'Linda Song',
        createdTime: '2019-12-30T21:21:31.218853651Z',
        updatedTime: '2019-12-30T21:21:31.218853651Z',
        canModify: false
    },
    {
        id: '3',
        message: 'Also, do you want to hear a joke?',
        user: 'Saif Chaudhry',
        createdTime: '2019-12-30T22:21:31.218853651Z',
        updatedTime: '2019-12-30T22:21:31.218853651Z',
        canModify: true
    },
    {
        id: '4',
        message: 'No',
        user: 'Linda Song',
        createdTime: '2019-12-31T21:21:31.218853651Z',
        updatedTime: '2019-12-31T21:21:31.218853651Z',
        canModify: false
    }
];

const AnalystComments = ({ className, type }) => {
    const [comments, setComments] = useState(defaultComments);

    const currentUser = getUserName();

    function onSave(comment, message) {
        const newComments = comments.filter(datum => datum.id !== comment.id);
        const newComment = { ...comment, message };
        if (!comment.id) {
            newComment.id = comments.length;
            newComment.createdTime = new Date().toISOString();
        } else {
            newComment.updatedTime = new Date().toISOString();
        }
        newComments.push(newComment);
        setComments(newComments);
    }

    function onDelete(comment) {
        setComments(comments.filter(datum => datum.id !== comment.id));
    }

    return (
        <CommentThread
            className={className}
            type={type}
            currentUser={currentUser}
            comments={comments}
            onSave={onSave}
            onDelete={onDelete}
            defaultOpen
        />
    );
};

AnalystComments.propTypes = {
    type: PropTypes.string.isRequired,
    className: PropTypes.string
};

AnalystComments.defaultProps = {
    className: 'border border-base-400'
};

export default React.memo(AnalystComments);
