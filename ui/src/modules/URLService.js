import React from 'react';
import qs from 'qs';
import { generatePath, Link } from 'react-router-dom';

function getURLParamNames(str) {
    const PATH_REGEXP = new RegExp(
        [
            '(\\\\.)',
            '(?:\\:(\\w+)(?:\\(((?:\\\\.|[^\\\\()])+)\\))?|\\(((?:\\\\.|[^\\\\()])+)\\))([+*?])?'
        ].join('|'),
        'g'
    );

    const paramNames = [];
    let index = 0;
    let path = '';
    let res;
    res = PATH_REGEXP.exec(str);

    while (res !== null) {
        const escaped = res[1];
        const offset = res.index;
        path += str.slice(index, offset);
        index = offset + res[0].length;

        if (escaped) {
            path += escaped[1];
        } else {
            const name = res[2];

            if (path.length) {
                path = path.slice(0, path.length - 1);
            }

            paramNames.push(name);
            res = PATH_REGEXP.exec(str);
        }
    }

    return paramNames;
}

class URLService {
    constructor(match, location) {
        this.queryParams = qs.parse(location.search, { ignoreQueryPrefix: true });
        this.urlParams = match.params;
        this.path = match.path;
        this.originalURL = match.url;
        this.urlParamNames = getURLParamNames(match.path);
    }

    setParams(params) {
        if (!params) return this;

        Object.keys(params).forEach(key => {
            if (this.urlParamNames.includes(key)) this.urlParams[key] = params[key];
            else this.queryParams[key] = params[key];
        });
        return this;
    }

    setPath(path) {
        this.replace = this.path !== path;
        this.path = path;
        this.urlParamNames = getURLParamNames(path);
        return this;
    }

    clearParams() {
        this.urlParams = {};
        this.queryParams = {};
        return this;
    }

    getParams(flat) {
        if (flat) {
            return {
                ...this.urlParams,
                ...this.queryParams
            };
        }

        return {
            ...this.urlParams,
            query: this.queryParams
        };
    }

    getLink(child) {
        const pathname = generatePath(this.path, this.urlParams);
        const replace = pathname === this.originalURL;
        const search = qs.stringify(this.queryParams, { addQueryPrefix: true });
        return (
            <Link to={{ pathname, search }} replace={replace}>
                {child}
            </Link>
        );
    }
}

export default URLService;
