import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Select, SelectOption } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { timeWindows } from 'constants/timeWindows';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

type TimeWindowSelectorProps = {
    setActivityTimeWindow: (timeWindow) => void;
    activityTimeWindow: string;
    isDisabled?: boolean;
};

function TimeWindowSelector({
    setActivityTimeWindow,
    activityTimeWindow,
    isDisabled = false,
}: TimeWindowSelectorProps) {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();
    function selectTimeWindow(_event, selection) {
        closeSelect();
        setActivityTimeWindow(selection);
    }

    return (
        <Select
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={selectTimeWindow}
            selections={activityTimeWindow}
            isDisabled={isDisabled}
        >
            {timeWindows.map((window) => (
                <SelectOption key={window} value={window}>
                    {window}
                </SelectOption>
            ))}
        </Select>
    );
}

const mapStateToProps = createStructuredSelector({
    activityTimeWindow: selectors.getNetworkActivityTimeWindow,
});

const mapDispatchToProps = {
    setActivityTimeWindow: pageActions.setNetworkActivityTimeWindow,
};

export default connect(mapStateToProps, mapDispatchToProps)(TimeWindowSelector);
