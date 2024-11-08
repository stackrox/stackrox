import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';

const CountWidget = ({ title, count, description, linkUrl }) => {
    const countNode = (
        <div
            className={`${
                count > 0 ? `hover:bg-tertiary-200` : ``
            } text-6xl px-5 py-8 h-full flex items-center justify-center`}
        >
            {count}
        </div>
    );
    const descNode = description && <div className="pt-1">{description}</div>;
    return (
        <Widget
            header={title}
            className="bg-base-100"
            bodyClassName="flex-col h-full justify-center text-center"
        >
            {linkUrl ? (
                <Link
                    to={linkUrl}
                    className="no-underline w-full h-full flex flex-col justify-center text-primary-700"
                >
                    {countNode}
                    {descNode}
                </Link>
            ) : (
                <>
                    {countNode}
                    {descNode}
                </>
            )}
        </Widget>
    );
};

CountWidget.propTypes = {
    title: PropTypes.string.isRequired,
    count: PropTypes.number,
    description: PropTypes.string,
    linkUrl: PropTypes.string,
};

CountWidget.defaultProps = {
    description: null,
    linkUrl: null,
    count: 0,
};

export default connect()(CountWidget);
