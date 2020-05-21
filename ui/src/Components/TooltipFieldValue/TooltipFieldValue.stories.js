import React from 'react';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipFieldValue from 'Components/TooltipFieldValue';

export default {
    title: 'TooltipFieldValue',
    component: TooltipFieldValue,
};

export const withFieldValues = () => (
    <DetailedTooltipOverlay
        title="/usr/bin/uptonogood"
        body={
            <>
                <TooltipFieldValue field="Type" value="Secret" />
                <TooltipFieldValue field="Count" value={1000} />
            </>
        }
    />
);

export const withNoValues = () => (
    <DetailedTooltipOverlay
        title="/usr/bin/uptonogood"
        body={
            <>
                <TooltipFieldValue field="Type" value={null} />
                <TooltipFieldValue field="Count" value={null} />
            </>
        }
    />
);

export const withAlertType = () => (
    <DetailedTooltipOverlay
        title="/usr/bin/uptonogood"
        body={<TooltipFieldValue field="Type" value="Alert" type="alert" />}
    />
);

export const withCautionType = () => (
    <DetailedTooltipOverlay
        title="/usr/bin/uptonogood"
        body={<TooltipFieldValue field="Type" value="Caution" type="caution" />}
    />
);

export const withWarningType = () => (
    <DetailedTooltipOverlay
        title="/usr/bin/uptonogood"
        body={<TooltipFieldValue field="Type" value="Warning" type="warning" />}
    />
);
