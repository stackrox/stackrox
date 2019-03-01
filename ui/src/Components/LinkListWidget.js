import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';
import Message from 'Components/Message';
import Query from 'Components/ThrowingQuery';
import NoResultsMessage from 'Components/NoResultsMessage';

function getLI(item) {
    if (!item) return null;

    const content = item.link ? (
        <Link
            to={item.link}
            title={item.label}
            className="font-600 text-base-600 hover:bg-primary-100 focus:bg-primary-100 focus:text-primary-700 hover:text-primary-700 leading-normal px-2 inline-block w-full h-8 items-center flex"
        >
            <span className="truncate w-full">{item.label}</span>
        </Link>
    ) : (
        item.label
    );
    return (
        <li
            key={item.label}
            className="border-b border-base-300"
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
    limit,
    id,
    showEmpty
}) => (
    <Query query={query} variables={variables}>
        {({ loading, data, error }) => {
            let contents;
            let headline = getHeadline();

            if (loading) {
                contents = <Loader />;
            } else if (error) {
                contents = <Message type="error" message="An error occurred loading this data" />;
            } else if (data) {
                const items = processData(data);
                headline = getHeadline(items);

                if (items.length === 0) {
                    if (!showEmpty) {
                        return null;
                    }
                    contents = <NoResultsMessage message="No data matched your search" />;
                } else {
                    contents = (
                        <ul
                            className={`${
                                items.length > 5 ? `columns-2` : `columns-1`
                            } list-reset p-3 py-1 w-full leading-normal overflow-hidden`}
                        >
                            {items.slice(0, limit).map(item => getLI(item))}
                        </ul>
                    );
                }
            }

            return (
                <Widget
                    className={`${className}`}
                    header={headline}
                    headerComponents={headerComponents}
                    id={id}
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
    limit: PropTypes.number,
    showEmpty: PropTypes.bool,
    id: PropTypes.string
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
    limit: 10,
    showEmpty: false,
    id: 'link-list-widget'
};

export default LinkListWidget;
