import { delay } from 'redux-saga';
import { take, fork, call } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';
import fetchSummaryCounts from 'services/SummaryService';
import { pollSummaryCounts, watchLocation } from './summarySagas';

describe('Summary Sagas Test', () => {
    it('Should call pollSummaryCounts when location changes to /main/violations', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).toEqual(fork(pollSummaryCounts));
    });

    it('Should not call pollSummaryCounts a second time when location changes from /main/violations to /main/images', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).toEqual(fork(pollSummaryCounts));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/images'
            }
        }));
        expect(value).not.toEqual(fork(pollSummaryCounts));
    });

    it('Should call pollSummaryCounts with a success every 5 seconds', () => {
        const gen = pollSummaryCounts();
        let { value } = gen.next();
        expect(value).toEqual(call(fetchSummaryCounts));
        const result = {
            response: 'POLL_TEST'
        };
        ({ value } = gen.next(result));
        expect(value).toEqual({
            '@@redux-saga/IO': true,
            PUT: {
                action: {
                    params: undefined,
                    response: 'POLL_TEST',
                    type: 'summaries/FETCH_SUMMARY_COUNTS_SUCCESS'
                },
                channel: null
            }
        });
        ({ value } = gen.next());
        expect(value).toEqual(call(delay, 30000));
    });

    it('Should call pollSummaryCounts with a fail', () => {
        const error = new Error('POLL_ERROR');
        const gen = pollSummaryCounts();
        let { value } = gen.next();
        expect(value).toEqual(call(fetchSummaryCounts));
        ({ value } = gen.throw(error));
        expect(value).toEqual({
            '@@redux-saga/IO': true,
            PUT: {
                action: {
                    params: undefined,
                    error,
                    type: 'summaries/FETCH_SUMMARY_COUNTS_FAILURE'
                },
                channel: null
            }
        });
    });
});
