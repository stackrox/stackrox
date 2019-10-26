import React from 'react';
import PropTypes from 'prop-types';
import { isValid, parse, format } from 'date-fns';

const DateTimeField = ({ date, asString }) => {
    if (!date || !isValid(parse(date))) {
        return '—';
    }

    const datePart = format(date, 'MM/DD/YYYY');
    const timePart = format(date, 'h:mm:ssA');

    return asString ? (
        <span>{`${datePart} | ${timePart}`}</span>
    ) : (
        <div className="flex flex-col">
            <span>{datePart}</span>
            <span>{timePart}</span>
        </div>
    );
};

DateTimeField.propTypes = {
    date: PropTypes.string,
    asString: PropTypes.bool
};

DateTimeField.defaultProps = {
    date: '',
    asString: false
};

export default DateTimeField;
