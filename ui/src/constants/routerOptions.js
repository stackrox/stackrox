import { shape } from 'prop-types';

const getRouterOptions = fn => ({
    context: {
        router: {
            history: {
                push: fn(),
                replace: fn(),
                createHref: fn()
            },
            route: {
                location: {
                    hash: '',
                    pathname: '',
                    search: '',
                    state: ''
                },
                match: {
                    params: {},
                    isExact: false,
                    path: '',
                    url: ''
                }
            }
        }
    },
    childContextTypes: {
        router: shape({
            route: shape({
                location: shape(),
                match: shape()
            }),
            history: shape({})
        })
    }
});

export default getRouterOptions;
