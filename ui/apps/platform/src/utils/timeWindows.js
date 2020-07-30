import dateFns from 'date-fns';

export default function timeWindowToDate(timeWindow) {
    if (timeWindow === 'Past hour') {
        return dateFns.subHours(new Date(), 1);
    }
    if (timeWindow === 'Past 8 hours') {
        return dateFns.subHours(new Date(), 8);
    }
    if (timeWindow === 'Past day') {
        return dateFns.subHours(new Date(), 24);
    }
    if (timeWindow === 'Past week') {
        return dateFns.subWeeks(new Date(), 1);
    }
    if (timeWindow === 'Past month') {
        return dateFns.subMonths(new Date(), 1);
    }
    if (timeWindow === 'All time') {
        // Just make the time window go back to the date it all began.....
        return new Date(2014, 11);
    }
    // Should not happen.
    return null;
}
