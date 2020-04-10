import { differenceInMilliseconds, parse } from 'date-fns';

// assumes both date's are in the ISO 8601 format (2011-08-12T20:17:46.384Z)
function getDifferenceInHours(dateLeft, dateRight) {
    return differenceInMilliseconds(parse(dateLeft), parse(dateRight)) / 3600000;
}

export default getDifferenceInHours;
