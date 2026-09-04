import { group } from 'k6';
import http from 'k6/http';

export function collections(host, headers, tags) {
    group('collections', function () {
        http.batch([
            [
                'GET',
                `${host}/v1/collections?query.pagination.offset=0&query.pagination.limit=20&query.pagination.sortOption.field=${encodeURIComponent('Collection Name')}&query.pagination.sortOption.reversed=false`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/collectionscount`,
                null,
                { headers, tags },
            ],
        ]);
    });
}
