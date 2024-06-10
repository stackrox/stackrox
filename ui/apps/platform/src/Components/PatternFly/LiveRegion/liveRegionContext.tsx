import { createContext } from 'react';

const LiveRegionContext = createContext({
    isUpdating: false,
});

export default LiveRegionContext;
