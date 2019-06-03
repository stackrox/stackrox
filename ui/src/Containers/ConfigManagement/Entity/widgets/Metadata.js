import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';

const Metadata = ({ keyValuePairs, counts, ...rest }) => {
    const keyValueList = keyValuePairs.map(({ key, value }) => (
        <div className="border-b border-base-300 px-4 py-2 capitalize" key={key}>
            <span className="text-base-700 font-600 mr-2">{key}:</span>
            {value}
        </div>
    ));
    const countsList = counts.map(({ value, text }) => (
        <div className="rounded border border-base-400 m-4 p-1 text-center" key={text}>
            {value} {text}
        </div>
    ));
    return (
        <Widget header="Metadata" {...rest}>
            <div className="flex w-full text-sm">
                <div className="w-1/2 border-r border-base-300">{keyValueList}</div>
                <div className="w-1/2">{countsList}</div>
            </div>
        </Widget>
    );
};

PropTypes.propTypes = {
    keyValuePairs: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired
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
