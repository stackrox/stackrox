import axios from './instance';

import { EventProto } from '../types/event.proto';

const eventURL = '/v1/events';

export function fetchEvents(): Promise<{
    response: { events: EventProto[] };
}> {
    return axios.get<{ events: EventProto[] }>(eventURL).then((response) => ({
        response: response.data,
    }));
}
