import React from 'react';
import PropTypes from 'prop-types';

import { httpURLPattern, isValidURL } from 'utils/urlUtils';

const CommentMessage = ({ message }) => {
    // split the message by URLs
    return message.split(httpURLPattern).map(word => {
        // create links for each URL string
        if (isValidURL(word)) {
            return (
                // https://mathiasbynens.github.io/rel-noopener/ explains why we add the rel="noopener noreferrer" attribute
                <a
                    href={word}
                    target="_blank"
                    rel="noopener noreferrer"
                    key={word}
                    className="text-primary-700"
                    data-testid="comment-link"
                >
                    {word}
                </a>
            );
        }
        return word;
    });
};

CommentMessage.propTypes = {
    message: PropTypes.string
};

CommentMessage.defaultProps = {
    message: ''
};

export default CommentMessage;
