import { waitFor } from '@testing-library/react';

async function waitForNextUpdate(result) {
    const initialValue = result.current;
    await waitFor(() => {
        expect(result.current).not.toBe(initialValue);
    });
}

export default waitForNextUpdate;
