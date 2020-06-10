import { scaleLinear } from 'd3-scale';

/**
 * @typedef {Object} GroupedEvents
 * @property {number} differenceInMilliseconds
 * @property {Object[]} events
 */

/**
 * @typedef {Object} Params
 * @property {Events} events - The timeline events
 * @property {number} minDomain
 * @property {number} maxDomain
 * @property {number} minRange
 * @property {number} maxRange
 * @property {number} partitionSize - The size of the segments that contain events to be grouped
 */

/**
 * @typedef {Array<GroupedEvents>>} Result
 */

/**
 * This function will group events, within a segment, based on a partition size
 * @param {Params} params
 * @returns {Result}
 */
const getGroupedEvents = ({ events, minDomain, maxDomain, minRange, maxRange, partitionSize }) => {
    if (maxRange - minRange === 0) return [];

    const numPartitions = Math.round((maxRange - minRange) / (partitionSize * 2));
    const scale = scaleLinear().domain([minDomain, maxDomain]).range([minRange, maxRange]);
    const partitionScale = scale.rangeRound([0, numPartitions]);

    // create a mapping where the key is the differenceInMilliseconds and the value is
    // the events
    const groups = events.reduce((groupsMapping, event) => {
        const newGroupsMapping = { ...groupsMapping };
        const { differenceInMilliseconds } = event;
        const groupedDifferenceInMilliseconds = scale.invert(
            partitionScale(differenceInMilliseconds)
        );
        if (newGroupsMapping[groupedDifferenceInMilliseconds]) {
            newGroupsMapping[groupedDifferenceInMilliseconds].push(event);
        } else {
            newGroupsMapping[groupedDifferenceInMilliseconds] = [event];
        }
        return newGroupsMapping;
    }, {});

    const groupedEvents = Object.keys(groups).map((differenceInMilliseconds) => {
        return {
            differenceInMilliseconds: parseInt(differenceInMilliseconds, 10),
            events: groups[differenceInMilliseconds],
        };
    });
    return groupedEvents;
};

export default getGroupedEvents;
