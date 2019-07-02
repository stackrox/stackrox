import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';

const Metadata = ({ keyValuePairs, counts, ...rest }) => {
    const keyValueList = keyValuePairs.map(({ key, value }) => (
        <li className="border-b border-base-300 px-4 py-2 capitalize" key={key}>
            <span className="text-base-700 font-600 mr-2">{key}:</span>
            {value}
        </li>
    ));
    const countsList = counts.map(({ value, text }) => (
        <li className="rounded border border-base-400 m-4 p-1 px-4 text-center" key={text}>
            {value} {text}
        </li>
    ));
    return (
        <Widget header="Metadata" {...rest}>
            <div className="flex w-full text-sm">
                <ul className="flex-1 list-reset border-r border-base-300">{keyValueList}</ul>
                <ul className="list-reset">{countsList}</ul>
            </div>
        </Widget>
    );
};

PropTypes.propTypes = {
    keyValuePairs: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string.isRequired,
            value: PropTypes.oneOf([PropTypes.string.isRequired, PropTypes.element.isRequired])
        })
    ),
    counts: PropTypes.arrayOf(
        PropTypes.shape({
            value: PropTypes.string.isRequired,
            text: PropTypes.string.isRequired
        })
    )
};

export default Metadata;
