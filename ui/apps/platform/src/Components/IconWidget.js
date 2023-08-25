import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';

const IconWidget = ({ title, icon, description, linkUrl, loading, textSizeClass }) => {
    const contents = loading ? (
        <Loader />
    ) : (
        <>
            <div className="flex h-full items-end justify-center">
                <img src={icon} alt={title} />
            </div>
            <div
                className={`h-full flex items-start justify-center pt-3 leading-normal ${textSizeClass}`}
            >
                <span>{description}</span>
            </div>
        </>
    );

    return (
        <Widget
            header={title}
            className="bg-base-100"
            bodyClassName="flex-col h-full justify-center text-center text-base-600 px-3 pt-8 pb-3"
        >
            {linkUrl && !loading ? (
                <Link
                    to={linkUrl}
                    className="w-full h-full flex flex-col justify-center text-base-600 break-all"
                >
                    {contents}
                </Link>
            ) : (
                <div className="break-all">{contents}</div>
            )}
        </Widget>
    );
};

IconWidget.propTypes = {
    title: PropTypes.string.isRequired,
    icon: PropTypes.node.isRequired,
    description: PropTypes.string,
    textSizeClass: PropTypes.string,
    linkUrl: PropTypes.string,
    loading: PropTypes.bool,
};

IconWidget.defaultProps = {
    description: null,
    linkUrl: null,
    textSizeClass: 'text-2xl',
    loading: false,
};

export default connect()(IconWidget);
