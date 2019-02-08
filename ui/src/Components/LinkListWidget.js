import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';
import Message from 'Components/Message';
import Query from 'Components/ThrowingQuery';

function getLI(item) {
    if (!item) return null;

    const content = item.link ? (
        <Link
            to={item.link}
            className="font-600 text-base-600 leading-normal p-2 inline-block w-full"
        >
            {item.label}
        </Link>
    ) : (
        item.label
    );
    return (
        <li
            key={item.label}
            className="border-b border-base-300 truncate"
            style={{
                columnBreakInside: 'avoid',
                pageBreakInside: 'avoid'
            }}
        >
            {content}
        </li>
    );
}

const LinkListWidget = ({
    query,
    variables,
    processData,
    getHeadline,
    className,
    headerComponents,
    numColumns
}) => (
    <Query query={query} variables={variables}>
        {({ loading, data, error }) => {
            let contents;
            let headline = getHeadline();

            if (loading) {
                contents = <Loader />;
            } else if (error) {
                contents = <Message type="error" message="An error occured loading this data" />;
            } else if (data) {
                const items = processData(data);

                if (items.length === 0) {
                    return null;
                }

                headline = getHeadline(items);
                contents = (
                    <ul
                        className={`columns-${numColumns} list-reset p-3 pt-0 w-full leading-normal`}
                    >
                        {items.map(item => getLI(item))}
                    </ul>
                );
            }

            return (
                <Widget
                    className={`${className}`}
                    header={headline}
                    headerComponents={headerComponents}
                >
                    {contents}
                </Widget>
            );
        }}
    </Query>
);

LinkListWidget.propTypes = {
    query: PropTypes.shape({}).isRequired,
    variables: PropTypes.shape({}),
    processData: PropTypes.func,
    getHeadline: PropTypes.func,
    className: PropTypes.string,
    headerComponents: PropTypes.node,
    numColumns: PropTypes.number
};

LinkListWidget.defaultProps = {
    variables: null,
    processData(data) {
        return data;
    },
    getHeadline() {
        return null;
    },
    className: null,
    headerComponents: null,
    numColumns: 1
};

export default LinkListWidget;
