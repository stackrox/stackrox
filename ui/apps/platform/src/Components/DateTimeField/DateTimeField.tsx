import React, { ReactElement } from 'react';
import { isValid, parse, format } from 'date-fns';

type DateTimeFieldProps = {
    date?: string; // ISO 8601 formatted date
    asString?: boolean;
};

function DateTimeField({ date = '', asString = false }: DateTimeFieldProps): ReactElement {
    if (!date || !isValid(parse(date))) {
        return <span>—</span>;
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
}

export default DateTimeField;
