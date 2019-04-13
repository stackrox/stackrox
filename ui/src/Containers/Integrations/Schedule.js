import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import { formValueSelector } from 'redux-form';
import { createSelector, createStructuredSelector } from 'reselect';

const timezones = ['PST', 'EST', 'MST', 'CST', 'UTC'];
const timezoneOptions = timezones.map(o => ({ label: o, value: o }));

const getTimeOfDayOptions = () => {
    const times = ['12:00'];
    for (let i = 1; i <= 11; i += 1) {
        if (i < 10) {
            times.push(`0${i}:00`);
        } else {
            times.push(`${i}:00`);
        }
    }
    return times
        .map(x => ({ label: `${x}AM`, value: `${x}AM` }))
        .concat(times.map(x => ({ label: `${x}PM`, value: `${x}PM` })));
};

const timeOfDayOptions = getTimeOfDayOptions();

const daysOfWeek = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursdays', 'Friday', 'Saturday'];
const dayOfWeekOptions = daysOfWeek.map((i, v) => ({ label: v, value: i }));

const intervals = ['Daily', 'Weekly'];
const intervalOptions = intervals.map(x => ({ label: x, value: x }));

const Schedule = ({ formData }) => {
    const needDayOption = formData.schedule && formData.schedule.interval === 'Weekly';
    return (
        <div className="w-full">
            <ReduxSelectField
                className="w-1/3"
                name="schedule.interval"
                placeholder="Interval"
                options={intervalOptions}
            />
            {needDayOption ? (
                <ReduxSelectField
                    className="w-1/3"
                    name="schedule.weekly.dayOfWeek"
                    placeholder="Day of the week"
                    options={dayOfWeekOptions}
                />
            ) : null}
            <div className="w-full flex">
                <ReduxSelectField
                    className="w-1/3"
                    name="schedule.timeOfDay"
                    placeholder="Time of Day"
                    options={timeOfDayOptions}
                />
                <ReduxSelectField
                    className="w-1/3"
                    name="schedule.timezone"
                    placeholder="Timezone"
                    options={timezoneOptions}
                />
            </div>
        </div>
    );
};

Schedule.propTypes = {
    formData: PropTypes.shape({}).isRequired
};

const getFormFieldKeys = () => [
    'schedule.timezone',
    'schedule.interval',
    'schedule.timeOfDay',
    'schedule.weekly.day'
];

const formFieldKeys = state => formValueSelector('integrationForm')(state, ...getFormFieldKeys());

const getFormData = createSelector(
    [formFieldKeys],
    formData => formData
);

const mapStateToProps = createStructuredSelector({
    formData: getFormData
});

export default connect(mapStateToProps)(Schedule);
