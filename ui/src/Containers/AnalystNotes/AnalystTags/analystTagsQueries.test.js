import {
    getTagsDataByType,
    getQueriesByType,
    GET_ALERT_TAGS,
    GET_PROCESS_TAGS,
    ADD_ALERT_TAGS,
    ADD_PROCESS_TAGS,
    REMOVE_ALERT_TAGS,
    REMOVE_PROCESS_TAGS
} from './analystTagsQueries';

describe('analystTagsQueries.getQueriesByType()', () => {
    it('should get analyst tags queries by the Violation type', () => {
        const type = 'VIOLATION';

        const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(type);

        expect(GET_TAGS).toEqual(GET_ALERT_TAGS);
        expect(ADD_TAGS).toEqual(ADD_ALERT_TAGS);
        expect(REMOVE_TAGS).toEqual(REMOVE_ALERT_TAGS);
    });

    it('should get analyst tags queries by the Process type', () => {
        const type = 'PROCESS';

        const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(type);

        expect(GET_TAGS).toEqual(GET_PROCESS_TAGS);
        expect(ADD_TAGS).toEqual(ADD_PROCESS_TAGS);
        expect(REMOVE_TAGS).toEqual(REMOVE_PROCESS_TAGS);
    });

    it('should not return analyst tags queries for non-Violation/Process types', () => {
        const type = 'SHAZAM';

        const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(type);

        expect(GET_TAGS).toEqual(undefined);
        expect(ADD_TAGS).toEqual(undefined);
        expect(REMOVE_TAGS).toEqual(undefined);
    });
});

describe('analystTagsQueries.getTagsDataByType()', () => {
    it('should get analyst tags data by the Violation type', () => {
        const type = 'VIOLATION';
        const data = {
            violation: {
                tags: ['tag-1', 'tag-2']
            }
        };

        const tags = getTagsDataByType(type, data);

        expect(tags).toEqual(data.violation.tags);
    });

    it('should get analyst tags data by the Proccess type', () => {
        const type = 'PROCESS';
        const data = {
            processTags: ['tag-1', 'tag-2']
        };

        const tags = getTagsDataByType(type, data);

        expect(tags).toEqual(data.processTags);
    });

    it('should not return analyst tags data for non-Violation/Process types', () => {
        const type = 'COVID-19';
        const data = {
            violation: {
                tags: ['tag-1', 'tag-2']
            },
            processTags: ['tag-1', 'tag-2']
        };

        const tags = getQueriesByType(type, data);

        expect(tags).toEqual({});
    });
});
