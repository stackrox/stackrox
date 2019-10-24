import React from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';

const DateTimeField = ({ date }) => {
    if (!date) {
        return '—';
    }

    const datePart = format(date, 'MM/DD/YYYY');
    const timePart = format(date, 'h:mm:ssA');

    return (
        <div className="flex flex-col">
            <span>{datePart}</span>
            <span>{timePart}</span>
        </div>
    );
};

DateTimeField.propTypes = {
    date: PropTypes.string
};

DateTimeField.defaultProps = {
    date: ''
};

export default DateTimeField;
