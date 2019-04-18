import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import { formValueSelector } from 'redux-form';
import { createSelector, createStructuredSelector } from 'reselect';

const getTimes = () => {
    const times = ['12:00'];
    for (let i = 1; i <= 11; i += 1) {
        if (i < 10) {
            times.push(`0${i}:00`);
        } else {
            times.push(`${i}:00`);
        }
    }
    return times.map(x => `${x}AM`).concat(times.map(x => `${x}PM`));
};

export const times = getTimes();

const timeOfDayOptions = times.map((x, i) => ({ label: `${x} UTC`, value: i }));

export const daysOfWeek = [
    'Sunday',
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday'
];

const dayOfWeekOptions = daysOfWeek.map((day, i) => ({ label: day, value: i }));

const intervalOptions = [{ label: 'Daily', value: 'DAILY' }, { label: 'Weekly', value: 'WEEKLY' }];

const normalizeValue = (value, normalizationFactor) =>
    (value + normalizationFactor) % normalizationFactor;

const getLocalTimeString = schedule => {
    if (schedule && schedule.intervalType && Number.isInteger(schedule.hour)) {
        const offsetInHours = new Date().getTimezoneOffset() / 60;
        const rawTOD = schedule.hour - offsetInHours;
        const tod = times[normalizeValue(rawTOD, 24)];
        let day = 'Daily';
        if (schedule.intervalType === 'WEEKLY') {
            if (schedule.weekly && Number.isInteger(schedule.weekly.day)) {
                if (rawTOD < 0) {
                    day = daysOfWeek[normalizeValue(schedule.weekly.day - 1, 7)];
                } else if (rawTOD > 23) {
                    day = daysOfWeek[normalizeValue(schedule.weekly.day + 1, 7)];
                } else {
                    day = daysOfWeek[normalizeValue(schedule.weekly.day, 7)];
                }
            } else {
                return '';
            }
        }
        return `${day} ${tod || ''}`;
    }
    return '';
};

const Schedule = ({ formData }) => {
    const isWeeklyInterval = formData.schedule && formData.schedule.intervalType === 'WEEKLY';
    const localTime = getLocalTimeString(formData.schedule);

    return (
        <div className="w-full">
            <ReduxSelectField
                className="w-1/3"
                name="schedule.intervalType"
                placeholder="Interval"
                options={intervalOptions}
            />
            {isWeeklyInterval && (
                <ReduxSelectField
                    className="w-1/3"
                    name="schedule.weekly.day"
                    placeholder="Day of Week"
                    options={dayOfWeekOptions}
                />
            )}
            <div className="w-full flex">
                <ReduxSelectField
                    className="w-48"
                    name="schedule.hour"
                    placeholder="Time of Day"
                    options={timeOfDayOptions}
                />
                <div className="p-3 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                    {localTime ? `Local: ${localTime}` : 'Please specify a complete schedule'}
                </div>
            </div>
        </div>
    );
};

Schedule.propTypes = {
    formData: PropTypes.shape({}).isRequired
};

const getFormFieldKeys = () => ['schedule.intervalType', 'schedule.hour', 'schedule.weekly.day'];

const formFieldKeys = state => formValueSelector('integrationForm')(state, ...getFormFieldKeys());

const getFormData = createSelector(
    [formFieldKeys],
    formData => formData
);

const mapStateToProps = createStructuredSelector({
    formData: getFormData
});

export default connect(mapStateToProps)(Schedule);
