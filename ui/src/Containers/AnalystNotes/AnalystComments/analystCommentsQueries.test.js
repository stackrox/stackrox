import {
    getQueriesByType,
    GET_ALERT_COMMENTS,
    GET_PROCESS_COMMENTS,
    ADD_ALERT_COMMENT,
    ADD_PROCESS_COMMENT,
    UPDATE_ALERT_COMMENT,
    UPDATE_PROCESS_COMMENT,
    REMOVE_ALERT_COMMENT,
    REMOVE_PROCESS_COMMENT
} from './analystCommentsQueries';

describe('analystCommentsQueries.getQueriesByType()', () => {
    it('should get analyst comments queries by the Violation type', () => {
        const type = 'VIOLATION';

        const { GET_COMMENTS, ADD_COMMENT, UPDATE_COMMENT, REMOVE_COMMENT } = getQueriesByType(
            type
        );

        expect(GET_COMMENTS).toEqual(GET_ALERT_COMMENTS);
        expect(ADD_COMMENT).toEqual(ADD_ALERT_COMMENT);
        expect(UPDATE_COMMENT).toEqual(UPDATE_ALERT_COMMENT);
        expect(REMOVE_COMMENT).toEqual(REMOVE_ALERT_COMMENT);
    });

    it('should get analyst comments queries by the Process type', () => {
        const type = 'PROCESS';

        const { GET_COMMENTS, ADD_COMMENT, UPDATE_COMMENT, REMOVE_COMMENT } = getQueriesByType(
            type
        );

        expect(GET_COMMENTS).toEqual(GET_PROCESS_COMMENTS);
        expect(ADD_COMMENT).toEqual(ADD_PROCESS_COMMENT);
        expect(UPDATE_COMMENT).toEqual(UPDATE_PROCESS_COMMENT);
        expect(REMOVE_COMMENT).toEqual(REMOVE_PROCESS_COMMENT);
    });

    it('should not return analyst comments queries for non-Violation/Process types', () => {
        const type = 'SHAZAM';

        const { GET_COMMENTS, ADD_COMMENT, UPDATE_COMMENT, REMOVE_COMMENT } = getQueriesByType(
            type
        );

        expect(GET_COMMENTS).toEqual(undefined);
        expect(ADD_COMMENT).toEqual(undefined);
        expect(UPDATE_COMMENT).toEqual(undefined);
        expect(REMOVE_COMMENT).toEqual(undefined);
    });
});
