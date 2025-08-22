import { act } from '@testing-library/react';

export default async function actAndFlushTaskQueue(callback) {
    return await act(async () => {
        callback();
        await Promise.resolve(); // flush the microtask queue by creating a promise and awaiting its resolution
    });
}
