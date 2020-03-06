import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

import Button from 'Components/Button';

export const NotFoundMessage = ({ message, actionText, onClick, url }) => {
    const buttonClassName =
        'p-4 text-uppercase text-base-100 focus:text-base-100 hover:text-base-100 bg-primary-700 hover:bg-primary-800 no-underline focus:bg-primary-800 inline-block text-center rounded-sm';
    const isButtonVisible = actionText && onClick;
    const isLinkVisible = actionText && url;
    return (
        <div className="text-center flex w-full justify-center items-center p-8 min-h-full bg-primary-200">
            <div>
                <p className="text-tertiary-800 mb-8">{message}</p>
                {isButtonVisible && (
                    <Button className={buttonClassName} text={actionText} onClick={onClick} />
                )}
                {isLinkVisible && (
                    <Link className={buttonClassName} to={url}>
                        {actionText}
                    </Link>
                )}
            </div>
        </div>
    );
};

NotFoundMessage.propTypes = {
    message: PropTypes.oneOfType([PropTypes.string, PropTypes.element]),
    actionText: PropTypes.string,
    onClick: PropTypes.func,
    url: PropTypes.string
};

NotFoundMessage.defaultProps = {
    message: 'This page was not found',
    actionText: null,
    onClick: null,
    url: null
};

export default NotFoundMessage;
